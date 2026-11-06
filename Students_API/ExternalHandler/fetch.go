package response

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"
)

const (
	maxErrBody  = 4 << 10  // cap on how much of an error response we read
	maxOKBody   = 64 << 10 // cap on how much of a success response we read
	clientLimit = 5 * time.Second

	defaultUserAgent = "catfact-client/1.0"
	defaultBackoff   = 200 * time.Millisecond
	maxBackoff       = 5 * time.Second
)

var (
	// ErrEmptyFact is returned when the API answers 200 but without a usable fact.
	ErrEmptyFact = errors.New("catfact: empty fact in response")

	// ErrFactTooLong is returned when a fact exceeds the configured rune limit.
	ErrFactTooLong = errors.New("catfact: fact exceeds maximum length")

	// ErrInvalidURL is returned when the fetcher was built with a bad URL.
	ErrInvalidURL = errors.New("catfact: invalid URL")
)

var (
	_ FactFetcher = (*catFactFetcher)(nil)
	_ FactFetcher = (*CachedFetcher)(nil)
	_ FactFetcher = (*FakeFetcher)(nil)
)

// FactFetcher retrieves a single fact. It is an interface so handlers can be
// tested with a fake instead of hitting the network.
type FactFetcher interface {
	Fetch(ctx context.Context) (string, error)
}

// ---------------------------------------------------------------------
// errors
// ---------------------------------------------------------------------

// StatusError is returned when the API answers with a non-200 status.
// Callers can use errors.As to inspect the code and body snippet.
type StatusError struct {
	Code       int
	Body       string
	RetryAfter time.Duration // zero if the server sent no usable Retry-After
}

func (e *StatusError) Error() string {
	if e.Body == "" {
		return fmt.Sprintf("catfact: unexpected status %d", e.Code)
	}
	return fmt.Sprintf("catfact: unexpected status %d: %s", e.Code, e.Body)
}

// Retryable reports whether trying again could plausibly succeed.
func (e *StatusError) Retryable() bool {
	return e.Code == http.StatusTooManyRequests || e.Code >= 500
}

// transportError marks failures that happened before a response arrived
// (DNS, connection reset, client timeout). These are worth retrying.
type transportError struct{ err error }

func (e *transportError) Error() string { return "catfact: request failed: " + e.err.Error() }
func (e *transportError) Unwrap() error { return e.err }

func isRetryable(err error) bool {
	var se *StatusError
	if errors.As(err, &se) {
		return se.Retryable()
	}
	var te *transportError
	return errors.As(err, &te)
}

type config struct {
	userAgent  string
	retries    int
	backoff    time.Duration
	maxFactLen int // in runes; 0 means unlimited
}

// Option customises a fetcher created by NewCatFactFetcher.
type Option func(*config)

// WithUserAgent sets the User-Agent header.
func WithUserAgent(ua string) Option {
	return func(c *config) {
		if strings.TrimSpace(ua) != "" {
			c.userAgent = ua
		}
	}
}

// WithRetries sets how many extra attempts are made after a retryable
// failure, with exponential backoff starting at base.
func WithRetries(n int, base time.Duration) Option {
	return func(c *config) {
		if n < 0 {
			n = 0
		}
		c.retries = n
		if base > 0 {
			c.backoff = base
		}
	}
}

// WithMaxFactLen rejects facts longer than n runes (0 disables the check).
func WithMaxFactLen(n int) Option {
	return func(c *config) {
		if n >= 0 {
			c.maxFactLen = n
		}
	}
}

// ---------------------------------------------------------------------
// HTTP fetcher
// ---------------------------------------------------------------------

type catFactFetcher struct {
	client *http.Client
	url    string
	urlErr error
	cfg    config
}

// NewCatFactFetcher returns a FactFetcher for the given URL
// (for example "https://catfact.ninja/fact").
// If client is nil, a client with a 5s timeout is used.
// An invalid URL is reported by Fetch as ErrInvalidURL.
func NewCatFactFetcher(client *http.Client, rawURL string, opts ...Option) FactFetcher {
	if client == nil {
		client = &http.Client{Timeout: clientLimit}
	}
	cfg := config{userAgent: defaultUserAgent, backoff: defaultBackoff}
	for _, opt := range opts {
		opt(&cfg)
	}

	f := &catFactFetcher{client: client, url: rawURL, cfg: cfg}
	u, err := url.Parse(rawURL)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		f.urlErr = fmt.Errorf("%w: %q", ErrInvalidURL, rawURL)
	}
	return f
}

// Fetch gets a fact, retrying transient failures according to the options.
func (f *catFactFetcher) Fetch(ctx context.Context) (string, error) {
	if f.urlErr != nil {
		return "", f.urlErr
	}

	var lastErr error
	for attempt := 0; attempt <= f.cfg.retries; attempt++ {
		fact, err := f.fetchOnce(ctx)
		if err == nil {
			return fact, nil
		}
		lastErr = err

		if attempt == f.cfg.retries || ctx.Err() != nil || !isRetryable(err) {
			break
		}
		if werr := sleepCtx(ctx, f.delay(attempt, err)); werr != nil {
			return "", werr
		}
	}
	return "", lastErr
}

func (f *catFactFetcher) fetchOnce(ctx context.Context) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, f.url, nil)
	if err != nil {
		return "", fmt.Errorf("catfact: build request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", f.cfg.userAgent)

	resp, err := f.client.Do(req)
	if err != nil {
		return "", &transportError{err: err}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// Read a bounded snippet for diagnostics. Reading it also lets the
		// connection be reused by the transport.
		snippet, _ := io.ReadAll(io.LimitReader(resp.Body, maxErrBody))
		return "", &StatusError{
			Code:       resp.StatusCode,
			Body:       strings.TrimSpace(string(snippet)),
			RetryAfter: parseRetryAfter(resp.Header.Get("Retry-After")),
		}
	}

	var payload struct {
		Fact string `json:"fact"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, maxOKBody)).Decode(&payload); err != nil {
		return "", fmt.Errorf("catfact: decode response: %w", err)
	}

	fact := strings.TrimSpace(payload.Fact)
	if fact == "" {
		return "", ErrEmptyFact
	}
	if max := f.cfg.maxFactLen; max > 0 && utf8.RuneCountInString(fact) > max {
		return "", ErrFactTooLong
	}
	return fact, nil
}

// delay picks how long to wait before the next attempt: the server's
// Retry-After if present, otherwise exponential backoff. Both are capped.
func (f *catFactFetcher) delay(attempt int, err error) time.Duration {
	var se *StatusError
	if errors.As(err, &se) && se.RetryAfter > 0 {
		return min(se.RetryAfter, maxBackoff)
	}
	d := f.cfg.backoff << uint(attempt)
	if d <= 0 || d > maxBackoff { // <= 0 guards against shift overflow
		return maxBackoff
	}
	return d
}

// parseRetryAfter understands the delta-seconds form of Retry-After.
// HTTP-date values are ignored (treated as "no hint").
func parseRetryAfter(v string) time.Duration {
	secs, err := strconv.Atoi(strings.TrimSpace(v))
	if err != nil || secs <= 0 {
		return 0
	}
	return time.Duration(secs) * time.Second
}

// sleepCtx waits for d or until ctx is done, whichever comes first.
func sleepCtx(ctx context.Context, d time.Duration) error {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}

// ---------------------------------------------------------------------
// caching decorator
// ---------------------------------------------------------------------

// CachedFetcher remembers the last successful fact for ttl. If a refresh
// fails, the stale fact is served instead of an error. The mutex is held
// during the refresh, which also prevents a burst of callers from all
// hitting the upstream API at once.
type CachedFetcher struct {
	next FactFetcher
	ttl  time.Duration
	now  func() time.Time // injectable for tests

	mu      sync.Mutex
	fact    string
	fetched time.Time
}

// NewCachedFetcher wraps next with a single-entry TTL cache.
func NewCachedFetcher(next FactFetcher, ttl time.Duration) *CachedFetcher {
	return &CachedFetcher{next: next, ttl: ttl, now: time.Now}
}

func (c *CachedFetcher) Fetch(ctx context.Context) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.fact != "" && c.now().Sub(c.fetched) < c.ttl {
		return c.fact, nil
	}

	fact, err := c.next.Fetch(ctx)
	if err != nil {
		if c.fact != "" {
			return c.fact, nil // serve stale rather than fail
		}
		return "", err
	}
	c.fact, c.fetched = fact, c.now()
	return fact, nil
}

// ---------------------------------------------------------------------
// test double
// ---------------------------------------------------------------------

// FakeFetcher is a FactFetcher for tests. It returns Fact (or Err if set)
// and counts how many times it was called.
type FakeFetcher struct {
	Fact string
	Err  error

	mu    sync.Mutex
	calls int
}

func (f *FakeFetcher) Fetch(ctx context.Context) (string, error) {
	f.mu.Lock()
	f.calls++
	f.mu.Unlock()

	if err := ctx.Err(); err != nil {
		return "", err
	}
	if f.Err != nil {
		return "", f.Err
	}
	return f.Fact, nil
}

// Calls reports how many times Fetch has been invoked.
func (f *FakeFetcher) Calls() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.calls
}
