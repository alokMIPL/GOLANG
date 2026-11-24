// Package config loads and validates application configuration from the
// environment. It never reads os.Getenv directly: callers inject a getenv
// function, which keeps Load deterministic and trivial to test.
package config

import (
	"errors"
	"fmt"
	"io"
	"log"
	"log/slog"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// Environment identifies the deployment tier.
type Environment string

const (
	EnvDevelopment Environment = "development"
	EnvStaging     Environment = "staging"
	EnvProduction  Environment = "production"
)

// LogFormat selects the slog handler.
type LogFormat string

const (
	LogText LogFormat = "text"
	LogJSON LogFormat = "json"
)

type Config struct {
	Env Environment

	// HTTP server.
	Addr              string
	ReadHeaderTimeout time.Duration
	ReadTimeout       time.Duration
	WriteTimeout      time.Duration
	IdleTimeout       time.Duration
	MaxHeaderBytes    int
	MaxBodyBytes      int64

	// Shutdown.
	ShutdownDelay   time.Duration // wait after SIGTERM so load balancers stop routing
	ShutdownTimeout time.Duration // max time to drain in-flight requests

	// Logging.
	LogLevel  slog.Level
	LogFormat LogFormat

	// CORS.
	CORSAllowedOrigins []string
}

// Load reads configuration through getenv, applies defaults, and validates
// the result. All problems are reported together via errors.Join.
func Load(getenv func(string) string) (Config, error) {
	l := &loader{getenv: getenv}

	env := Environment(l.oneOf("APP_ENV", string(EnvDevelopment),
		string(EnvDevelopment), string(EnvStaging), string(EnvProduction)))

	// Production defaults to JSON logs; everything else to human-readable text.
	defaultFormat := LogText
	if env == EnvProduction {
		defaultFormat = LogJSON
	}

	cfg := Config{
		Env: env,

		Addr:              net.JoinHostPort(l.str("HOST", ""), l.port("PORT", 8080)),
		ReadHeaderTimeout: l.duration("READ_HEADER_TIMEOUT", 2*time.Second),
		ReadTimeout:       l.duration("READ_TIMEOUT", 5*time.Second),
		WriteTimeout:      l.duration("WRITE_TIMEOUT", 10*time.Second),
		IdleTimeout:       l.duration("IDLE_TIMEOUT", 60*time.Second),
		MaxHeaderBytes:    int(l.int64("MAX_HEADER_BYTES", 1<<20, 1<<10, 16<<20)),
		MaxBodyBytes:      l.int64("MAX_BODY_BYTES", 1<<20, 1, 1<<30),

		ShutdownDelay:   l.optDuration("SHUTDOWN_DELAY", 0),
		ShutdownTimeout: l.duration("SHUTDOWN_TIMEOUT", 15*time.Second),

		LogLevel:  l.logLevel("LOG_LEVEL", slog.LevelInfo),
		LogFormat: LogFormat(l.oneOf("LOG_FORMAT", string(defaultFormat), string(LogText), string(LogJSON))),

		CORSAllowedOrigins: l.list("CORS_ALLOWED_ORIGINS"),
	}

	l.errs = append(l.errs, cfg.validate()...)
	return cfg, errors.Join(l.errs...)
}

// validate enforces rules that span several fields.
func (c Config) validate() []error {
	var errs []error

	if c.ReadHeaderTimeout > c.ReadTimeout {
		errs = append(errs, fmt.Errorf(
			"READ_HEADER_TIMEOUT (%s) must not exceed READ_TIMEOUT (%s)",
			c.ReadHeaderTimeout, c.ReadTimeout))
	}

	// If draining is shorter than a request may legitimately run, shutdown
	// would cut off requests that were still within their write deadline.
	if c.ShutdownTimeout < c.WriteTimeout {
		errs = append(errs, fmt.Errorf(
			"SHUTDOWN_TIMEOUT (%s) must be at least WRITE_TIMEOUT (%s)",
			c.ShutdownTimeout, c.WriteTimeout))
	}

	if c.Env == EnvProduction {
		for _, o := range c.CORSAllowedOrigins {
			if o == "*" {
				errs = append(errs, errors.New(`CORS_ALLOWED_ORIGINS must not contain "*" in production`))
				break
			}
		}
	}

	return errs
}

// IsProduction reports whether the service runs in production.
func (c Config) IsProduction() bool { return c.Env == EnvProduction }

// NewLogger builds a slog.Logger matching the configured level and format.
func (c Config) NewLogger(w io.Writer) *slog.Logger {
	opts := &slog.HandlerOptions{Level: c.LogLevel}

	var h slog.Handler
	if c.LogFormat == LogJSON {
		h = slog.NewJSONHandler(w, opts)
	} else {
		h = slog.NewTextHandler(w, opts)
	}
	return slog.New(h).With("env", string(c.Env))
}

// NewServer returns an http.Server with all configured limits applied.
func (c Config) NewServer(h http.Handler, errorLog *log.Logger) *http.Server {
	return &http.Server{
		Addr:              c.Addr,
		Handler:           h,
		ReadHeaderTimeout: c.ReadHeaderTimeout,
		ReadTimeout:       c.ReadTimeout,
		WriteTimeout:      c.WriteTimeout,
		IdleTimeout:       c.IdleTimeout,
		MaxHeaderBytes:    c.MaxHeaderBytes,
		ErrorLog:          errorLog,
	}
}

// LogValue implements slog.LogValuer so the config can be logged safely with
// logger.Info("config", "config", cfg). Add redaction here if secrets are
// ever introduced (database URLs, API keys, and so on).
func (c Config) LogValue() slog.Value {
	return slog.GroupValue(
		slog.String("env", string(c.Env)),
		slog.String("addr", c.Addr),
		slog.Duration("read_header_timeout", c.ReadHeaderTimeout),
		slog.Duration("read_timeout", c.ReadTimeout),
		slog.Duration("write_timeout", c.WriteTimeout),
		slog.Duration("idle_timeout", c.IdleTimeout),
		slog.Int("max_header_bytes", c.MaxHeaderBytes),
		slog.Int64("max_body_bytes", c.MaxBodyBytes),
		slog.Duration("shutdown_delay", c.ShutdownDelay),
		slog.Duration("shutdown_timeout", c.ShutdownTimeout),
		slog.String("log_level", c.LogLevel.String()),
		slog.String("log_format", string(c.LogFormat)),
		slog.Any("cors_allowed_origins", c.CORSAllowedOrigins),
	)
}

// ---------------------------------------------------------------------------
// loader: typed env readers that collect errors instead of failing on the first.
// ---------------------------------------------------------------------------

type loader struct {
	getenv func(string) string
	errs   []error
}

func (l *loader) fail(key, format string, args ...any) {
	l.errs = append(l.errs, fmt.Errorf("%s: %s", key, fmt.Sprintf(format, args...)))
}

// str returns the trimmed value of key, or def when unset or blank.
func (l *loader) str(key, def string) string {
	if v := strings.TrimSpace(l.getenv(key)); v != "" {
		return v
	}
	return def
}

// port returns a TCP port as a string. Port 0 is allowed so tests can ask
// the OS for a free port.
func (l *loader) port(key string, def uint16) string {
	raw := l.str(key, "")
	if raw == "" {
		return strconv.Itoa(int(def))
	}
	n, err := strconv.ParseUint(raw, 10, 16)
	if err != nil {
		l.fail(key, "%q must be a number between 0 and 65535", raw)
		return strconv.Itoa(int(def))
	}
	return strconv.FormatUint(n, 10)
}

// duration parses a strictly positive time.Duration such as "5s" or "1m30s".
func (l *loader) duration(key string, def time.Duration) time.Duration {
	return l.parseDuration(key, def, false)
}

// optDuration is like duration but also accepts zero.
func (l *loader) optDuration(key string, def time.Duration) time.Duration {
	return l.parseDuration(key, def, true)
}

func (l *loader) parseDuration(key string, def time.Duration, allowZero bool) time.Duration {
	raw := l.str(key, "")
	if raw == "" {
		return def
	}
	d, err := time.ParseDuration(raw)
	if err != nil {
		l.fail(key, "%q is not a valid duration (examples: 500ms, 5s, 1m30s)", raw)
		return def
	}
	if d < 0 || (d == 0 && !allowZero) {
		l.fail(key, "%q must be positive", raw)
		return def
	}
	return d
}

// int64 parses an integer and checks it lies within [min, max].
func (l *loader) int64(key string, def, min, max int64) int64 {
	raw := l.str(key, "")
	if raw == "" {
		return def
	}
	n, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		l.fail(key, "%q is not a valid integer", raw)
		return def
	}
	if n < min || n > max {
		l.fail(key, "%d is out of range (%d to %d)", n, min, max)
		return def
	}
	return n
}

// oneOf returns the value of key if it is in allowed (case-insensitive).
func (l *loader) oneOf(key, def string, allowed ...string) string {
	raw := strings.ToLower(l.str(key, ""))
	if raw == "" {
		return def
	}
	for _, a := range allowed {
		if raw == a {
			return a
		}
	}
	l.fail(key, "%q must be one of: %s", raw, strings.Join(allowed, ", "))
	return def
}

// list splits a comma-separated value, trimming blanks and dropping empties.
func (l *loader) list(key string) []string {
	var out []string
	for _, part := range strings.Split(l.getenv(key), ",") {
		if p := strings.TrimSpace(part); p != "" {
			out = append(out, p)
		}
	}
	return out
}

// logLevel parses debug, info, warn, or error (and offsets like "info+2").
func (l *loader) logLevel(key string, def slog.Level) slog.Level {
	raw := l.str(key, "")
	if raw == "" {
		return def
	}
	var lvl slog.Level
	if err := lvl.UnmarshalText([]byte(raw)); err != nil {
		l.fail(key, "%q must be one of: debug, info, warn, error", raw)
		return def
	}
	return lvl
}
