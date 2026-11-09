package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/yourname/taskapi/internal/config"
	"github.com/yourname/taskapi/internal/handler"
	"github.com/yourname/taskapi/internal/store"
)

const (
	defaultReadTimeout     = 5 * time.Second
	defaultWriteTimeout    = 10 * time.Second
	defaultIdleTimeout     = 60 * time.Second
	defaultShutdownTimeout = 5 * time.Second
	maxHeaderBytes         = 1 << 20 // 1 MiB
)

// logger gives every log line the same prefix.
var logger = log.New(os.Stdout, "[taskapi] ", log.LstdFlags|log.Lmsgprefix)

func main() {
	if err := run(); err != nil {
		logger.Printf("fatal: %v", err)
		os.Exit(1)
	}
}

// run wires the app together and blocks until the server stops.
func run() error {
	logger.Println("starting up")

	cfg := config.Load()
	if err := validatePort(cfg.Port); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}

	// Dependencies.
	taskStore := store.NewMemory()
	h := handler.New(taskStore)

	// Routes plus the middleware chain (defined in middleware.go).
	root := buildHandler(h.Routes())

	srv := &http.Server{
		Addr:           ":" + cfg.Port,
		Handler:        root,
		ReadTimeout:    defaultReadTimeout,
		WriteTimeout:   defaultWriteTimeout,
		IdleTimeout:    defaultIdleTimeout,
		MaxHeaderBytes: maxHeaderBytes,
		ErrorLog:       logger,
	}

	// Run the server in the background and report failures on a channel.
	serverErr := make(chan error, 1)
	go func() {
		logger.Printf("listening on :%s", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
		}
		close(serverErr)
	}()

	// Wait for either a shutdown signal or a server failure.
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(stop)

	select {
	case sig := <-stop:
		logger.Printf("received signal %q, shutting down", sig)
	case err := <-serverErr:
		if err != nil {
			return fmt.Errorf("server error: %w", err)
		}
	}

	return shutdown(srv, shutdownTimeout())
}

// shutdown gracefully stops the server, waiting for in-flight requests.
func shutdown(srv *http.Server, timeout time.Duration) error {
	logger.Printf("waiting up to %s for in-flight requests", timeout)

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		// Graceful shutdown failed or timed out, so force close.
		_ = srv.Close()
		return fmt.Errorf("graceful shutdown failed: %w", err)
	}

	logger.Println("server stopped")
	return nil
}

// shutdownTimeout reads SHUTDOWN_TIMEOUT_SECONDS, falling back to a default.
func shutdownTimeout() time.Duration {
	raw := os.Getenv("SHUTDOWN_TIMEOUT_SECONDS")
	if raw == "" {
		return defaultShutdownTimeout
	}

	secs, err := strconv.Atoi(raw)
	if err != nil || secs <= 0 {
		logger.Printf("ignoring invalid SHUTDOWN_TIMEOUT_SECONDS=%q", raw)
		return defaultShutdownTimeout
	}
	return time.Duration(secs) * time.Second
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
