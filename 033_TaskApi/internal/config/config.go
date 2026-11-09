package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// Default values used when an environment variable is not set.
const (
	defaultPort            = "8080"
	defaultEnv             = "development"
	defaultLogLevel        = "info"
	defaultReadTimeout     = 5 * time.Second
	defaultWriteTimeout    = 10 * time.Second
	defaultShutdownTimeout = 5 * time.Second
)

// Config holds all runtime settings for the application.
type Config struct {
	Port            string
	DBPath          string // empty means use the in-memory store
	Env             string // development, staging or production
	LogLevel        string // debug, info, warn or error
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	ShutdownTimeout time.Duration
	AllowedOrigins  []string // CORS origins, empty means none
	Debug           bool
}

// Load reads configuration from environment variables, applying defaults
// for anything that is missing. It never fails; use Validate to check the
// result.
func Load() Config {
	return Config{
		Port:            getenv("PORT", defaultPort),
		DBPath:          getenv("DB_PATH", ""),
		Env:             getenv("APP_ENV", defaultEnv),
		LogLevel:        strings.ToLower(getenv("LOG_LEVEL", defaultLogLevel)),
		ReadTimeout:     getDuration("READ_TIMEOUT", defaultReadTimeout),
		WriteTimeout:    getDuration("WRITE_TIMEOUT", defaultWriteTimeout),
		ShutdownTimeout: getDuration("SHUTDOWN_TIMEOUT", defaultShutdownTimeout),
		AllowedOrigins:  getList("ALLOWED_ORIGINS"),
		Debug:           getBool("DEBUG", false),
	}
}

// MustLoad loads the configuration and panics if it is invalid.
// Handy for tests and small programs.
func MustLoad() Config {
	cfg := Load()
	if err := cfg.Validate(); err != nil {
		panic(fmt.Sprintf("invalid config: %v", err))
	}
	return cfg
}

// Validate checks that the configuration values make sense.
func (c Config) Validate() error {
	var errs []error

	port, err := strconv.Atoi(c.Port)
	if err != nil {
		errs = append(errs, fmt.Errorf("PORT %q is not a number", c.Port))
	} else if port < 1 || port > 65535 {
		errs = append(errs, fmt.Errorf("PORT %d is out of range (1-6535)", port))
	}

	switch c.Env {
	case "development", "staging", "production":
	default:
		errs = append(errs, fmt.Errorf("APP_ENV %q must be development, staging or production", c.Env))
	}

	switch c.LogLevel {
	case "debug", "info", "warn", "error":
	default:
		errs = append(errs, fmt.Errorf("LOG_LEVEL %q must be debug, info, warn or error", c.LogLevel))
	}

	if c.ReadTimeout <= 0 {
		errs = append(errs, errors.New("READ_TIMEOUT must be positive"))
	}
	if c.WriteTimeout <= 0 {
		errs = append(errs, errors.New("WRITE_TIMEOUT must be positive"))
	}
	if c.ShutdownTimeout <= 0 {
		errs = append(errs, errors.New("SHUTDOWN_TIMEOUT must be positive"))
	}

	return errors.Join(errs...)
}

// UseMemoryStore reports whether the app should use the in-memory store.
func (c Config) UseMemoryStore() bool {
	return c.DBPath == ""
}

// IsProduction reports whether the app is running in production.
func (c Config) IsProduction() bool {
	return c.Env == "production"
}

// Addr returns the address the HTTP server should listen on.
func (c Config) Addr() string {
	return ":" + c.Port
}

// String returns a readable summary of the config, safe to log.
func (c Config) String() string {
	store := "memory"
	if !c.UseMemoryStore() {
		store = c.DBPath
	}
	return fmt.Sprintf(
		"env=%s port=%s store=%s log_level=%s debug=%t",
		c.Env, c.Port, store, c.LogLevel, c.Debug,
	)
}

// getenv returns the value of an environment variable, or fallback if the
// variable is unset or empty.
func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// getBool parses a boolean environment variable such as "true" or "1".
// Invalid values fall back to the default.
func getBool(key string, fallback bool) bool {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return fallback
	}
	return b
}

// getDuration parses a duration such as "5s" or "2m". A plain number is
// treated as seconds. Invalid values fall back to the default.
func getDuration(key string, fallback time.Duration) time.Duration {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	if d, err := time.ParseDuration(v); err == nil {
		return d
	}
	if secs, err := strconv.Atoi(v); err == nil {
		return time.Duration(secs) * time.Second
	}
	return fallback
}

// getList parses a comma separated environment variable into a slice,
// trimming spaces and dropping empty entries.
func getList(key string) []string {
	v := os.Getenv(key)
	if v == "" {
		return nil
	}
	var out []string
	for _, part := range strings.Split(v, ",") {
		if part = strings.TrimSpace(part); part != "" {
			out = append(out, part)
		}
	}
	return out
}
