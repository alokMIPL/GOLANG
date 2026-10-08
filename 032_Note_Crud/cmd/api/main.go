package main

import (
	"fmt"
	"log"
	"notes_api/internal/config"
	"notes_api/internal/db"
	"notes_api/internal/server"
)

// Now in tihs main.go we wire all the things means configure and connect each and every services.
// config -> db -> router -> run server

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Config error")
	}

	// Now DB connection

	client, _, err := db.Connect(cfg)
	if err != nil {
		log.Fatalf("Config error")
	}

	defer func() {
		if err := db.Disconnect(client); err != nil {
			log.Printf("mongo disconnect error: %v", err)
		}
	}()

	router := server.NewRouter()
	addr := fmt.Sprintf(":%s", cfg.ServerPort)

	if err := router.Run(addr); err != nil {
		log.Fatalf("Server failed")
	}
}
