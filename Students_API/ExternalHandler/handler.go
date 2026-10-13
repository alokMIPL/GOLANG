package response

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"runtime/debug"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/sync/singleflight"
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

var _ http.Handler = (*Handler)(nil)

// FactFetcher lets tests swap in a fake upstream.
type FactFetcher func(ctx context.Context) (*CatFactResponse, error)

// cacheEntry is immutable once published, so readers never need a lock.
type cacheEntry struct {
	fact      CatFactResponse
	fetchedAt time.Time
}

// failure records the last upstream error so a broken upstream is not
// hammered by every incoming request.
type failure struct {
	until time.Time
	err   error
}

// backoffError is returned when the upstream is failing, nothing usable is
// cached, and we are inside the backoff window.
type backoffError struct {
	until time.Time
	err   error
}

func (e *backoffError) Error() string { return "upstream in backoff: " + e.err.Error() }
func (e *backoffError) Unwrap() error { return e.err }

type cacheState uint8

const (
	stateHit cacheState = iota
	stateMiss
	stateStale
)

func (s cacheState) String() string {
	switch s {
	case stateHit:
		return "HIT"
	case stateStale:
		return "STALE"
	default:
		return "MISS"
	}
}

type result struct {
	fact  CatFactResponse
	state cacheState
	age   time.Duration
}

// Stats is a point-in-time snapshot of handler counters.
type Stats struct {
	Hits          uint64 // served from a fresh cache entry
	Misses        uint64 // had to wait for upstream
	StaleServed   uint64 // served an expired entry (fallback or background refresh)
	Refreshes     uint64 // upstream calls started
	RefreshErrors uint64 // upstream calls that failed
}

type counters struct {
	hits, misses, stale, refreshes, refreshErrors atomic.Uint64
}

// Handler serves a cat fact.
//
//   - A fact stays fresh for ttl.
//   - Concurrent cache misses share one upstream call (singleflight).
//   - Optionally, expired facts are served immediately while a single
//     background refresh runs (stale-while-revalidate).
//   - If a refresh fails, a fact up to ttl+maxStale old is served instead.
//   - After a failure, upstream is not retried for the backoff duration.
//   - Failed upstream calls can be retried with exponential delay.
type Handler struct {
	fetch      FactFetcher
	ttl        time.Duration
	maxStale   time.Duration
	timeout    time.Duration
	backoff    time.Duration
	retries    int
	retryDelay time.Duration
	bgRefresh  bool
	now        func() time.Time
	log        *slog.Logger

	group   singleflight.Group
	entry   atomic.Pointer[cacheEntry]
	failure atomic.Pointer[failure]
	stats   counters

	bgActive atomic.Bool
	bgMu     sync.Mutex // guards closed and wg.Add against Close
	closed   bool
	wg       sync.WaitGroup
}

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

// Prime warms the cache, e.g. at startup, so the first request is a hit.
func (h *Handler) Prime(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case res := <-h.flight(ctx):
		return res.Err
	}
}

// Close stops new background refreshes and waits for running ones, or for
// ctx to end. The handler can still serve requests afterwards.
func (h *Handler) Close(ctx context.Context) error {
	h.bgMu.Lock()
	h.closed = true
	h.bgMu.Unlock()

	done := make(chan struct{})
	go func() {
		h.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Stats returns a snapshot of the handler's counters.
func (h *Handler) Stats() Stats {
	return Stats{
		Hits:          h.stats.hits.Load(),
		Misses:        h.stats.misses.Load(),
		StaleServed:   h.stats.stale.Load(),
		Refreshes:     h.stats.refreshes.Load(),
		RefreshErrors: h.stats.refreshErrors.Load(),
	}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.Header().Set("Allow", "GET, HEAD")
		_ = WriteJSON(w, http.StatusMethodNotAllowed, apiResponse{
			OK:    false,
			Error: "method not allowed",
		})
		return
	}

	res, err := h.get(r.Context())
	if err != nil {
		h.writeError(w, err)
		return
	}

	h.setCacheHeaders(w, res)
	// net/http discards the body for HEAD, so no special casing is needed.
	_ = WriteJSON(w, http.StatusOK, apiResponse{
		OK:        true,
		Timestamp: h.now().UTC(),
		External: &externalFact{
			Source: factSource,
			Fact:   res.fact.Fact,
			Length: res.fact.Length,
		},
	})
}

func (h *Handler) setCacheHeaders(w http.ResponseWriter, res result) {
	age := max(res.age, 0)
	maxAge := max((h.ttl - age), 0)
	if res.state == stateStale {
		maxAge = 0
	}

	hdr := w.Header()
	hdr.Set("X-Cache", res.state.String())
	hdr.Set("Age", strconv.FormatInt(int64(age/time.Second), 10))
	hdr.Set("Cache-Control", "public, max-age="+strconv.FormatInt(int64(maxAge/time.Second), 10))
}

func (h *Handler) writeError(w http.ResponseWriter, err error) {
	var be *backoffError
	switch {
	case errors.As(err, &be):
		retry := int((be.until.Sub(h.now()) + time.Second - 1) / time.Second) // round up
		w.Header().Set("Retry-After", strconv.Itoa(max(retry, 1)))
		h.log.Warn("external: upstream in backoff", "err", be.err, "until", be.until)
		_ = WriteJSON(w, http.StatusServiceUnavailable, apiResponse{
			OK:    false,
			Error: "upstream temporarily unavailable",
		})
	case errors.Is(err, context.Canceled):
		// The client went away; there is nobody left to answer.
		h.log.Debug("external: client canceled request")
	case errors.Is(err, context.DeadlineExceeded):
		h.log.Warn("external: upstream timeout", "err", err)
		_ = WriteJSON(w, http.StatusGatewayTimeout, apiResponse{
			OK:    false,
			Error: "upstream timeout",
		})
	default:
		h.log.Error("external: upstream error", "err", err)
		_ = WriteJSON(w, http.StatusBadGateway, apiResponse{
			OK:    false,
			Error: "failed to fetch from upstream",
		})
	}
}

// get resolves a fact through the cache layers, in order:
// fresh hit -> background-refreshed stale -> backoff fallback -> upstream.
func (h *Handler) get(ctx context.Context) (result, error) {
	now := h.now() // read once so every decision below sees the same instant
	e := h.entry.Load()

	if e != nil {
		age := now.Sub(e.fetchedAt)
		if age < h.ttl {
			h.stats.hits.Add(1)
			return result{e.fact, stateHit, age}, nil
		}
		if h.bgRefresh && age < h.ttl+h.maxStale {
			h.refreshAsync(ctx)
			h.stats.stale.Add(1)
			return result{e.fact, stateStale, age}, nil
		}
	}

	h.stats.misses.Add(1)

	// Upstream recently failed: don't make this request pay for another try.
	if f := h.failure.Load(); f != nil && now.Before(f.until) {
		if res, ok := h.staleResult(e, now); ok {
			h.stats.stale.Add(1)
			return res, nil
		}
		return result{}, &backoffError{until: f.until, err: f.err}
	}

	select {
	case <-ctx.Done():
		return result{}, ctx.Err()
	case res := <-h.flight(ctx):
		if res.Err == nil {
			fresh := res.Val.(*cacheEntry)
			return result{fresh.fact, stateMiss, max(h.now().Sub(fresh.fetchedAt), 0)}, nil
		}
		if sr, ok := h.staleResult(h.entry.Load(), h.now()); ok {
			h.stats.stale.Add(1)
			h.log.Warn("external: serving stale fact", "err", res.Err, "age", sr.age)
			return sr, nil
		}
		return result{}, res.Err
	}
}

// staleResult returns e if it is still inside the stale window.
func (h *Handler) staleResult(e *cacheEntry, now time.Time) (result, bool) {
	if e == nil {
		return result{}, false
	}
	age := now.Sub(e.fetchedAt)
	if age >= h.ttl+h.maxStale {
		return result{}, false
	}
	return result{e.fact, stateStale, age}, true
}

// flight runs refresh through singleflight so concurrent callers share one
// upstream call.
func (h *Handler) flight(ctx context.Context) <-chan singleflight.Result {
	return h.group.DoChan(flightKey, func() (any, error) {
		// A flight may have finished between the caller's cache check and now.
		if e := h.entry.Load(); e != nil && h.now().Sub(e.fetchedAt) < h.ttl {
			return e, nil
		}
		return h.refresh(ctx)
	})
}

// refreshAsync starts at most one background refresh at a time.
func (h *Handler) refreshAsync(ctx context.Context) {
	if f := h.failure.Load(); f != nil && h.now().Before(f.until) {
		return
	}
	if !h.bgActive.CompareAndSwap(false, true) {
		return
	}

	h.bgMu.Lock()
	if h.closed {
		h.bgMu.Unlock()
		h.bgActive.Store(false)
		return
	}
	h.wg.Add(1)
	h.bgMu.Unlock()

	go func() {
		defer h.wg.Done()
		defer h.bgActive.Store(false)
		if res := <-h.flight(ctx); res.Err != nil {
			h.log.Warn("external: background refresh failed", "err", res.Err)
		}
	}()
}

// refresh calls upstream and publishes the result. It runs inside a
// singleflight, so it is detached from the triggering request's cancellation:
// one impatient client must not fail the fetch for everyone waiting on it.
// The timeout still bounds it, and context values (trace IDs) are kept.
func (h *Handler) refresh(ctx context.Context) (e *cacheEntry, err error) {
	h.stats.refreshes.Add(1)

	defer func() {
		if r := recover(); r != nil {
			h.log.Error("external: upstream fetch panicked",
				"panic", r, "stack", string(debug.Stack()))
			e, err = nil, fmt.Errorf("upstream fetch panicked: %v", r)
		}
		if err != nil {
			h.recordFailure(err)
		} else {
			h.failure.Store(nil)
		}
	}()

	fctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), h.timeout)
	defer cancel()

	got, err := h.fetchWithRetry(fctx)
	switch {
	case err != nil:
		return nil, fmt.Errorf("upstream fetch: %w", err)
	case got == nil:
		return nil, errors.New("upstream fetch: empty response")
	case got.Fact == "":
		return nil, errors.New("upstream fetch: response has no fact")
	}

	e = &cacheEntry{fact: *got, fetchedAt: h.now()}
	h.entry.Store(e)
	return e, nil
}

func (h *Handler) fetchWithRetry(ctx context.Context) (*CatFactResponse, error) {
	var lastErr error
	for attempt := 0; attempt <= h.retries; attempt++ {
		if attempt > 0 {
			if err := sleep(ctx, h.retryDelay<<(attempt-1)); err != nil {
				return nil, errors.Join(lastErr, err)
			}
		}

		got, err := h.fetch(ctx)
		if err == nil {
			return got, nil
		}
		lastErr = err
		if ctx.Err() != nil {
			break
		}
	}
	return nil, lastErr
}

func (h *Handler) recordFailure(err error) {
	h.stats.refreshErrors.Add(1)
	if h.backoff > 0 {
		h.failure.Store(&failure{until: h.now().Add(h.backoff), err: err})
	}
}

// sleep waits for d or until ctx ends.
func sleep(ctx context.Context, d time.Duration) error {
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
