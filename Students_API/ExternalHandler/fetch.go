package response

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand/v2"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
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
	ErrEmptyFact   = errors.New("catfact: empty fact in response")
	ErrFactTooLong = errors.New("catfact: fact exceeds maximum length")
	ErrInvalidURL  = errors.New("catfact: invalid URL")
	ErrCircuitOpen = errors.New("catfact: circuit breaker is open")
)

var (
	_ FactFetcher = (*catFactFetcher)(nil)
	_ FactFetcher = (*CachedFetcher)(nil)
	_ FactFetcher = (*CircuitBreakerFetcher)(nil)
	_ FactFetcher = (*InstrumentedFetcher)(nil)
	_ FactFetcher = (*FakeFetcher)(nil)
)

type FactFetcher interface {
	Fetch(ctx context.Context) (string, error)
}

type StatusError struct {
	Code       int
	Body       string
	RetryAfter time.Duration
}

func (e *StatusError) Error() string {
	if e.Body == "" {
		return fmt.Sprintf("catfact: unexpected status %d", e.Code)
	}
	return fmt.Sprintf("catfact: unexpected status %d: %s", e.Code, e.Body)
}

func (e *StatusError) Retryable() bool {
	return e.Code == http.StatusTooManyRequests ||
		e.Code == http.StatusRequestTimeout ||
		e.Code >= 500
}

type transportError struct{ err error }

func (e *transportError) Error() string { return "catfact: request failed: " + e.err.Error() }
func (e *transportError) Unwrap() error { return e.err }

func isRetryable(err error) bool {
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}
	var se *StatusError
	if errors.As(err, &se) {
		return se.Retryable()
	}
	var te *transportError
	return errors.As(err, &te)
}

type config struct {
	userAgent   string
	retries     int
	backoff     time.Duration
	maxBackoff  time.Duration
	jitter      bool
	maxFactLen  int // in runes; 0 means unlimited
	headers     http.Header
	now         func() time.Time
	randFloat64 func() float64
}

type Option func(*config)

func WithUserAgent(ua string) Option {
	return func(c *config) {
		if strings.TrimSpace(ua) != "" {
			c.userAgent = ua
		}
	}
}

func WithHeader(key, value string) Option {
	return func(c *config) {
		if strings.TrimSpace(key) != "" {
			if c.headers == nil {
				c.headers = make(http.Header)
			}
			c.headers.Set(key, value)
		}
	}
}

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

func WithMaxBackoff(d time.Duration) Option {
	return func(c *config) {
		if d > 0 {
			c.maxBackoff = d
		}
	}
}

func WithJitter(enabled bool) Option {
	return func(c *config) {
		c.jitter = enabled
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

// NewCatFactFetcher builds an HTTP-backed FactFetcher with retry and validation support.
func NewCatFactFetcher(client *http.Client, rawURL string, opts ...Option) FactFetcher {
	if client == nil {
		client = &http.Client{Timeout: clientLimit}
	}
	cfg := config{
		userAgent:   defaultUserAgent,
		backoff:     defaultBackoff,
		maxBackoff:  maxBackoff,
		jitter:      true,
		now:         time.Now,
		randFloat64: rand.Float64,
	}
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

func (f *catFactFetcher) Fetch(ctx context.Context) (string, error) {
	if f.urlErr != nil {
		return "", f.urlErr
	}
	if err := ctx.Err(); err != nil {
		return "", err
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
	for k, vals := range f.cfg.headers {
		for _, v := range vals {
			req.Header.Add(k, v)
		}
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", f.cfg.userAgent)

	resp, err := f.client.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return "", ctx.Err()
		}
		return "", &transportError{err: err}
	}
	defer func() {
		// Drain any unread bytes up to maxErrBody so net/http can reuse the TCP connection.
		_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, maxErrBody))
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		snippet, _ := io.ReadAll(io.LimitReader(resp.Body, maxErrBody))
		return "", &StatusError{
			Code:       resp.StatusCode,
			Body:       strings.TrimSpace(string(snippet)),
			RetryAfter: parseRetryAfter(resp.Header.Get("Retry-After"), f.cfg.now()),
		}
	}

	var payload struct {
		Fact   string `json:"fact"`
		Length int    `json:"length"`
	}
	dec := json.NewDecoder(io.LimitReader(resp.Body, maxOKBody))
	if err := dec.Decode(&payload); err != nil {
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
// Retry-After if present, otherwise exponential backoff (with optional full jitter).
func (f *catFactFetcher) delay(attempt int, err error) time.Duration {
	var se *StatusError
	if errors.As(err, &se) && se.RetryAfter > 0 {
		return min(se.RetryAfter, f.cfg.maxBackoff)
	}
	d := f.cfg.backoff << uint(attempt)
	if d <= 0 || d > f.cfg.maxBackoff { // <= 0 guards against shift overflow
		d = f.cfg.maxBackoff
	}
	if f.cfg.jitter && d > 0 {
		// Full jitter: random value in [0, d]
		return time.Duration(f.cfg.randFloat64() * float64(d))
	}
	return d
}

// parseRetryAfter supports both delta-seconds and HTTP-date forms of Retry-After.
func parseRetryAfter(v string, now time.Time) time.Duration {
	v = strings.TrimSpace(v)
	if v == "" {
		return 0
	}
	if secs, err := strconv.Atoi(v); err == nil {
		if secs <= 0 {
			return 0
		}
		return time.Duration(secs) * time.Second
	}
	if t, err := http.ParseTime(v); err == nil {
		if d := t.Sub(now); d > 0 {
			return d
		}
	}
	return 0
}

// sleepCtx waits for d or until ctx is done, whichever comes first.
func sleepCtx(ctx context.Context, d time.Duration) error {
	if d <= 0 {
		return ctx.Err()
	}
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
// caching decorator (with singleflight & stale-if-error)
// ---------------------------------------------------------------------

type inflightCall struct {
	wg   sync.WaitGroup
	fact string
	err  error
}

// CachedFetcher remembers the last successful fact for ttl.
// Concurrent callers arriving when the cache is expired share a single
// upstream flight (singleflight), and if the refresh fails, a previously
// cached fact (within maxStale, or forever if maxStale == 0) is returned.
type CachedFetcher struct {
	next     FactFetcher
	ttl      time.Duration
	maxStale time.Duration
	now      func() time.Time // injectable for tests

	mu       sync.Mutex
	fact     string
	fetched  time.Time
	inflight *inflightCall
}

// NewCachedFetcher wraps next with a single-entry TTL cache.
func NewCachedFetcher(next FactFetcher, ttl time.Duration) *CachedFetcher {
	return &CachedFetcher{
		next: next,
		ttl:  ttl,
		now:  time.Now,
	}
}

// WithMaxStale limits how old a cached entry can be when serving stale on error.
// A zero duration (the default) allows serving stale entries indefinitely on error.
func (c *CachedFetcher) WithMaxStale(d time.Duration) *CachedFetcher {
	c.mu.Lock()
	defer c.mu.Unlock()
	if d >= 0 {
		c.maxStale = d
	}
	return c
}

// Invalidate clears the cached fact so the next Fetch hits upstream.
func (c *CachedFetcher) Invalidate() {
	c.mu.Lock()
	c.fact = ""
	c.fetched = time.Time{}
	c.mu.Unlock()
}

func (c *CachedFetcher) Fetch(ctx context.Context) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}

	c.mu.Lock()
	now := c.now()
	if c.fact != "" && now.Sub(c.fetched) < c.ttl {
		fact := c.fact
		c.mu.Unlock()
		return fact, nil
	}

	// Join an already-running refresh if one exists.
	if call := c.inflight; call != nil {
		c.mu.Unlock()
		done := make(chan struct{})
		go func() {
			call.wg.Wait()
			close(done)
		}()
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-done:
			return call.fact, call.err
		}
	}

	// Start a new singleflight refresh.
	call := &inflightCall{}
	call.wg.Add(1)
	c.inflight = call
	c.mu.Unlock()

	fact, err := c.next.Fetch(ctx)

	c.mu.Lock()
	c.inflight = nil
	if err == nil {
		c.fact = fact
		c.fetched = c.now()
		call.fact = fact
	} else if c.canServeStaleLocked(c.now()) {
		call.fact = c.fact
		call.err = nil
	} else {
		call.err = err
	}
	c.mu.Unlock()

	call.wg.Done()
	return call.fact, call.err
}

func (c *CachedFetcher) canServeStaleLocked(now time.Time) bool {
	if c.fact == "" {
		return false
	}
	if c.maxStale == 0 {
		return true
	}
	return now.Sub(c.fetched) <= c.ttl+c.maxStale
}

// ---------------------------------------------------------------------
// circuit breaker decorator
// ---------------------------------------------------------------------

// CircuitState represents the current state of a CircuitBreakerFetcher.
type CircuitState int

const (
	CircuitClosed CircuitState = iota
	CircuitOpen
	CircuitHalfOpen
)

func (s CircuitState) String() string {
	switch s {
	case CircuitClosed:
		return "closed"
	case CircuitOpen:
		return "open"
	case CircuitHalfOpen:
		return "half-open"
	default:
		return "unknown"
	}
}

// CircuitBreakerFetcher stops forwarding requests to next after threshold
// consecutive failures, failing fast with ErrCircuitOpen until cooldown elapses.
type CircuitBreakerFetcher struct {
	next      FactFetcher
	threshold int
	cooldown  time.Duration
	now       func() time.Time

	mu          sync.Mutex
	state       CircuitState
	failures    int
	openedAt    time.Time
	halfOpenRun bool
}

// NewCircuitBreakerFetcher wraps next with a circuit breaker.
func NewCircuitBreakerFetcher(next FactFetcher, failureThreshold int, cooldown time.Duration) *CircuitBreakerFetcher {
	if failureThreshold < 1 {
		failureThreshold = 1
	}
	if cooldown <= 0 {
		cooldown = 10 * time.Second
	}
	return &CircuitBreakerFetcher{
		next:      next,
		threshold: failureThreshold,
		cooldown:  cooldown,
		now:       time.Now,
	}
}

// State returns the breaker's current CircuitState.
func (cb *CircuitBreakerFetcher) State() CircuitState {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	if cb.state == CircuitOpen && cb.now().Sub(cb.openedAt) >= cb.cooldown {
		return CircuitHalfOpen
	}
	return cb.state
}

func (cb *CircuitBreakerFetcher) Fetch(ctx context.Context) (string, error) {
	cb.mu.Lock()
	now := cb.now()
	switch cb.state {
	case CircuitOpen:
		if now.Sub(cb.openedAt) < cb.cooldown {
			cb.mu.Unlock()
			return "", ErrCircuitOpen
		}
		cb.state = CircuitHalfOpen
		cb.halfOpenRun = true
	case CircuitHalfOpen:
		if cb.halfOpenRun {
			cb.mu.Unlock()
			return "", ErrCircuitOpen
		}
		cb.halfOpenRun = true
	}
	cb.mu.Unlock()

	fact, err := cb.next.Fetch(ctx)

	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.halfOpenRun = false

	if err == nil {
		cb.failures = 0
		cb.state = CircuitClosed
		return fact, nil
	}

	// Do not penalize upstream for caller cancellations.
	if errors.Is(err, context.Canceled) {
		return "", err
	}

	cb.failures++
	if cb.state == CircuitHalfOpen || cb.failures >= cb.threshold {
		cb.state = CircuitOpen
		cb.openedAt = cb.now()
	}
	return "", err
}

// ---------------------------------------------------------------------
// observability / metrics decorator
// ---------------------------------------------------------------------

// MetricsSnapshot is a point-in-time view of fetcher counters.
type MetricsSnapshot struct {
	Requests      uint64
	Successes     uint64
	Failures      uint64
	TotalDuration time.Duration
}

// AvgLatency returns the mean request duration across completed requests.
func (m MetricsSnapshot) AvgLatency() time.Duration {
	if m.Requests == 0 {
		return 0
	}
	return m.TotalDuration / time.Duration(m.Requests)
}

// InstrumentedFetcher records atomic request counts, outcomes, and latency.
type InstrumentedFetcher struct {
	next      FactFetcher
	requests  atomic.Uint64
	successes atomic.Uint64
	failures  atomic.Uint64
	totalNs   atomic.Int64
}

// NewInstrumentedFetcher wraps next with lightweight atomic telemetry.
func NewInstrumentedFetcher(next FactFetcher) *InstrumentedFetcher {
	return &InstrumentedFetcher{next: next}
}

func (m *InstrumentedFetcher) Fetch(ctx context.Context) (string, error) {
	start := time.Now()
	fact, err := m.next.Fetch(ctx)
	elapsed := time.Since(start)

	m.requests.Add(1)
	m.totalNs.Add(int64(elapsed))
	if err != nil {
		m.failures.Add(1)
	} else {
		m.successes.Add(1)
	}
	return fact, err
}

// Snapshot returns current counters safely across goroutines.
func (m *InstrumentedFetcher) Snapshot() MetricsSnapshot {
	return MetricsSnapshot{
		Requests:      m.requests.Load(),
		Successes:     m.successes.Load(),
		Failures:      m.failures.Load(),
		TotalDuration: time.Duration(m.totalNs.Load()),
	}
}

// ---------------------------------------------------------------------
// test double
// ---------------------------------------------------------------------

// FakeResult represents one scripted outcome for FakeFetcher.Sequence.
type FakeResult struct {
	Fact string
	Err  error
}

// FakeFetcher is a FactFetcher for tests. It supports static responses,
// scripted call sequences, artificial latency, and custom callbacks.
type FakeFetcher struct {
	Fact     string
	Err      error
	Delay    time.Duration
	Sequence []FakeResult
	Fn       func(ctx context.Context, callNum int) (string, error)

	mu    sync.Mutex
	calls int
}

func (f *FakeFetcher) Fetch(ctx context.Context) (string, error) {
	f.mu.Lock()
	f.calls++
	callNum := f.calls
	var seqItem *FakeResult
	if len(f.Sequence) > 0 {
		idx := min(callNum-1, len(f.Sequence)-1)
		item := f.Sequence[idx]
		seqItem = &item
	}
	f.mu.Unlock()

	if f.Delay > 0 {
		if err := sleepCtx(ctx, f.Delay); err != nil {
			return "", err
		}
	}
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if f.Fn != nil {
		return f.Fn(ctx, callNum)
	}
	if seqItem != nil {
		return seqItem.Fact, seqItem.Err
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

// Reset clears the recorded call count.
func (f *FakeFetcher) Reset() {
	f.mu.Lock()
	f.calls = 0
	f.mu.Unlock()
}
