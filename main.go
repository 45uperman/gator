package main

import (
	"fmt"
	"log"

	"github.com/45uperman/gator/internal/config"
)

func main() {
	cfg, err := config.Read()
	if err != nil {
		log.Fatal(err)
	}

	cfg.SetUser("Superman")

	cfg, err = config.Read()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("db_url: %s\ncurrent_user_name: %s\n", cfg.DBURL, cfg.CurrentUserName)
}
