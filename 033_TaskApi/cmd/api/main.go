package main

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"strconv"
	"strings"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/yourname/taskapi/internal/config"
	"github.com/yourname/taskapi/internal/handler"
	"github.com/yourname/taskapi/internal/store"
)

// Defaults used when the matching environment variable is unset or invalid.
const (
	defaultReadTimeout     = 5 * time.Second
	defaultWriteTimeout    = 10 * time.Second
	defaultIdleTimeout     = 60 * time.Second
	defaultShutdownTimeout = 5 * time.Second
	defaultDrainDelay      = 0 * time.Second
	defaultMaxHeaderBytes  = 1 << 20 // 1 MiB
)

// Build information. Override at build time, for example:
//
//	go build -ldflags "-X main.version=1.2.3 -X main.commit=abc123 -X main.buildTime=2025-01-01T00:00:00Z"
var (
	version   = "dev"
	commit    = "none"
	buildTime = "unknown"
)

// logger gives every log line the same prefix.
var logger = log.New(os.Stdout, "[taskapi] ", log.LstdFlags|log.Lmsgprefix)

// appState tracks whether the process is ready to receive traffic.
// The flag is flipped to false at the start of shutdown so that load
// balancers can stop routing requests before the listener closes.
type appState struct {
	ready     atomic.Bool
	startedAt time.Time
}

func newAppState() *appState {
	s := &appState{startedAt: time.Now()}
	s.ready.Store(false)
	return s
}

// serverSettings holds every tunable that comes from the environment.
type serverSettings struct {
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	IdleTimeout     time.Duration
	ShutdownTimeout time.Duration
	DrainDelay      time.Duration
	MaxHeaderBytes  int
	TLSCertFile     string
	TLSKeyFile      string
}

// tlsEnabled reports whether both a certificate and key were supplied.
func (s serverSettings) tlsEnabled() bool {
	return s.TLSCertFile != "" && s.TLSKeyFile != ""
}

// loadSettings reads all server tunables from the environment.
func loadSettings() serverSettings {
	return serverSettings{
		ReadTimeout:     envDuration("READ_TIMEOUT_SECONDS", defaultReadTimeout),
		WriteTimeout:    envDuration("WRITE_TIMEOUT_SECONDS", defaultWriteTimeout),
		IdleTimeout:     envDuration("IDLE_TIMEOUT_SECONDS", defaultIdleTimeout),
		ShutdownTimeout: envDuration("SHUTDOWN_TIMEOUT_SECONDS", defaultShutdownTimeout),
		DrainDelay:      envDurationAllowZero("DRAIN_DELAY_SECONDS", defaultDrainDelay),
		MaxHeaderBytes:  envInt("MAX_HEADER_BYTES", defaultMaxHeaderBytes),
		TLSCertFile:     strings.TrimSpace(os.Getenv("TLS_CERT_FILE")),
		TLSKeyFile:      strings.TrimSpace(os.Getenv("TLS_KEY_FILE")),
	}
}

// validate checks the settings for combinations that cannot work.
func (s serverSettings) validate() error {
	if (s.TLSCertFile == "") != (s.TLSKeyFile == "") {
		return errors.New("TLS_CERT_FILE and TLS_KEY_FILE must be set together")
	}
	if s.tlsEnabled() {
		for _, path := range []string{s.TLSCertFile, s.TLSKeyFile} {
			if _, err := os.Stat(path); err != nil {
				return fmt.Errorf("tls file %q is not readable: %w", path, err)
			}
		}
	}
	if s.MaxHeaderBytes < 1024 {
		return fmt.Errorf("MAX_HEADER_BYTES %d is too small (minimum 1024)", s.MaxHeaderBytes)
	}
	if s.WriteTimeout < s.ReadTimeout {
		logger.Printf("warning: write timeout (%s) is shorter than read timeout (%s)",
			s.WriteTimeout, s.ReadTimeout)
	}
	return nil
}

// logSummary prints the effective configuration at startup.
func (s serverSettings) logSummary() {
	logger.Printf("read timeout:     %s", s.ReadTimeout)
	logger.Printf("write timeout:    %s", s.WriteTimeout)
	logger.Printf("idle timeout:     %s", s.IdleTimeout)
	logger.Printf("shutdown timeout: %s", s.ShutdownTimeout)
	logger.Printf("drain delay:      %s", s.DrainDelay)
	logger.Printf("max header bytes: %d", s.MaxHeaderBytes)
	logger.Printf("tls enabled:      %t", s.tlsEnabled())
}

func main() {
	if err := run(); err != nil {
		logger.Printf("fatal: %v", err)
		os.Exit(1)
	}
}

// run wires the app together and blocks until the server stops.
func run() error {
	logBuildInfo()
	logger.Println("starting up")

	cfg := config.Load()
	if err := validatePort(cfg.Port); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}

	settings := loadSettings()
	if err := settings.validate(); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}
	settings.logSummary()

	// Dependencies.
	taskStore := store.NewMemory()
	h := handler.New(taskStore)

	// Routes plus the middleware chain (defined in middleware.go).
	app := buildHandler(h.Routes())

	// Operational endpoints (health, readiness, version) sit in front of the app.
	state := newAppState()
	root := withOpsEndpoints(app, state)

	srv := &http.Server{
		Addr:           ":" + cfg.Port,
		Handler:        root,
		ReadTimeout:    settings.ReadTimeout,
		WriteTimeout:   settings.WriteTimeout,
		IdleTimeout:    settings.IdleTimeout,
		MaxHeaderBytes: settings.MaxHeaderBytes,
		ErrorLog:       logger,
	}
	if settings.tlsEnabled() {
		srv.TLSConfig = &tls.Config{MinVersion: tls.VersionTLS12}
	}

	// Run the server in the background and report failures on a channel.
	serverErr := make(chan error, 1)
	go func() {
		defer close(serverErr)

		var err error
		if settings.tlsEnabled() {
			logger.Printf("listening on :%s (https)", cfg.Port)
			state.ready.Store(true)
			err = srv.ListenAndServeTLS(settings.TLSCertFile, settings.TLSKeyFile)
		} else {
			logger.Printf("listening on :%s (http)", cfg.Port)
			state.ready.Store(true)
			err = srv.ListenAndServe()
		}

		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			state.ready.Store(false)
			serverErr <- err
		}
	}()

	// Wait for either a shutdown signal or a server failure.
	stop := make(chan os.Signal, 2)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(stop)

	select {
	case sig := <-stop:
		logger.Printf("received signal %q, shutting down", sig)
	case err := <-serverErr:
		if err != nil {
			return fmt.Errorf("server error: %w", err)
		}
		return nil
	}

	// A second signal while shutting down means "stop waiting and exit now".
	go func() {
		sig, ok := <-stop
		if ok {
			logger.Printf("received second signal %q, forcing exit", sig)
			os.Exit(1)
		}
	}()

	// Tell load balancers we are going away, then give them time to notice.
	state.ready.Store(false)
	drain(settings.DrainDelay)

	return shutdown(srv, settings.ShutdownTimeout)
}

// withOpsEndpoints serves /healthz, /readyz and /version itself and
// forwards every other request to next.
func withOpsEndpoints(next http.Handler, state *appState) http.Handler {
	mux := http.NewServeMux()

	// Liveness: the process is up and able to answer.
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{
			"status": "ok",
			"uptime": time.Since(state.startedAt).Round(time.Second).String(),
		})
	})

	// Readiness: the process wants traffic. Fails during startup and shutdown.
	mux.HandleFunc("/readyz", func(w http.ResponseWriter, r *http.Request) {
		if !state.ready.Load() {
			writeJSON(w, http.StatusServiceUnavailable, map[string]any{"status": "not ready"})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"status": "ready"})
	})

	// Version: which build is running.
	mux.HandleFunc("/version", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{
			"version":   version,
			"commit":    commit,
			"buildTime": buildTime,
			"goVersion": runtime.Version(),
		})
	})

	// Everything else belongs to the application.
	mux.Handle("/", next)
	return mux
}

// writeJSON encodes v as a JSON response with the given status code.
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		logger.Printf("failed to write json response: %v", err)
	}
}

// drain waits for the given delay so that upstream load balancers can
// observe the failing readiness probe before the listener closes.
func drain(delay time.Duration) {
	if delay <= 0 {
		return
	}
	logger.Printf("draining for %s before closing the listener", delay)
	time.Sleep(delay)
}

// shutdown gracefully stops the server, waiting for in-flight requests.
func shutdown(srv *http.Server, timeout time.Duration) error {
	logger.Printf("waiting up to %s for in-flight requests", timeout)

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	started := time.Now()
	if err := srv.Shutdown(ctx); err != nil {
		// Graceful shutdown failed or timed out, so force close.
		logger.Printf("graceful shutdown did not finish after %s", time.Since(started).Round(time.Millisecond))
		if closeErr := srv.Close(); closeErr != nil {
			logger.Printf("force close failed: %v", closeErr)
		}
		return fmt.Errorf("graceful shutdown failed: %w", err)
	}

	logger.Printf("server stopped cleanly in %s", time.Since(started).Round(time.Millisecond))
	return nil
}

// logBuildInfo prints version and runtime details once at startup.
func logBuildInfo() {
	logger.Printf("version=%s commit=%s built=%s", version, commit, buildTime)
	logger.Printf("go=%s os=%s arch=%s cpus=%d pid=%d",
		runtime.Version(), runtime.GOOS, runtime.GOARCH, runtime.NumCPU(), os.Getpid())
}

// envDuration reads a positive whole number of seconds from the environment.
// Unset, invalid or non-positive values fall back to def.
func envDuration(name string, def time.Duration) time.Duration {
	raw := os.Getenv(name)
	if raw == "" {
		return def
	}

	secs, err := strconv.Atoi(raw)
	if err != nil || secs <= 0 {
		logger.Printf("ignoring invalid %s=%q", name, raw)
		return def
	}
	return time.Duration(secs) * time.Second
}

// envDurationAllowZero is like envDuration but accepts 0 as a real value.
func envDurationAllowZero(name string, def time.Duration) time.Duration {
	raw := os.Getenv(name)
	if raw == "" {
		return def
	}

	secs, err := strconv.Atoi(raw)
	if err != nil || secs < 0 {
		logger.Printf("ignoring invalid %s=%q", name, raw)
		return def
	}
	return time.Duration(secs) * time.Second
}

// envInt reads a positive integer from the environment, falling back to def.
func envInt(name string, def int) int {
	raw := os.Getenv(name)
	if raw == "" {
		return def
	}

	n, err := strconv.Atoi(raw)
	if err != nil || n <= 0 {
		logger.Printf("ignoring invalid %s=%q", name, raw)
		return def
	}
	return n
}

// validatePort makes sure the port is a number in the valid TCP range.
func validatePort(port string) error {
	n, err := strconv.Atoi(port)
	if err != nil {
		return fmt.Errorf("port %q is not a number", port)
	}
	if n < 1 || n > 65535 {
		return fmt.Errorf("port %d is out of range (1-65535)", n)
	}
	return nil
}
