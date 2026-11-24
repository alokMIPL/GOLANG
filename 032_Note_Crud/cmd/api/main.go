package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/yourname/taskapi/internal/config"
	"github.com/yourname/taskapi/internal/handler"
	"github.com/yourname/taskapi/internal/store"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil)).With("app", "taskapi")
	slog.SetDefault(logger)

	if err := run(context.Background(), logger, os.Getenv); err != nil {
		logger.Error("fatal", "err", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, logger *slog.Logger, getenv func(string) string) error {
	cfg, err := config.Load(getenv)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	ctx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer stop()

	taskStore := store.NewMemory()
	h := handler.New(taskStore, logger)

	srv := &http.Server{
		Handler:           buildHandler(h.Routes(), logger),
		ReadHeaderTimeout: cfg.ReadHeaderTimeout,
		ReadTimeout:       cfg.ReadTimeout,
		WriteTimeout:      cfg.WriteTimeout,
		IdleTimeout:       cfg.IdleTimeout,
		MaxHeaderBytes:    cfg.MaxHeaderBytes,
		ErrorLog:          slog.NewLogLogger(logger.Handler(), slog.LevelError),
	}

	ln, err := net.Listen("tcp", cfg.Addr)
	if err != nil {
		return fmt.Errorf("listen on %s: %w", cfg.Addr, err)
	}

	serverErr := make(chan error, 1)
	go func() {
		logger.Info("listening", "addr", ln.Addr().String())
		if err := srv.Serve(ln); !errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
		}
	}()

	select {
	case err := <-serverErr:
		return fmt.Errorf("serve: %w", err)
	case <-ctx.Done():
		stop() // a second Ctrl+C now terminates immediately
		logger.Info("shutdown signal received", "timeout", cfg.ShutdownTimeout.String())
	}

	// ctx is already cancelled, so detach it for the shutdown deadline.
	shutCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), cfg.ShutdownTimeout)
	defer cancel()

	if err := srv.Shutdown(shutCtx); err != nil {
		_ = srv.Close()
		return fmt.Errorf("graceful shutdown: %w", err)
	}

	logger.Info("server stopped")
	return nil
}
