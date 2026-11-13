package response

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math/rand/v2"
	"net/http"
	"strings"
	"sync"
	"time"
)

const (
	// DefaultCatFactURL is the public upstream used when no Fetcher is given.
	DefaultCatFactURL = "https://catfact.ninja/fact"

	maxUpstreamBody = 64 << 10 // never read more than 64 KiB from upstream
	maxErrorDrain   = 4 << 10  // drain a little of error bodies for connection reuse
	defaultClientTO = 5 * time.Second

	defaultMaxAttempts = 3
	defaultBaseBackoff = 100 * time.Millisecond
	defaultMaxBackoff  = 2 * time.Second
)

var (
	ErrUpstreamStatus = errors.New("unexpected upstream status")
	ErrInvalidFact    = errors.New("invalid cat fact payload")
)

// CatFactResponse mirrors the upstream JSON payload.
type CatFactResponse struct {
	Fact   string `json:"fact"`
	Length int    `json:"length"`
}

// Validate checks that the upstream payload is usable.
func (c *CatFactResponse) Validate() error {
	if strings.TrimSpace(c.Fact) == "" {
		return fmt.Errorf("%w: empty fact", ErrInvalidFact)
	}
	if c.Length < 0 {
		return fmt.Errorf("%w: negative length", ErrInvalidFact)
	}
	return nil
}

// Fetcher retrieves a cat fact.
type Fetcher func(ctx context.Context) (*CatFactResponse, error)

// ---------- HTTP fetcher with options ----------

type config struct {
	client      *http.Client
	url         string
	maxAttempts int
	baseBackoff time.Duration
	maxBackoff  time.Duration
	logger      *slog.Logger
	userAgent   string
}

// Option customises the HTTP fetcher.
type Option func(*config)

func WithClient(c *http.Client) Option { return func(cfg *config) { cfg.client = c } }
func WithURL(u string) Option          { return func(cfg *config) { cfg.url = u } }
func WithLogger(l *slog.Logger) Option { return func(cfg *config) { cfg.logger = l } }
func WithUserAgent(ua string) Option   { return func(cfg *config) { cfg.userAgent = ua } }
func WithMaxAttempts(n int) Option {
	return func(cfg *config) {
		if n > 0 {
			cfg.maxAttempts = n
		}
	}
}
func WithBackoff(base, max time.Duration) Option {
	return func(cfg *config) { cfg.baseBackoff, cfg.maxBackoff = base, max }
}

// NewHTTPFetcher builds a Fetcher that calls the upstream over HTTP,
// retrying transient failures with exponential backoff and jitter.
func NewHTTPFetcher(opts ...Option) Fetcher {
	cfg := &config{
		url:         DefaultCatFactURL,
		maxAttempts: defaultMaxAttempts,
		baseBackoff: defaultBaseBackoff,
		maxBackoff:  defaultMaxBackoff,
		logger:      slog.Default(),
		userAgent:   "catfact-client/1.0",
	}
	for _, o := range opts {
		o(cfg)
	}
	if cfg.client == nil {
		cfg.client = &http.Client{Timeout: defaultClientTO}
	}

	return func(ctx context.Context) (*CatFactResponse, error) {
		var lastErr error
		for attempt := 1; attempt <= cfg.maxAttempts; attempt++ {
			out, err := fetchOnce(ctx, cfg)
			if err == nil {
				return out, nil
			}
			lastErr = err

			if !retryable(err) || attempt == cfg.maxAttempts {
				break
			}
			wait := backoff(cfg.baseBackoff, cfg.maxBackoff, attempt)
			cfg.logger.WarnContext(ctx, "upstream fetch failed, retrying",
				"attempt", attempt, "wait", wait, "err", err)

			select {
			case <-time.After(wait):
			case <-ctx.Done():
				return nil, fmt.Errorf("retry aborted: %w", ctx.Err())
			}
		}
		return nil, lastErr
	}
}

func fetchOnce(ctx context.Context, cfg *config) (*CatFactResponse, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, cfg.url, nil)
	if err != nil {
		return nil, fmt.Errorf("build upstream request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", cfg.userAgent)

	resp, err := cfg.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("call upstream: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, maxErrorDrain))
		return nil, &statusError{code: resp.StatusCode, status: resp.Status}
	}

	var out CatFactResponse
	if err := json.NewDecoder(io.LimitReader(resp.Body, maxUpstreamBody)).Decode(&out); err != nil {
		return nil, fmt.Errorf("decode upstream body: %w", err)
	}
	if err := out.Validate(); err != nil {
		return nil, err
	}
	return &out, nil
}

// statusError wraps ErrUpstreamStatus while keeping the status code.
type statusError struct {
	code   int
	status string
}

func (e *statusError) Error() string { return fmt.Sprintf("%v: %s", ErrUpstreamStatus, e.status) }
func (e *statusError) Unwrap() error { return ErrUpstreamStatus }

// retryable reports whether an error is worth retrying.
func retryable(err error) bool {
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}
	if errors.Is(err, ErrInvalidFact) {
		return false
	}
	var se *statusError
	if errors.As(err, &se) {
		return se.code == http.StatusTooManyRequests || se.code >= 500
	}
	return true // network errors, truncated bodies, etc.
}

// backoff returns exponential backoff with full jitter.
func backoff(base, max time.Duration, attempt int) time.Duration {
	d := base << (attempt - 1)
	if d > max || d <= 0 {
		d = max
	}
	return time.Duration(rand.Int64N(int64(d)) + 1)
}

// ---------- Middleware ----------

// WithCache wraps a Fetcher and caches successful results for ttl.
// Concurrent callers share a single in-flight refresh.
func WithCache(next Fetcher, ttl time.Duration) Fetcher {
	var (
		mu      sync.Mutex
		cached  *CatFactResponse
		expires time.Time
	)
	return func(ctx context.Context) (*CatFactResponse, error) {
		mu.Lock()
		defer mu.Unlock()

		if cached != nil && time.Now().Before(expires) {
			c := *cached
			return &c, nil
		}
		out, err := next(ctx)
		if err != nil {
			// Serve stale data rather than fail, if we have any.
			if cached != nil {
				c := *cached
				return &c, nil
			}
			return nil, err
		}
		cached, expires = out, time.Now().Add(ttl)
		c := *out
		return &c, nil
	}
}

// ---------- HTTP handler ----------

// Handler exposes a Fetcher as an HTTP endpoint.
func Handler(f Fetcher, logger *slog.Logger) http.Handler {
	if logger == nil {
		logger = slog.Default()
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		fact, err := f(ctx)
		if err != nil {
			logger.ErrorContext(ctx, "fetch cat fact", "err", err)
			status := http.StatusBadGateway
			if errors.Is(err, context.DeadlineExceeded) {
				status = http.StatusGatewayTimeout
			}
			writeJSON(w, status, map[string]string{"error": "upstream unavailable"})
			return
		}
		writeJSON(w, http.StatusOK, fact)
	})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
