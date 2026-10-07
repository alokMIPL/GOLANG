package config

import (
	"github.com/joho/godotenv"
)

type Config struct {
	MongoURI   string
	MongoDB    string
	ServerPort string
}

// This Load Function load and validate every env value.
func Load() (Config, error) {

	// Imp
	// godotenv.Load() reads .env and set them into the process env
	// os.getenv -> reads those values

	if err := godotenv.Load(); err != nil {

	}
}
