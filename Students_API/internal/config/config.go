package config

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/ilyakaznacheev/cleanenv"
)

// go get -u github.com/ilyakaznacheev/cleanenv

// go mod tidy

const (
	// configPathEnv is the environment variable checked first for the path
	// of the config file.
	configPathEnv = "CONFIG_PATH"

	// configFlagName is the command-line flag checked when the environment
	// variable is empty, e.g. `go run cmd/students-api/main.go -config config/local.yml`
	configFlagName = "config"

	// Allowed values for Config.Env.
	EnvLocal = "local"
	EnvDev   = "dev"
	EnvProd  = "prod"
)

// HTTPServer holds the settings for the HTTP server.
type HTTPServer struct {
	// Addr is the listen address, e.g. "localhost:8082".
	// NOTE: the original tag was `env-required: "true"` (with a space), which
	// is a malformed struct tag and is silently ignored by reflection.
	Addr string `yaml:"address" env-required:"true"`
}

// Config is the top-level application configuration.
type Config struct {
	Env         string `yaml:"env" env:"ENV" env-required:"true"`
	StoragePath string `yaml:"storage_path" env-required:"true"`
	HTTPServer  `yaml:"http_server"`
}

// Validate checks the values that struct tags cannot express, such as the
// set of allowed environments. It returns every problem it finds, joined
// into one error, instead of stopping at the first.
func (c *Config) Validate() error {
	var errs []error

	switch c.Env {
	case EnvLocal, EnvDev, EnvProd:
	default:
		errs = append(errs, fmt.Errorf(
			"env must be one of %q, %q or %q, got %q",
			EnvLocal, EnvDev, EnvProd, c.Env))
	}

	if strings.TrimSpace(c.StoragePath) == "" {
		errs = append(errs, errors.New("storage_path must not be blank"))
	}

	if strings.TrimSpace(c.Addr) == "" {
		errs = append(errs, errors.New("http_server.address must not be blank"))
	}

	return errors.Join(errs...)
}

// IsProd reports whether the app is running in the production environment.
func (c *Config) IsProd() bool { return c.Env == EnvProd }

// pathFromFlag reads the config path from the -config command-line flag.
//
// flag.Parse reads os.Args[1:] (the arguments given when you ran the binary,
// excluding the program name at index 0), matches them against all registered
// flags and writes the matched values into the memory the flags point to.
//
// The flag is only registered once, so calling MustLoad or Load a second time
// (for example from tests) does not panic with "flag redefined".
func pathFromFlag() string {
	f := flag.Lookup(configFlagName)
	if f == nil {
		flag.String(configFlagName, "", "path to the configuration file")
		f = flag.Lookup(configFlagName)
	}

	if !flag.Parsed() {
		flag.Parse()
	}

	return f.Value.String()
}

// resolvePath finds the config file path. Sources are tried in order:
//
//  1. the CONFIG_PATH environment variable
//  2. the -config command-line flag
//
// An error is returned if neither is set.
func resolvePath() (string, error) {
	// 1. Environment variable.
	if p := strings.TrimSpace(os.Getenv(configPathEnv)); p != "" {
		return p, nil
	}

	// 2. Command-line flag, only checked when the env variable is empty.
	if p := strings.TrimSpace(pathFromFlag()); p != "" {
		return p, nil
	}

	// 3. Nothing found, so we have no way to locate the config file.
	return "", fmt.Errorf(
		"config path is not set: use the %s environment variable or the -%s flag",
		configPathEnv, configFlagName)
}

// checkFile verifies that path exists and is a regular file, not a directory.
func checkFile(path string) error {
	info, err := os.Stat(path)
	switch {
	case errors.Is(err, os.ErrNotExist):
		return fmt.Errorf("config file does not exist: %s", path)
	case err != nil:
		return fmt.Errorf("cannot access config file %s: %w", path, err)
	case info.IsDir():
		return fmt.Errorf("config path is a directory, not a file: %s", path)
	}
	return nil
}

// Load locates, reads and validates the configuration, returning an error
// instead of exiting. Prefer it over MustLoad in tests and in code that wants
// to handle failures itself.
func Load() (*Config, error) {
	configPath, err := resolvePath()
	if err != nil {
		return nil, err
	}

	if err := checkFile(configPath); err != nil {
		return nil, err
	}

	var cfg Config
	if err := cleanenv.ReadConfig(configPath, &cfg); err != nil {
		return nil, fmt.Errorf("cannot read config file %s: %w", configPath, err)
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config in %s: %w", configPath, err)
	}

	return &cfg, nil
}

// MustLoad is like Load but terminates the program with a fatal log message
// if the configuration cannot be loaded. It is meant to be called once at
// startup, where there is nothing useful to do without a valid config.
//
// log.Fatalf is like fmt.Printf + log.Fatal combined: it formats the message,
// prints it, and exits with status code 1.
func MustLoad() *Config {
	cfg, err := Load()
	if err != nil {
		log.Fatalf("Config error: %s", err.Error())
	}
	return cfg
}
