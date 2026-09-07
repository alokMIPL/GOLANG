// Package config is placed under an "internal/" directory.
//
// Go enforces a special rule for any path containing a directory named
// "internal": it can ONLY be imported by code rooted at the parent of
// that "internal" directory. Here that means only code inside
// "go-packages-demo/..." can import "go-packages-demo/internal/config" —
// an external module could never import it, even though Config and
// Load are capitalized/exported. This is Go's built-in way to expose
// an API within your own module/organization while hiding it from
// outside consumers, without needing a separate private repo.
package config

// Config holds app-wide settings.
type Config struct {
	Env     string
	Debug   bool
	Version string
}

// Load returns a Config. In a real app this might read env vars or a file.
func Load() Config {
	return Config{
		Env:     "development",
		Debug:   true,
		Version: "1.0.0",
	}
}
