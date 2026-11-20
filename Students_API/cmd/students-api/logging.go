package main

import (
	"io"
	"log/slog"
	"os"
	"strings"
)

const defaultServiceName = "taskapi"

// loggerConfig holds every knob for building the logger.
type loggerConfig struct {
	Writer     io.Writer
	Level      slog.Leveler
	AddSource  bool
	Service    string
	Version    string
	SetDefault bool
}

// defaultLoggerConfig returns the settings used by newLogger.
func defaultLoggerConfig(w io.Writer, level slog.Leveler) loggerConfig {
	return loggerConfig{
		Writer:     w,
		Level:      level,
		AddSource:  false,
		Service:    defaultServiceName,
		Version:    versionString(),
		SetDefault: true,
	}
}

func levelFromEnv(key string, fallback slog.Level) slog.Level {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback
	}

	var lvl slog.Level
	if err := lvl.UnmarshalText([]byte(raw)); err != nil {
		return fallback
	}
	return lvl
}

// replaceAttr renames the built-in time key to "timestamp" and lowercases
// the level value so output is consistent across tools.
func replaceAttr(groups []string, a slog.Attr) slog.Attr {
	if len(groups) > 0 {
		return a
	}

	switch a.Key {
	case slog.TimeKey:
		a.Key = "timestamp"
	case slog.LevelKey:
		if lvl, ok := a.Value.Any().(slog.Level); ok {
			a.Value = slog.StringValue(strings.ToLower(lvl.String()))
		}
	}
	return a
}

// newLoggerFromConfig builds a JSON logger
func newLoggerFromConfig(cfg loggerConfig) *slog.Logger {
	if cfg.Writer == nil {
		cfg.Writer = os.Stderr
	}
	if cfg.Level == nil {
		cfg.Level = slog.LevelInfo
	}
	if cfg.Service == "" {
		cfg.Service = defaultServiceName
	}

	handler := slog.NewJSONHandler(cfg.Writer, &slog.HandlerOptions{
		Level:       cfg.Level,
		AddSource:   cfg.AddSource,
		ReplaceAttr: replaceAttr,
	})

	logger := slog.New(handler).With(
		"service", cfg.Service,
		"version", cfg.Version,
	)

	if cfg.SetDefault {
		slog.SetDefault(logger)
	}
	return logger
}

// newLogger returns a JSON logger tagged with the service name and version,
// and installs it as the process default.
func newLogger(w io.Writer, level slog.Leveler) *slog.Logger {
	return newLoggerFromConfig(defaultLoggerConfig(w, level))
}
