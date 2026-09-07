package config

// Config holds app-wide settings.
type Config struct {
	Env     string
	Debug   bool
	Version string
}

func Load() Config {
	return Config{
		Env:     "development",
		Debug:   true,
		Version: "1.0.0",
	}
}
