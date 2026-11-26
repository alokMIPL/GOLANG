// Package config loads and validates application configuration from the
// environment. It never reads os.Getenv directly: callers inject a getenv
// function, which keeps Load deterministic and trivial to test.
//
// The package is deliberately pure data: it does not construct loggers or
// servers (see package app for that).
package config

import (
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/url"
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

// maxShutdownDelay guards against typos such as "5m" instead of "5s", which
// would silently slow every deploy.
const maxShutdownDelay = time.Minute

// Config is the validated application configuration. A Config returned
// alongside a non-nil error is always the zero value.
type Config struct {
	Env Environment

	// HTTP server.
	Addr              string
	ReadHeaderTimeout time.Duration
	ReadTimeout       time.Duration
	WriteTimeout      time.Duration
	IdleTimeout       time.Duration
	MaxHeaderBytes    int
	// MaxBodyBytes is not enforced by the server itself; handlers or
	// middleware should apply it with http.MaxBytesReader.
	MaxBodyBytes int64

	// Shutdown.
	ShutdownDelay      time.Duration // wait after SIGTERM so load balancers stop routing
	ShutdownTimeout    time.Duration // max time to drain in-flight requests
	LogLevel           slog.Level
	LogFormat          LogFormat
	CORSAllowedOrigins []string
}

func Load(getenv func(string) string) (Config, error) {
	if getenv == nil {
		return Config{}, errors.New("config: getenv must not be nil")
	}
	l := &loader{getenv: getenv}

	env := Environment(l.requiredOneOf("APP_ENV",
		string(EnvDevelopment), string(EnvStaging), string(EnvProduction)))

	// Production defaults to JSON logs; everything else to readable text.
	defaultFormat := LogText
	if env == EnvProduction {
		defaultFormat = LogJSON
	}

	// Bind to loopback in development; all interfaces elsewhere.
	defaultHost := ""
	if env == EnvDevelopment {
		defaultHost = "localhost"
	}

	cfg := Config{
		Env: env,

		Addr:              net.JoinHostPort(l.str("HOST", defaultHost), l.port("PORT", 8080)),
		ReadHeaderTimeout: l.duration("READ_HEADER_TIMEOUT", 2*time.Second),
		ReadTimeout:       l.duration("READ_TIMEOUT", 5*time.Second),
		WriteTimeout:      l.duration("WRITE_TIMEOUT", 10*time.Second),
		IdleTimeout:       l.duration("IDLE_TIMEOUT", 60*time.Second),
		MaxHeaderBytes:    int(l.int64("MAX_HEADER_BYTES", 1<<20, 1<<10, 16<<20)),
		MaxBodyBytes:      l.int64("MAX_BODY_BYTES", 1<<20, 1, 1<<30),

		ShutdownDelay:   l.boundedDuration("SHUTDOWN_DELAY", 0, true, maxShutdownDelay),
		ShutdownTimeout: l.duration("SHUTDOWN_TIMEOUT", 15*time.Second),

		LogLevel: l.logLevel("LOG_LEVEL", slog.LevelInfo),
		LogFormat: LogFormat(l.oneOf("LOG_FORMAT", string(defaultFormat),
			string(LogText), string(LogJSON))),

		CORSAllowedOrigins: l.origins("CORS_ALLOWED_ORIGINS", env != EnvProduction),
	}

	if len(l.errs) == 0 {
		l.errs = cfg.validate()
	}
	if err := errors.Join(l.errs...); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func (c Config) validate() []error {
	var errs []error

	if c.ReadHeaderTimeout > c.ReadTimeout {
		errs = append(errs, fmt.Errorf(
			"READ_HEADER_TIMEOUT (%s) must not exceed READ_TIMEOUT (%s)",
			c.ReadHeaderTimeout, c.ReadTimeout))
	}

	if c.ShutdownTimeout < c.WriteTimeout {
		errs = append(errs, fmt.Errorf(
			"SHUTDOWN_TIMEOUT (%s) must be at least WRITE_TIMEOUT (%s)",
			c.ShutdownTimeout, c.WriteTimeout))
	}

	return errs
}

func (c Config) IsProduction() bool { return c.Env == EnvProduction }

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

type loader struct {
	getenv func(string) string
	errs   []error
}

func (l *loader) fail(key, format string, args ...any) {
	l.errs = append(l.errs, fmt.Errorf("%s: %s", key, fmt.Sprintf(format, args...)))
}

func (l *loader) str(key, def string) string {
	if v := strings.TrimSpace(l.getenv(key)); v != "" {
		return v
	}
	return def
}

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

func (l *loader) duration(key string, def time.Duration) time.Duration {
	return l.boundedDuration(key, def, false, 0)
}

func (l *loader) boundedDuration(key string, def time.Duration, allowZero bool, hi time.Duration) time.Duration {
	raw := l.str(key, "")
	if raw == "" {
		return def
	}
	d, err := time.ParseDuration(raw)
	if err != nil {
		l.fail(key, "%q is not a valid duration (examples: 500ms, 5s, 1m30s)", raw)
		return def
	}
	switch {
	case d < 0:
		l.fail(key, "%q must not be negative", raw)
		return def
	case d == 0 && !allowZero:
		l.fail(key, "%q must be positive", raw)
		return def
	case hi > 0 && d > hi:
		l.fail(key, "%q must not exceed %s", raw, hi)
		return def
	}
	return d
}

func (l *loader) int64(key string, def, lo, hi int64) int64 {
	raw := l.str(key, "")
	if raw == "" {
		return def
	}
	n, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		l.fail(key, "%q is not a valid integer", raw)
		return def
	}
	if n < lo || n > hi {
		l.fail(key, "%d is out of range (%d to %d)", n, lo, hi)
		return def
	}
	return n
}

func (l *loader) oneOf(key, def string, allowed ...string) string {
	raw := l.str(key, "")
	if raw == "" {
		return def
	}
	if v, ok := match(raw, allowed); ok {
		return v
	}
	l.fail(key, "%q must be one of: %s", raw, strings.Join(allowed, ", "))
	return def
}

func (l *loader) requiredOneOf(key string, allowed ...string) string {
	raw := l.str(key, "")
	if raw == "" {
		l.fail(key, "is required (one of: %s)", strings.Join(allowed, ", "))
		return ""
	}
	if v, ok := match(raw, allowed); ok {
		return v
	}
	l.fail(key, "%q must be one of: %s", raw, strings.Join(allowed, ", "))
	return ""
}

func match(raw string, allowed []string) (string, bool) {
	for _, a := range allowed {
		if strings.EqualFold(raw, a) {
			return a, true
		}
	}
	return "", false
}

func (l *loader) list(key string) []string {
	var out []string
	for _, part := range strings.Split(l.getenv(key), ",") {
		if p := strings.TrimSpace(part); p != "" {
			out = append(out, p)
		}
	}
	return out
}

func (l *loader) origins(key string, allowWildcard bool) []string {
	var out []string
	seen := map[string]bool{}
	for _, o := range l.list(key) {
		if o == "*" {
			if !allowWildcard {
				l.fail(key, `"*" is not allowed in production`)
				continue
			}
		} else {
			norm, err := normalizeOrigin(o)
			if err != nil {
				l.fail(key, "%q: %v", o, err)
				continue
			}
			o = norm
		}
		if !seen[o] {
			seen[o] = true
			out = append(out, o)
		}
	}
	return out
}

func normalizeOrigin(raw string) (string, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return "", errors.New("not a valid URL")
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return "", errors.New("scheme must be http or https")
	}
	if u.Host == "" || u.User != nil || u.Opaque != "" ||
		u.Path != "" || u.RawQuery != "" || u.ForceQuery || u.Fragment != "" {
		return "", errors.New("must be scheme://host[:port] with no path, query, or trailing slash")
	}
	return u.Scheme + "://" + strings.ToLower(u.Host), nil
}

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
