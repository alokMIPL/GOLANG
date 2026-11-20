package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"time"

	"github.com/example/taskapi/internal/config"
)

const (
	// readHeaderTimeout guards against slowloris-style clients regardless of
	// the configurable read/write timeouts.
	readHeaderTimeout = 5 * time.Second

	// maxHeaderBytes caps request header size (the stdlib default is 1 MiB;
	// we make it explicit so it is visible and easy to tune).
	maxHeaderBytes = 1 << 20

	// shutdownTimeout is how long in-flight requests get to finish after the
	// shutdown signal before connections are forcibly closed.
	shutdownTimeout = 15 * time.Second
)

// newServer builds an http.Server with all timeouts set, routes the server's
// internal error log through slog, and logs connection state changes at debug
// level.
func newServer(cfg *config.Config, handler http.Handler, log *slog.Logger) *http.Server {
	return &http.Server{
		Addr:              cfg.Addr,
		Handler:           handler,
		ReadHeaderTimeout: readHeaderTimeout,
		ReadTimeout:       cfg.ReadTimeout,
		WriteTimeout:      cfg.WriteTimeout,
		IdleTimeout:       cfg.IdleTimeout,
		MaxHeaderBytes:    maxHeaderBytes,
		ErrorLog:          slog.NewLogLogger(log.Handler(), slog.LevelError),
		ConnState: func(c net.Conn, state http.ConnState) {
			log.Debug("connection state changed",
				"remote", c.RemoteAddr().String(),
				"state", state.String(),
			)
		},
	}
}

// run starts the server and blocks until ctx is cancelled or the server fails.
// On cancellation it shuts down gracefully, waiting up to shutdownTimeout for
// in-flight requests to complete.
func run(ctx context.Context, cfg *config.Config, handler http.Handler, log *slog.Logger) error {
	srv := newServer(cfg, handler, log)

	// Listen explicitly so bind errors (port in use, bad address) surface
	// immediately rather than inside the serving goroutine.
	ln, err := net.Listen("tcp", srv.Addr)
	if err != nil {
		return fmt.Errorf("listen on %q: %w", srv.Addr, err)
	}

	serveErr := make(chan error, 1)
	go func() {
		log.Info("server listening", "addr", ln.Addr().String())
		serveErr <- srv.Serve(ln)
	}()

	select {
	case err := <-serveErr:
		// Serve only returns on its own if something went wrong.
		return fmt.Errorf("serve: %w", err)
	case <-ctx.Done():
		log.Info("shutdown signal received", "timeout", shutdownTimeout.String())
	}

	// ctx is already cancelled, so derive the shutdown context from a fresh one.
	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		// Graceful shutdown timed out; drop remaining connections.
		log.Error("graceful shutdown failed, forcing close", "err", err)
		_ = srv.Close()
		return fmt.Errorf("shutdown: %w", err)
	}

	// Serve returns http.ErrServerClosed once Shutdown has been called.
	if err := <-serveErr; !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("serve: %w", err)
	}

	log.Info("server stopped cleanly")
	return nil
}

// Example wiring in main():
//
//	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
//	defer stop()
//
//	if err := run(ctx, cfg, mux, logger); err != nil {
//		logger.Error("server exited with error", "err", err)
//		os.Exit(1)
//	}
