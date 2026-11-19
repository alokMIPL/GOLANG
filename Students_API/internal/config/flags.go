package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

// Set at build time:
//
//	go build -ldflags "-X main.version=1.2.3 -X main.commit=abc123 -X main.date=2026-11-19"
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

type options struct {
	showVersion bool
	migrateOnly bool
	healthcheck bool

	addr            string
	logLevel        string
	readTimeout     time.Duration
	writeTimeout    time.Duration
	shutdownTimeout time.Duration
	healthTimeout   time.Duration
}

// envOr returns the environment variable's value, or def if unset/empty.
func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func parseFlags(args []string, out io.Writer) (options, error) {
	var o options

	fs := flag.NewFlagSet("taskapi", flag.ContinueOnError)
	fs.SetOutput(out)

	// Modes.
	fs.BoolVar(&o.showVersion, "version", false, "print version and exit")
	fs.BoolVar(&o.migrateOnly, "migrate-only", false, "apply database migrations and exit")
	fs.BoolVar(&o.healthcheck, "healthcheck", false,
		"probe the local /healthz endpoint and exit 0 if healthy (for container HEALTHCHECK)")

	// Server settings (flags override env vars, env vars override defaults).
	fs.StringVar(&o.addr, "addr", envOr("TASKAPI_ADDR", ":8080"), "listen address (env TASKAPI_ADDR)")
	fs.StringVar(&o.logLevel, "log-level", envOr("TASKAPI_LOG_LEVEL", "info"),
		"log level: debug, info, warn, error (env TASKAPI_LOG_LEVEL)")
	fs.DurationVar(&o.readTimeout, "read-timeout", 10*time.Second, "HTTP read timeout")
	fs.DurationVar(&o.writeTimeout, "write-timeout", 15*time.Second, "HTTP write timeout")
	fs.DurationVar(&o.shutdownTimeout, "shutdown-timeout", 20*time.Second, "graceful shutdown timeout")
	fs.DurationVar(&o.healthTimeout, "healthcheck-timeout", 3*time.Second, "healthcheck probe timeout")

	fs.Usage = func() {
		fmt.Fprintf(out, "Usage: taskapi [flags]\n\nFlags:\n")
		fs.PrintDefaults()
	}

	if err := fs.Parse(args); err != nil {
		return options{}, err
	}
	if fs.NArg() > 0 {
		return options{}, fmt.Errorf("unexpected arguments: %s", strings.Join(fs.Args(), " "))
	}
	if err := o.validate(); err != nil {
		return options{}, err
	}
	return o, nil
}

func (o options) validate() error {
	modes := 0
	for _, m := range []bool{o.showVersion, o.migrateOnly, o.healthcheck} {
		if m {
			modes++
		}
	}
	if modes > 1 {
		return errors.New("-version, -migrate-only and -healthcheck are mutually exclusive")
	}
	if _, err := parseLevel(o.logLevel); err != nil {
		return err
	}
	if o.addr == "" {
		return errors.New("-addr must not be empty")
	}
	for name, d := range map[string]time.Duration{
		"read-timeout":        o.readTimeout,
		"write-timeout":       o.writeTimeout,
		"shutdown-timeout":    o.shutdownTimeout,
		"healthcheck-timeout": o.healthTimeout,
	} {
		if d <= 0 {
			return fmt.Errorf("-%s must be greater than zero", name)
		}
	}
	return nil
}

func parseLevel(s string) (slog.Level, error) {
	var l slog.Level
	switch strings.ToLower(s) {
	case "debug":
		l = slog.LevelDebug
	case "info":
		l = slog.LevelInfo
	case "warn", "warning":
		l = slog.LevelWarn
	case "error":
		l = slog.LevelError
	default:
		return l, fmt.Errorf("invalid log level %q (want debug, info, warn or error)", s)
	}
	return l, nil
}

// healthURL turns a listen address like ":8080" or "0.0.0.0:8080" into a
// loopback URL suitable for probing.
func healthURL(addr string) (string, error) {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return "", fmt.Errorf("invalid addr %q: %w", addr, err)
	}
	if host == "" || host == "0.0.0.0" || host == "::" {
		host = "127.0.0.1"
	}
	return "http://" + net.JoinHostPort(host, port) + "/healthz", nil
}

func runHealthcheck(ctx context.Context, o options) error {
	url, err := healthURL(o.addr)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(ctx, o.healthTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unhealthy: %s returned %s", url, resp.Status)
	}
	return nil
}

// runMigrations is a placeholder; wire in your migration tool here
// (golang-migrate, goose, atlas, ...).
func runMigrations(ctx context.Context, log *slog.Logger) error {
	log.InfoContext(ctx, "applying migrations")
	// TODO: apply migrations.
	log.InfoContext(ctx, "migrations complete")
	return nil
}

func newServer(o options, log *slog.Logger) *http.Server {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, "ok\n")
	})
	mux.HandleFunc("/version", func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprintf(w, "%s (commit %s, built %s)\n", version, commit, date)
	})

	return &http.Server{
		Addr:              o.addr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       o.readTimeout,
		WriteTimeout:      o.writeTimeout,
		IdleTimeout:       60 * time.Second,
		ErrorLog:          slog.NewLogLogger(log.Handler(), slog.LevelError),
	}
}

func serve(ctx context.Context, o options, log *slog.Logger) error {
	srv := newServer(o, log)

	errCh := make(chan error, 1)
	go func() {
		log.Info("server listening", "addr", o.addr, "version", version)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
		close(errCh)
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		log.Info("shutdown signal received")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), o.shutdownTimeout)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("graceful shutdown failed: %w", err)
	}
	log.Info("server stopped cleanly")
	return nil
}

// run holds all program logic and returns an exit code, which keeps main
// trivial and makes the whole thing testable.
func run(args []string, stdout, stderr io.Writer) int {
	o, err := parseFlags(args, stderr)
	if err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		fmt.Fprintf(stderr, "error: %v\n", err)
		return 2
	}

	if o.showVersion {
		fmt.Fprintf(stdout, "taskapi %s (commit %s, built %s)\n", version, commit, date)
		return 0
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if o.healthcheck {
		if err := runHealthcheck(ctx, o); err != nil {
			fmt.Fprintf(stderr, "healthcheck failed: %v\n", err)
			return 1
		}
		return 0
	}

	level, _ := parseLevel(o.logLevel) // already validated
	log := slog.New(slog.NewJSONHandler(stderr, &slog.HandlerOptions{Level: level}))

	if err := runMigrations(ctx, log); err != nil {
		log.Error("migrations failed", "err", err)
		return 1
	}
	if o.migrateOnly {
		return 0
	}

	if err := serve(ctx, o, log); err != nil {
		log.Error("server error", "err", err)
		return 1
	}
	return 0
}

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}
