package response

import (
	"context"
	"log/slog"
	"net/http"
	"strings"
	"time"
)

// Default configuration values used by NewHandler when no option overrides
// them.
const (
	defaultTTL      = 30 * time.Second
	defaultMaxStale = 5 * time.Minute
	defaultTimeout  = 5 * time.Second
	defaultBackoff  = 2 * time.Second
	defaultRetries  = 0
	defaultURL      = "https://catfact.ninja/fact"
	factSource      = "catfact.ninja"
	flightKey       = "fact"
	loggerName      = "response"
)

// FactFetcher lets tests swap in a fake upstream.
type FactFetcher func(ctx context.Context) (*CatFactResponse, error)

// Option configures a Handler. Options are applied in order, so a later
// option wins over an earlier one. Options with invalid arguments are
// ignored and leave the default (or previously set) value in place.
type Option func(*Handler)

// WithFetcher replaces the upstream fetcher entirely. It takes precedence
// over WithURL and WithHTTPClient, which only affect the default fetcher.
// A nil fetcher is ignored.
func WithFetcher(f FactFetcher) Option {
	return func(h *Handler) {
		if f != nil {
			h.fetch = f
			h.customFetch = true
		}
	}
}

// WithURL sets the upstream endpoint used by the default fetcher. Leading
// and trailing whitespace is trimmed; an empty URL is ignored.
func WithURL(url string) Option {
	return func(h *Handler) {
		if url = strings.TrimSpace(url); url != "" {
			h.url = url
		}
	}
}

// WithHTTPClient sets the HTTP client used by the default fetcher, for
// example to add a custom transport, proxy or TLS configuration. A nil
// client is ignored, which makes the fetcher fall back to its own default.
func WithHTTPClient(c *http.Client) Option {
	return func(h *Handler) {
		if c != nil {
			h.httpClient = c
		}
	}
}

// WithTTL sets how long a fact stays fresh. Zero disables caching.
// Negative values are ignored.
func WithTTL(d time.Duration) Option {
	return func(h *Handler) {
		if d >= 0 {
			h.ttl = d
		}
	}
}

// WithMaxStale sets how long past its TTL a fact may still be served when
// refreshing fails. Zero disables stale serving. Negative values are ignored.
func WithMaxStale(d time.Duration) Option {
	return func(h *Handler) {
		if d >= 0 {
			h.maxStale = d
		}
	}
}

// WithTimeout bounds each refresh, including retries. Zero and negative
// values are ignored, because a refresh with no deadline could hang forever.
func WithTimeout(d time.Duration) Option {
	return func(h *Handler) {
		if d > 0 {
			h.timeout = d
		}
	}
}

// WithBackoff sets how long to leave upstream alone after a failed refresh.
// Zero disables the backoff. Negative values are ignored.
func WithBackoff(d time.Duration) Option {
	return func(h *Handler) {
		if d >= 0 {
			h.backoff = d
		}
	}
}

// WithRetries retries a failed upstream call up to n extra times. The delay
// doubles after each attempt. All attempts share the WithTimeout budget.
// Negative arguments are ignored.
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

// WithClock replaces the time source, which makes TTL, stale and backoff
// behavior deterministic in tests. A nil clock is ignored.
func WithClock(now func() time.Time) Option {
	return func(h *Handler) {
		if now != nil {
			h.now = now
		}
	}
}

// WithLogger sets the logger. A nil logger is ignored. NewHandler tags the
// logger with a component attribute so handler logs are easy to filter.
func WithLogger(l *slog.Logger) Option {
	return func(h *Handler) {
		if l != nil {
			h.log = l
		}
	}
}

// NewHandler builds a Handler with sensible defaults, then applies opts in
// order.
func NewHandler(opts ...Option) *Handler {
	h := &Handler{
		ttl:      defaultTTL,
		maxStale: defaultMaxStale,
		timeout:  defaultTimeout,
		backoff:  defaultBackoff,
		retries:  defaultRetries,
		url:      defaultURL,
		now:      time.Now,
		log:      slog.Default(),
	}
	for _, opt := range opts {
		if opt != nil {
			opt(h)
		}
	}

	// Build the default fetcher only after all options have run, so WithURL
	// and WithHTTPClient take effect regardless of their position.
	if !h.customFetch {
		h.fetch = NewCatFactFetcher(h.httpClient, h.url)
	}
	h.log = h.log.With("component", loggerName, "source", factSource)

	return h
}
