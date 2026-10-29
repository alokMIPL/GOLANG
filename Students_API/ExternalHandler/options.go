package response

import (
	"context"
	"log/slog"
	"time"
)

const (
	defaultTTL      = 30 * time.Second
	defaultMaxStale = 5 * time.Minute
	defaultTimeout  = 5 * time.Second
	defaultBackoff  = 2 * time.Second
	defaultURL      = "https://catfact.ninja/fact"
	factSource      = "catfact.ninja"
	flightKey       = "fact"
)

// FactFetcher lets tests swap in a fake upstream.
type FactFetcher func(ctx context.Context) (*CatFactResponse, error)

type Option func(*Handler)

func WithFetcher(f FactFetcher) Option {
	return func(h *Handler) {
		if f != nil {
			h.fetch = f
		}
	}
}

// WithTTL sets how long a fact stays fresh. Zero disables caching.
func WithTTL(d time.Duration) Option {
	return func(h *Handler) {
		if d >= 0 {
			h.ttl = d
		}
	}
}

// WithMaxStale sets how long past its TTL a fact may still be served when
// refreshing fails. Zero disables stale serving.
func WithMaxStale(d time.Duration) Option {
	return func(h *Handler) {
		if d >= 0 {
			h.maxStale = d
		}
	}
}

// WithTimeout bounds each refresh, including retries.
func WithTimeout(d time.Duration) Option {
	return func(h *Handler) {
		if d > 0 {
			h.timeout = d
		}
	}
}

// WithBackoff sets how long to leave upstream alone after a failed refresh.
// Zero disables the backoff.
func WithBackoff(d time.Duration) Option {
	return func(h *Handler) {
		if d >= 0 {
			h.backoff = d
		}
	}
}

// WithRetries retries a failed upstream call up to n extra times. The delay
// doubles after each attempt. All attempts share the WithTimeout budget.
func WithRetries(n int, delay time.Duration) Option {
	return func(h *Handler) {
		if n >= 0 && delay >= 0 {
			h.retries, h.retryDelay = n, delay
		}
	}
}

// WithBackgroundRefresh serves an expired (but not too old) fact immediately
// and refreshes it in the background, so clients rarely wait on upstream.
func WithBackgroundRefresh(enabled bool) Option {
	return func(h *Handler) { h.bgRefresh = enabled }
}

func WithClock(now func() time.Time) Option {
	return func(h *Handler) {
		if now != nil {
			h.now = now
		}
	}
}

func WithLogger(l *slog.Logger) Option {
	return func(h *Handler) {
		if l != nil {
			h.log = l
		}
	}
}

func NewHandler(opts ...Option) *Handler {
	h := &Handler{
		fetch:    NewCatFactFetcher(nil, defaultURL),
		ttl:      defaultTTL,
		maxStale: defaultMaxStale,
		timeout:  defaultTimeout,
		backoff:  defaultBackoff,
		now:      time.Now,
		log:      slog.Default(),
	}
	for _, opt := range opts {
		opt(h)
	}
	return h
}
