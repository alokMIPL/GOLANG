package response

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"
)

const (
	defaultURL       = "https://catfact.ninja/fact"
	upstreamTimeout  = 5 * time.Second
	maxUpstreamBody  = 1 << 20 // 1 MiB
	defaultUA        = "catfact-fetcher/1.0"
	maxStatusSnippet = 256
)

// Sentinel errors let callers branch with errors.Is instead of matching strings.
var (
	ErrBuildRequest   = errors.New("build request failed")
	ErrRequestFailed  = errors.New("upstream request failed")
	ErrUpstreamStatus = errors.New("unexpected upstream status")
	ErrBodyTooLarge   = errors.New("upstream body too large")
	ErrDecode         = errors.New("decode response failed")
	ErrEmptyFact      = errors.New("upstream returned no fact")
	ErrLengthMismatch = errors.New("upstream length does not match fact")
)

// StatusError carries the upstream HTTP status and a short body snippet,
// which makes debugging failing upstream calls much easier.
type StatusError struct {
	StatusCode int
	Snippet    string
}

func (e *StatusError) Error() string {
	if e.Snippet == "" {
		return fmt.Sprintf("%v: %d", ErrUpstreamStatus, e.StatusCode)
	}
	return fmt.Sprintf("%v: %d (%s)", ErrUpstreamStatus, e.StatusCode, e.Snippet)
}

// Unwrap allows errors.Is(err, ErrUpstreamStatus).
func (e *StatusError) Unwrap() error { return ErrUpstreamStatus }

// Retryable reports whether the status is worth retrying.
func (e *StatusError) Retryable() bool {
	return e.StatusCode == http.StatusTooManyRequests || e.StatusCode >= 500
}

// CatFactResponse is the payload returned by the upstream API.
type CatFactResponse struct {
	Fact   string `json:"fact"`
	Length int    `json:"length"`
}

// Validate checks the response for basic sanity and normalizes it in place.
func (r *CatFactResponse) Validate() error {
	r.Fact = strings.TrimSpace(r.Fact)
	if r.Fact == "" {
		return ErrEmptyFact
	}
	if !utf8.ValidString(r.Fact) {
		return fmt.Errorf("%w: fact is not valid UTF-8", ErrDecode)
	}
	if r.Length == 0 {
		r.Length = utf8.RuneCountInString(r.Fact)
	}
	return nil
}

// FactFetcher fetches a single cat fact.
type FactFetcher func(ctx context.Context) (*CatFactResponse, error)

// Logger is a minimal logging hook so callers can plug in their own logger.
type Logger interface {
	Printf(format string, args ...any)
}

type noopLogger struct{}

func (noopLogger) Printf(string, ...any) {}

// config holds the tunable settings for a fetcher.
type config struct {
	client       *http.Client
	url          string
	userAgent    string
	maxBody      int64
	maxAttempts  int
	baseBackoff  time.Duration
	maxBackoff   time.Duration
	logger       Logger
	strictLength bool
}

// Option customizes a fetcher.
type Option func(*config)

// WithUserAgent overrides the User-Agent header.
func WithUserAgent(ua string) Option {
	return func(c *config) {
		if ua != "" {
			c.userAgent = ua
		}
	}
}

// WithMaxBody overrides the maximum number of body bytes accepted.
func WithMaxBody(n int64) Option {
	return func(c *config) {
		if n > 0 {
			c.maxBody = n
		}
	}
}

// WithRetries enables retries on transport errors, 429 and 5xx responses.
// attempts is the total number of tries (1 means no retries).
func WithRetries(attempts int, baseBackoff, maxBackoff time.Duration) Option {
	return func(c *config) {
		if attempts > 0 {
			c.maxAttempts = attempts
		}
		if baseBackoff > 0 {
			c.baseBackoff = baseBackoff
		}
		if maxBackoff > 0 {
			c.maxBackoff = maxBackoff
		}
	}
}

// WithLogger sets a logger for debug output.
func WithLogger(l Logger) Option {
	return func(c *config) {
		if l != nil {
			c.logger = l
		}
	}
}

// WithStrictLength makes the fetcher fail when the upstream length field
// disagrees with the rune count of the fact.
func WithStrictLength(strict bool) Option {
	return func(c *config) { c.strictLength = strict }
}

// NewCatFactFetcher returns a FactFetcher backed by the given client and URL.
// A nil client gets a default one with a timeout as a safety net; the
// per-call deadline normally comes from the context.
func NewCatFactFetcher(client *http.Client, url string) FactFetcher {
	return NewCatFactFetcherWithOptions(client, url)
}

// NewCatFactFetcherWithOptions is like NewCatFactFetcher but accepts options.
func NewCatFactFetcherWithOptions(client *http.Client, url string, opts ...Option) FactFetcher {
	cfg := &config{
		client:      client,
		url:         url,
		userAgent:   defaultUA,
		maxBody:     maxUpstreamBody,
		maxAttempts: 1,
		baseBackoff: 100 * time.Millisecond,
		maxBackoff:  2 * time.Second,
		logger:      noopLogger{},
	}
	for _, opt := range opts {
		opt(cfg)
	}
	if cfg.client == nil {
		cfg.client = &http.Client{Timeout: upstreamTimeout}
	}
	if cfg.url == "" {
		cfg.url = defaultURL
	}

	return func(ctx context.Context) (*CatFactResponse, error) {
		var lastErr error
		for attempt := 1; attempt <= cfg.maxAttempts; attempt++ {
			fact, err := fetchOnce(ctx, cfg)
			if err == nil {
				return fact, nil
			}
			lastErr = err

			if attempt == cfg.maxAttempts || !shouldRetry(ctx, err) {
				break
			}

			delay := backoff(cfg.baseBackoff, cfg.maxBackoff, attempt)
			cfg.logger.Printf("catfact: attempt %d/%d failed: %v; retrying in %s",
				attempt, cfg.maxAttempts, err, delay)

			if err := sleep(ctx, delay); err != nil {
				return nil, fmt.Errorf("retry wait interrupted: %w", err)
			}
		}
		return nil, lastErr
	}
}

// fetchOnce performs a single request/decode/validate cycle.
func fetchOnce(ctx context.Context, cfg *config) (*CatFactResponse, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, cfg.url, nil)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrBuildRequest, err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", cfg.userAgent)

	resp, err := cfg.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrRequestFailed, err)
	}
	defer resp.Body.Close()

	// Read one byte past the limit so we can tell "exactly at limit" from
	// "truncated".
	data, err := io.ReadAll(io.LimitReader(resp.Body, cfg.maxBody+1))
	if err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}
	if int64(len(data)) > cfg.maxBody {
		return nil, ErrBodyTooLarge
	}

	if resp.StatusCode != http.StatusOK {
		return nil, &StatusError{
			StatusCode: resp.StatusCode,
			Snippet:    snippet(data),
		}
	}

	var fact CatFactResponse
	if err := json.Unmarshal(data, &fact); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrDecode, err)
	}

	upstreamLength := fact.Length
	if err := fact.Validate(); err != nil {
		return nil, err
	}
	if cfg.strictLength && upstreamLength != 0 && upstreamLength != fact.Length {
		// Validate only fills Length when it is zero, so compare against the
		// real rune count here.
		if real := utf8.RuneCountInString(fact.Fact); upstreamLength != real {
			return nil, fmt.Errorf("%w: upstream=%d actual=%d",
				ErrLengthMismatch, upstreamLength, real)
		}
	}
	return &fact, nil
}

// shouldRetry decides whether an error is transient.
func shouldRetry(ctx context.Context, err error) bool {
	if ctx.Err() != nil {
		return false
	}
	var se *StatusError
	if errors.As(err, &se) {
		return se.Retryable()
	}
	if errors.Is(err, ErrRequestFailed) {
		return !errors.Is(err, context.Canceled)
	}
	return false
}

// backoff returns an exponential delay capped at max.
func backoff(base, max time.Duration, attempt int) time.Duration {
	d := base << (attempt - 1)
	if d <= 0 || d > max {
		return max
	}
	return d
}

// sleep waits for d or until ctx is done.
func sleep(ctx context.Context, d time.Duration) error {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}

// snippet returns a short, single-line, printable excerpt of a body.
func snippet(b []byte) string {
	s := strings.TrimSpace(string(b))
	s = strings.Join(strings.Fields(s), " ")
	if len(s) > maxStatusSnippet {
		s = s[:maxStatusSnippet] + "..."
	}
	return strings.ToValidUTF8(s, "")
}
