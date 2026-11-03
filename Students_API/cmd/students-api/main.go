package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	if err := run(os.Getenv, os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "fatal:", err)
		os.Exit(1)
	}
}

func run(getenv func(string) string, stdout io.Writer) error {
	cfg, err := loadConfig(getenv)
	if err != nil {
		return fmt.Errorf("config: %w", err)
	}

	logger := slog.New(slog.NewJSONHandler(stdout, &slog.HandlerOptions{Level: cfg.LogLevel}))
	slog.SetDefault(logger)

	// Cancelled on Ctrl+C / SIGTERM.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Startup work (open, migrate) gets its own deadline and is released as
	// soon as it is done rather than living until run returns.
	startCtx, cancelStart := context.WithTimeout(ctx, 10*time.Second)
	db, err := openSQLite(startCtx, cfg.DBPath, cfg.MaxOpenConns)
	if err != nil {
		cancelStart()
		return fmt.Errorf("open database: %w", err)
	}
	defer db.Close()

	store := newSQLiteStore(db)
	err = store.migrate(startCtx)
	cancelStart()
	if err != nil {
		return fmt.Errorf("migrate: %w", err)
	}

	if cfg.APIToken == "" {
		logger.Warn("API_TOKEN is not set: authentication is disabled")
	}

	srv := &http.Server{
		Addr:              cfg.Addr,
		Handler:           newHandler(store, logger, cfg),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      cfg.RequestTimeout + 5*time.Second, // must outlast the handler deadline
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    1 << 16,
		ErrorLog:          slog.NewLogLogger(logger.Handler(), slog.LevelError),
	}

	errCh := make(chan error, 1)
	go func() {
		logger.Info("server listening", "addr", cfg.Addr, "db", cfg.DBPath)
		errCh <- srv.ListenAndServe()
	}()

	select {
	case err := <-errCh:
		// Server failed on its own (e.g. port already in use).
		if !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("server: %w", err)
		}
		return nil
	case <-ctx.Done():
		stop() // restore default signal handling so a second Ctrl+C force-quits
		logger.Info("shutting down", "timeout", cfg.ShutdownTimeout.String())

		shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
		defer cancel()
		if err := srv.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("shutdown: %w", err)
		}
		if err := <-errCh; !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("server: %w", err)
		}
		logger.Info("stopped cleanly")
		return nil
	}
}
