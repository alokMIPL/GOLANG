package main

import (
	"io"
	"log/slog"
)

// newLogger returns a JSON logger tagged with the service name and version,
// and installs it as the process default.
func newLogger(w io.Writer, level slog.Leveler) *slog.Logger {
	h := slog.NewJSONHandler(w, &slog.HandlerOptions{Level: level})
	logger := slog.New(h).With(
		"service", "taskapi",
		"version", versionString(),
	)
	slog.SetDefault(logger)
	return logger
}
