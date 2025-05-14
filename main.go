package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/45uperman/gator/internal/config"
	"github.com/45uperman/gator/internal/database"
	"github.com/45uperman/gator/internal/feed"
	"github.com/google/uuid"
	_ "github.com/lib/pq"
)

type state struct {
	db  *database.Queries
	cfg *config.Config
}

type command struct {
	name string
	args []string
}

type commands struct {
	supportedCommands map[string]func(*state, command) error
}

func (c *commands) run(s *state, cmd command) error {
	callback, ok := c.supportedCommands[cmd.name]
	if !ok {
		return fmt.Errorf("invalid command: %s", cmd.name)
	}

	err := callback(s, cmd)
	if err != nil {
		return err
	}

	return nil
}

func (c *commands) register(name string, f func(*state, command) error) {
	c.supportedCommands[name] = f
}

func main() {
	var s state
	{
		cfg, err := config.Read()
		if err != nil {
			log.Fatal(err)
		}

		s.cfg = &cfg

		db, err := sql.Open("postgres", s.cfg.DBURL)
		if err != nil {
			log.Fatal(err)
		}

		dbQueries := database.New(db)

		s.db = dbQueries
	}

	c := commands{supportedCommands: make(map[string]func(*state, command) error)}

	c.register("login", handlerLogin)
	c.register("register", handlerRegister)
	c.register("reset", handlerReset)
	c.register("users", handlerUsers)
	c.register("agg", handlerAgg)
	c.register("addfeed", middlewareLoggedIn(handlerAddFeed))
	c.register("feeds", handlerFeeds)
	c.register("follow", middlewareLoggedIn(handlerFollow))
	c.register("following", middlewareLoggedIn(handlerFollowing))

	if len(os.Args) < 2 {
		log.Fatal("error: no command given")
	}

	cmd := command{
		name: os.Args[1],
		args: make([]string, 0),
	}

	if len(os.Args) > 2 {
		cmd.args = os.Args[2:]
	}

	err := c.run(&s, cmd)
	if err != nil {
		log.Fatal(err)
	}
}

func handlerLogin(s *state, cmd command) error {
	if len(cmd.args) == 0 {
		return fmt.Errorf("login requires a username to log in as")
	}

	_, err := s.db.GetUser(context.Background(), cmd.args[0])
	if err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("cannot log in as user with name %s because that user does not exist", cmd.args[0])
		}
		return err
	}

	err = s.cfg.SetUser(cmd.args[0])
	if err != nil {
		return err
	}

	fmt.Printf("Logged in as user: %s\n", cmd.args[0])

	return nil
}

func handlerRegister(s *state, cmd command) error {
	if len(cmd.args) == 0 {
		return fmt.Errorf("register requires a username to register")
	}

	_, err := s.db.GetUser(context.Background(), cmd.args[0])
	if err != nil {
		if err != sql.ErrNoRows {
			return err
		}
	} else {
		return fmt.Errorf("cannot create user with name %s because that user already exists", cmd.args[0])
	}

	u, err := s.db.CreateUser(
		context.Background(),
		database.CreateUserParams{
			ID:        uuid.New(),
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
			Name:      cmd.args[0],
		},
	)
	if err != nil {
		return err
	}

	s.cfg.SetUser(cmd.args[0])
	fmt.Printf("\nsuccessfully created user: %s\n\n", cmd.args[0])

	fmt.Printf(
		"  ID: %v\n  CreatedAt: %v\n  UpdatedAt: %v\n  Name: %v\n\n",
		u.ID,
		u.CreatedAt,
		u.UpdatedAt,
		u.Name,
	)

	return nil
}

func handlerReset(s *state, cmd command) error {
	err := s.db.ResetUsers(context.Background())
	if err != nil {
		return err
	}

	return nil
}

func handlerUsers(s *state, cmd command) error {
	users, err := s.db.GetUsers(context.Background())
	if err != nil {
		return err
	}

	for _, user := range users {
		fmt.Printf("* %s", user.Name)
		if user.Name == s.cfg.CurrentUserName {
			fmt.Print(" (current)")
		}
		fmt.Print("\n")
	}

	return nil
}

func handlerAgg(s *state, cmd command) error {
	newFeed, err := feed.FetchFeed(context.Background(), "https://www.wagslane.dev/index.xml")
	if err != nil {
		return err
	}
	newFeed.Unescape()

	fmt.Println(*newFeed)

	return nil
}

func handlerAddFeed(s *state, cmd command, user database.User) error {
	if len(cmd.args) < 2 {
		return fmt.Errorf("addfeed requires the name and url of the feed to be added as arguments")
	}

	f, err := s.db.CreateFeed(
		context.Background(),
		database.CreateFeedParams{
			ID:        uuid.New(),
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
			Name:      cmd.args[0],
			Url:       cmd.args[1],
			UserID:    user.ID,
		},
	)
	if err != nil {
		return err
	}

	_, err = s.db.CreateFeedFollow(
		context.Background(),
		database.CreateFeedFollowParams{
			ID:        uuid.New(),
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
			UserID:    user.ID,
			FeedID:    f.ID,
		},
	)
	if err != nil {
		return err
	}

	fmt.Printf("\nsuccessfully added feed: %s\n\n", f.Name)
	fmt.Printf(
		"  ID: %v\n  CreatedAt: %v\n  UpdatedAt: %v\n  Name: %v\n  Url: %v\n  UserId: %v\n\n",
		f.ID,
		f.CreatedAt,
		f.UpdatedAt,
		f.Name,
		f.Url,
		f.UserID,
	)

	return nil
}

func handlerFeeds(s *state, cmd command) error {
	feeds, err := s.db.GetFeeds(context.Background())
	if err != nil {
		return err
	}

	for _, f := range feeds {
		feedOwner, err := s.db.GetUserByID(context.Background(), f.UserID)
		if err != nil {
			return err
		}

		fmt.Printf("\nFeed '%s' added by user '%s'\n\n", f.Name, feedOwner.Name)
		fmt.Printf(
			"  ID: %v\n  CreatedAt: %v\n  UpdatedAt: %v\n  Name: %v\n  Url: %v\n  UserId: %v\n\n",
			f.ID,
			f.CreatedAt,
			f.UpdatedAt,
			f.Name,
			f.Url,
			f.UserID,
		)
	}

	return nil
}

func handlerFollow(s *state, cmd command, user database.User) error {
	if len(cmd.args) < 1 {
		return fmt.Errorf("follow requires the url of the feed to be followed as an argument")
	}

	f, err := s.db.GetFeedByURL(
		context.Background(),
		cmd.args[0],
	)
	if err != nil {
		return err
	}

	record, err := s.db.CreateFeedFollow(
		context.Background(),
		database.CreateFeedFollowParams{
			ID:        uuid.New(),
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
			UserID:    user.ID,
			FeedID:    f.ID,
		},
	)
	if err != nil {
		return err
	}

	fmt.Printf("Feed '%s' followed by user '%s'\n", record.FeedName, record.UserName)

	return nil
}

func handlerFollowing(s *state, cmd command, user database.User) error {
	follows, err := s.db.GetFeedFollowsForUser(
		context.Background(),
		user.Name,
	)
	if err != nil {
		return err
	}

	for _, feed := range follows {
		fmt.Println(feed.FeedName)
	}

	return nil
}

func middlewareLoggedIn(handler func(s *state, cmd command, user database.User) error) func(*state, command) error {
	return func(s *state, cmd command) error {
		current_user, err := s.db.GetUser(
			context.Background(),
			s.cfg.CurrentUserName,
		)
		if err != nil {
			return err
		}

		return handler(s, cmd, current_user)
	}
}
