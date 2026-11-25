package config

import (
	"log/slog"
	"strings"
	"testing"
	"time"
)

func fakeEnv(m map[string]string) func(string) string {
	return func(k string) string { return m[k] }
}

func TestLoad_Defaults(t *testing.T) {
	cfg, err := Load(fakeEnv(nil))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	checks := []struct {
		name      string
		got, want any
	}{
		{"Env", cfg.Env, EnvDevelopment},
		{"Addr", cfg.Addr, ":8080"},
		{"ReadHeaderTimeout", cfg.ReadHeaderTimeout, 2 * time.Second},
		{"ReadTimeout", cfg.ReadTimeout, 5 * time.Second},
		{"WriteTimeout", cfg.WriteTimeout, 10 * time.Second},
		{"IdleTimeout", cfg.IdleTimeout, 60 * time.Second},
		{"ShutdownTimeout", cfg.ShutdownTimeout, 15 * time.Second},
		{"ShutdownDelay", cfg.ShutdownDelay, time.Duration(0)},
		{"MaxHeaderBytes", cfg.MaxHeaderBytes, 1 << 20},
		{"MaxBodyBytes", cfg.MaxBodyBytes, int64(1 << 20)},
		{"LogLevel", cfg.LogLevel, slog.LevelInfo},
		{"LogFormat", cfg.LogFormat, LogText},
	}
	for _, c := range checks {
		if c.got != c.want {
			t.Errorf("%s = %v, want %v", c.name, c.got, c.want)
		}
	}
}

func TestLoad_Overrides(t *testing.T) {
	cfg, err := Load(fakeEnv(map[string]string{
		"APP_ENV":              "production",
		"HOST":                 "127.0.0.1",
		"PORT":                 "9090",
		"READ_TIMEOUT":         "7s",
		"SHUTDOWN_DELAY":       "3s",
		"LOG_LEVEL":            "debug",
		"CORS_ALLOWED_ORIGINS": " https://a.example , https://b.example,, ",
	}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Addr != "127.0.0.1:9090" {
		t.Errorf("Addr = %q", cfg.Addr)
	}
	if cfg.ReadTimeout != 7*time.Second {
		t.Errorf("ReadTimeout = %v", cfg.ReadTimeout)
	}
	if cfg.ShutdownDelay != 3*time.Second {
		t.Errorf("ShutdownDelay = %v", cfg.ShutdownDelay)
	}
	if cfg.LogLevel != slog.LevelDebug {
		t.Errorf("LogLevel = %v", cfg.LogLevel)
	}
	if cfg.LogFormat != LogJSON {
		t.Errorf("production should default to JSON logs, got %q", cfg.LogFormat)
	}
	if got := strings.Join(cfg.CORSAllowedOrigins, "|"); got != "https://a.example|https://b.example" {
		t.Errorf("CORSAllowedOrigins = %q", got)
	}
}

func TestLoad_PortZeroAllowed(t *testing.T) {
	cfg, err := Load(fakeEnv(map[string]string{"PORT": "0"}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Addr != ":0" {
		t.Errorf("Addr = %q, want :0", cfg.Addr)
	}
}

func TestLoad_Errors(t *testing.T) {
	tests := []struct {
		name    string
		env     map[string]string
		wantErr string
	}{
		{"port not a number", map[string]string{"PORT": "abc"}, "PORT"},
		{"port too large", map[string]string{"PORT": "70000"}, "PORT"},
		{"bad duration", map[string]string{"READ_TIMEOUT": "soon"}, "READ_TIMEOUT"},
		{"negative duration", map[string]string{"IDLE_TIMEOUT": "-5s"}, "IDLE_TIMEOUT"},
		{"zero duration rejected", map[string]string{"WRITE_TIMEOUT": "0s"}, "WRITE_TIMEOUT"},
		{"bad env", map[string]string{"APP_ENV": "prod"}, "APP_ENV"},
		{"bad log level", map[string]string{"LOG_LEVEL": "loud"}, "LOG_LEVEL"},
		{"bad log format", map[string]string{"LOG_FORMAT": "xml"}, "LOG_FORMAT"},
		{"body limit too small", map[string]string{"MAX_BODY_BYTES": "0"}, "MAX_BODY_BYTES"},
		{"header timeout above read timeout", map[string]string{
			"READ_HEADER_TIMEOUT": "10s", "READ_TIMEOUT": "5s",
		}, "READ_HEADER_TIMEOUT"},
		{"shutdown shorter than write", map[string]string{
			"SHUTDOWN_TIMEOUT": "1s", "WRITE_TIMEOUT": "10s",
		}, "SHUTDOWN_TIMEOUT"},
		{"wildcard CORS in production", map[string]string{
			"APP_ENV": "production", "CORS_ALLOWED_ORIGINS": "*",
		}, "CORS_ALLOWED_ORIGINS"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Load(fakeEnv(tt.env))
			if err == nil {
				t.Fatal("expected an error, got nil")
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("error %q does not mention %q", err, tt.wantErr)
			}
		})
	}
}

func TestLoad_WildcardCORSAllowedOutsideProduction(t *testing.T) {
	_, err := Load(fakeEnv(map[string]string{
		"APP_ENV": "development", "CORS_ALLOWED_ORIGINS": "*",
	}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLoad_ReportsAllErrorsAtOnce(t *testing.T) {
	_, err := Load(fakeEnv(map[string]string{
		"PORT":         "nope",
		"READ_TIMEOUT": "bad",
		"LOG_LEVEL":    "loud",
	}))
	if err == nil {
		t.Fatal("expected an error")
	}
	for _, key := range []string{"PORT", "READ_TIMEOUT", "LOG_LEVEL"} {
		if !strings.Contains(err.Error(), key) {
			t.Errorf("error should mention %s, got: %v", key, err)
		}
	}
}

func TestNewServer_AppliesLimits(t *testing.T) {
	cfg, err := Load(fakeEnv(map[string]string{"PORT": "0"}))
	if err != nil {
		t.Fatal(err)
	}
	srv := cfg.NewServer(nil, nil)

	if srv.ReadHeaderTimeout != cfg.ReadHeaderTimeout ||
		srv.ReadTimeout != cfg.ReadTimeout ||
		srv.WriteTimeout != cfg.WriteTimeout ||
		srv.IdleTimeout != cfg.IdleTimeout ||
		srv.MaxHeaderBytes != cfg.MaxHeaderBytes {
		t.Error("server did not receive configured limits")
	}
}
