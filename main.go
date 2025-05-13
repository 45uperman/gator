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

	username := sql.NullString{String: cmd.args[0], Valid: true}

	_, err := s.db.GetUser(context.Background(), username)
	if err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("cannot log in as user with name %s because that user does not exist", username.String)
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

	username := sql.NullString{String: cmd.args[0], Valid: true}

	_, err := s.db.GetUser(context.Background(), username)
	if err != nil {
		if err != sql.ErrNoRows {
			return err
		}
	} else {
		return fmt.Errorf("cannot create user with name %s because that user already exists", username.String)
	}

	u, err := s.db.CreateUser(
		context.Background(),
		database.CreateUserParams{
			ID:        uuid.New(),
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
			Name:      username,
		},
	)
	if err != nil {
		return err
	}

	s.cfg.SetUser(username.String)
	fmt.Printf("successfully created user: %s\n", username.String)

	fmt.Printf(
		"ID: %v\n  CreatedAt: %v\n  UpdatedAt: %v\n  Name: %v\n",
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
