package main

import (
	"log"
	"notes_api/internal/config"
)

// Now in tihs main.go we wire all the things means configure and connect each and every services.
// config -> db -> router -> run server

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Config error")
	}

	// Now DB connection

}
