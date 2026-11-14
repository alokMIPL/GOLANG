package main

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/example/taskapi/internal/config"
)

// readHeaderTimeout guards against slowloris-style clients regardless of the
// configurable read/write timeouts.
const readHeaderTimeout = 5 * time.Second

// newServer builds an http.Server with all timeouts set, and routes the
// server's internal error log through slog.
func newServer(cfg *config.Config, handler http.Handler, log *slog.Logger) *http.Server {
	return &http.Server{
		Addr:              cfg.Addr,
		Handler:           handler,
		ReadHeaderTimeout: readHeaderTimeout,
		ReadTimeout:       cfg.ReadTimeout,
		WriteTimeout:      cfg.WriteTimeout,
		IdleTimeout:       cfg.IdleTimeout,
		ErrorLog:          slog.NewLogLogger(log.Handler(), slog.LevelError),
	}
}
