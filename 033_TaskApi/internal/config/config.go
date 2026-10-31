package config

import "os"

type Config struct {
	Port   string
	DBPath string // empty means use the in-memory store
}

func Load() Config {
	return Config{
		Port:   getenv("PORT", "8080"),
		DBPath: getenv("DB_PATH", ""),
	}
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
