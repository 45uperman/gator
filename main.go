package main

import (
	"fmt"
	"log"
	"os"

	"github.com/45uperman/gator/internal/config"
)

type state struct {
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
	}

	c := commands{supportedCommands: make(map[string]func(*state, command) error)}

	c.register("login", handlerLogin)

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
		return fmt.Errorf("login requires a username")
	}

	err := s.cfg.SetUser(cmd.args[0])
	if err != nil {
		return err
	}

	fmt.Printf("Logged in as user: %s\n", cmd.args[0])

	return nil
}
