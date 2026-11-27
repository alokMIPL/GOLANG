package response

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"sync"
	"sync/atomic"
	"time"
)

const (
	defaultFactURL   = "https://catfact.ninja/fact"
	factSource       = "catfact.ninja"
	defaultTTL       = 30 * time.Second
	defaultMaxStale  = 2 * time.Minute
	defaultTimeout   = 5 * time.Second
	maxBodyBytes     = 1 << 20 // 1 MiB safety cap on upstream payloads
)

// CatFactResponse mirrors the JSON payload returned by catfact.ninja/fact.
type CatFactResponse struct {
	Fact   string `json:"fact"`
	Length int    `json:"length"`
}

// ExternalPayload is the normalized response payload returned to API callers.
type ExternalPayload struct {
	Fact      string    `json:"fact"`
	Length    int       `json:"length"`
	Source    string    `json:"source"`
	FetchedAt time.Time `json:"fetched_at"`
}

// apiResponse is the envelope serialized as JSON for both success and error responses.
type apiResponse struct {
	OK       bool             `json:"ok"`
	External *ExternalPayload `json:"external,omitempty"`
	Error    string           `json:"error,omitempty"`
}

// Stats exposes atomic cache and upstream call counters for observability.
type Stats struct {
	Hits      uint64 `json:"hits"`
	Misses    uint64 `json:"misses"`
	StaleHits uint64 `json:"stale_hits"`
	Refreshes uint64 `json:"refreshes"`
	Errors    uint64 `json:"errors"`
}

// FactFetcher abstracts the upstream HTTP call so tests can inject deterministic stubs.
type FactFetcher func(ctx context.Context) (*CatFactResponse, error)

// Option configures a Handler instance.
type Option func(*Handler)

func WithFetcher(fn FactFetcher) Option {
	return func(h *Handler) {
		if fn != nil {
			h.fetcher = fn
		}
	}
}

func WithHTTPClient(client *http.Client, endpoint string) Option {
	return func(h *Handler) {
		if client == nil {
			client = &http.Client{Timeout: defaultTimeout}
		}
		if endpoint == "" {
			endpoint = defaultFactURL
		}
		h.fetcher = newHTTPFactFetcher(client, endpoint)
	}
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
			h.logger = l
		}
	}
}

func WithTTL(ttl time.Duration) Option {
	return func(h *Handler) {
		if ttl > 0 {
			h.ttl = ttl
		}
	}
}

func WithMaxStale(maxStale time.Duration) Option {
	return func(h *Handler) {
		if maxStale >= 0 {
			h.maxStale = maxStale
		}
	}
}

func WithBackoff(backoff time.Duration) Option {
	return func(h *Handler) {
		if backoff >= 0 {
			h.backoff = backoff
		}
	}
}

func WithRetries(retries int, delay time.Duration) Option {
	return func(h *Handler) {
		if retries >= 0 {
			h.retries = retries
		}
		if delay >= 0 {
			h.retryDelay = delay
		}
	}
}

func WithBackgroundRefresh(enabled bool) Option {
	return func(h *Handler) {
		h.bgRefresh = enabled
	}
}

func WithFetchTimeout(timeout time.Duration) Option {
	return func(h *Handler) {
		if timeout > 0 {
			h.fetchTimeout = timeout
		}
	}
}

type cacheEntry struct {
	payload   ExternalPayload
	fetchedAt time.Time
}

type flightCall struct {
	wg      sync.WaitGroup
	payload ExternalPayload
	err     error
}

// Handler serves cached cat facts with single-flight coalescing, stale-while-revalidate,
// circuit-breaker backoff, retries, and graceful shutdown.
type Handler struct {
	fetcher      FactFetcher
	now          func() time.Time
	logger       *slog.Logger
	ttl          time.Duration
	maxStale     time.Duration
	backoff      time.Duration
	retries      int
	retryDelay   time.Duration
	bgRefresh    bool
	fetchTimeout time.Duration

	mu          sync.RWMutex
	entry       *cacheEntry
	lastFailure time.Time
	inFlight    *flightCall
	bgRunning   bool

	bgWG   sync.WaitGroup
	closed atomic.Bool

	hits      atomic.Uint64
	misses    atomic.Uint64
	staleHits atomic.Uint64
	refreshes atomic.Uint64
	errCount  atomic.Uint64
}

// NewHandler constructs a configured Handler ready to mount on an http.ServeMux.
func NewHandler(opts ...Option) *Handler {
	h := &Handler{
		fetcher:      newHTTPFactFetcher(&http.Client{Timeout: defaultTimeout}, defaultFactURL),
		now:          time.Now,
		logger:       slog.Default(),
		ttl:          defaultTTL,
		maxStale:     defaultMaxStale,
		fetchTimeout: defaultTimeout,
	}
	for _, opt := range opts {
		opt(h)
	}
	return h
}

func newHTTPFactFetcher(client *http.Client, endpoint string) FactFetcher {
	return func(ctx context.Context) (*CatFactResponse, error) {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
		if err != nil {
			return nil, fmt.Errorf("build upstream request: %w", err)
		}
		req.Header.Set("Accept", "application/json")

		resp, err := client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("execute upstream request: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("upstream returned status %d", resp.StatusCode)
		}

		limited := io.LimitReader(resp.Body, maxBodyBytes)
		var out CatFactResponse
		if err := json.NewDecoder(limited).Decode(&out); err != nil {
			return nil, fmt.Errorf("decode upstream json: %w", err)
		}
		return &out, nil
	}
}

// Stats returns a point-in-time snapshot of handler telemetry.
func (h *Handler) Stats() Stats {
	return Stats{
		Hits:      h.hits.Load(),
		Misses:    h.misses.Load(),
		StaleHits: h.staleHits.Load(),
		Refreshes: h.refreshes.Load(),
		Errors:    h.errCount.Load(),
	}
}

// Prime pre-warms the cache before accepting live traffic.
func (h *Handler) Prime(ctx context.Context) error {
	_, err := h.refreshSingleFlight(ctx, false)
	return err
}

// Close waits for any in-flight background refreshes to complete or until ctx expires.
func (h *Handler) Close(ctx context.Context) error {
	h.closed.Store(true)
	done := make(chan struct{})
	go func() {
		h.bgWG.Wait()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// ServeHTTP handles GET and HEAD requests to /external.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.Header().Set("Allow", "GET, HEAD")
		h.writeJSON(w, r.Method, http.StatusMethodNotAllowed, apiResponse{
			OK:    false,
			Error: fmt.Sprintf("method %s not allowed", r.Method),
		})
		return
	}

	now := h.now()

	h.mu.RLock()
	entry := h.entry
	lastFailure := h.lastFailure
	h.mu.RUnlock()

	// 1. Fresh cache hit.
	if entry != nil {
		age := now.Sub(entry.fetchedAt)
		if age < 0 {
			age = 0
		}
		if age < h.ttl {
			h.hits.Add(1)
			remaining := int((h.ttl - age).Seconds())
			if remaining < 0 {
				remaining = 0
			}
			w.Header().Set("X-Cache", "HIT")
			w.Header().Set("Age", strconv.Itoa(int(age.Seconds())))
			w.Header().Set("Cache-Control", fmt.Sprintf("public, max-age=%d", remaining))
			h.writeJSON(w, r.Method, http.StatusOK, apiResponse{
				OK:       true,
				External: &entry.payload,
			})
			return
		}

		// 2. Stale-while-revalidate when background refresh is enabled and entry is within maxStale.
		if h.bgRefresh && age <= h.ttl+h.maxStale {
			h.staleHits.Add(1)
			h.triggerBackgroundRefresh()
			w.Header().Set("X-Cache", "STALE")
			w.Header().Set("Age", strconv.Itoa(int(age.Seconds())))
			w.Header().Set("Cache-Control", "public, max-age=0")
			h.writeJSON(w, r.Method, http.StatusOK, apiResponse{
				OK:       true,
				External: &entry.payload,
			})
			return
		}
	}

	// 3. Circuit-breaker backoff window when no fresh entry exists.
	if h.backoff > 0 && !lastFailure.IsZero() {
		elapsed := now.Sub(lastFailure)
		if elapsed < h.backoff {
			// If we have a valid stale entry within maxStale, serve it instead of failing.
			if entry != nil && now.Sub(entry.fetchedAt) <= h.ttl+h.maxStale {
				h.serveStale(w, r.Method, entry, now)
				return
			}
			retryAfter := int((h.backoff - elapsed + time.Second - 1) / time.Second)
			if retryAfter < 1 {
				retryAfter = 1
			}
			w.Header().Set("Retry-After", strconv.Itoa(retryAfter))
			h.writeJSON(w, r.Method, http.StatusServiceUnavailable, apiResponse{
				OK:    false,
				Error: "upstream temporarily unavailable; retry later",
			})
			return
		}
	}

	// 4. Cache miss or synchronous refresh via single-flight coalescing.
	h.misses.Add(1)
	payload, err := h.refreshSingleFlight(r.Context(), true)
	if err == nil {
		w.Header().Set("X-Cache", "MISS")
		w.Header().Set("Cache-Control", fmt.Sprintf("public, max-age=%d", int(h.ttl.Seconds())))
		h.writeJSON(w, r.Method, http.StatusOK, apiResponse{
			OK:       true,
			External: &payload,
		})
		return
	}

	// 5. Synchronous upstream failed: fall back to stale entry if within maxStale.
	h.mu.RLock()
	staleCandidate := h.entry
	h.mu.RUnlock()

	if staleCandidate != nil && h.now().Sub(staleCandidate.fetchedAt) <= h.ttl+h.maxStale {
		h.serveStale(w, r.Method, staleCandidate, h.now())
		return
	}

	status := http.StatusBadGateway
	if errors.Is(err, context.DeadlineExceeded) {
		status = http.StatusGatewayTimeout
	}
	h.writeJSON(w, r.Method, status, apiResponse{
		OK:    false,
		Error: err.Error(),
	})
}

func (h *Handler) serveStale(w http.ResponseWriter, method string, entry *cacheEntry, now time.Time) {
	h.staleHits.Add(1)
	age := int(now.Sub(entry.fetchedAt).Seconds())
	if age < 0 {
		age = 0
	}
	w.Header().Set("X-Cache", "STALE")
	w.Header().Set("Age", strconv.Itoa(age))
	w.Header().Set("Cache-Control", "public, max-age=0")
	h.writeJSON(w, method, http.StatusOK, apiResponse{
		OK:       true,
		External: &entry.payload,
	})
}

func (h *Handler) triggerBackgroundRefresh() {
	if h.closed.Load() {
		return
	}
	h.mu.Lock()
	if h.bgRunning || h.inFlight != nil {
		h.mu.Unlock()
		return
	}
	h.bgRunning = true
	h.bgWG.Add(1)
	h.mu.Unlock()

	go func() {
		defer h.bgWG.Done()
		defer func() {
			h.mu.Lock()
			h.bgRunning = false
			h.mu.Unlock()
		}()

		ctx, cancel := context.WithTimeout(context.Background(), h.fetchTimeout)
		defer cancel()

		if _, err := h.refreshSingleFlight(ctx, false); err != nil {
			h.logger.Warn("background refresh failed", slog.Any("err", err))
		}
	}()
}

func (h *Handler) refreshSingleFlight(ctx context.Context, recordFailure bool) (ExternalPayload, error) {
	h.mu.Lock()
	if call := h.inFlight; call != nil {
		h.mu.Unlock()
		call.wg.Wait()
		return call.payload, call.err
	}

	call := &flightCall{}
	call.wg.Add(1)
	h.inFlight = call
	h.mu.Unlock()

	payload, err := h.fetchWithRetries(ctx)

	h.mu.Lock()
	h.inFlight = nil
	if err == nil {
		now := h.now()
		payload.FetchedAt = now
		h.entry = &cacheEntry{
			payload:   payload,
			fetchedAt: now,
		}
		h.lastFailure = time.Time{}
		h.refreshes.Add(1)
	} else {
		h.errCount.Add(1)
		if recordFailure {
			h.lastFailure = h.now()
		}
	}
	h.mu.Unlock()

	call.payload = payload
	call.err = err
	call.wg.Done()

	return payload, err
}

func (h *Handler) fetchWithRetries(ctx context.Context) (ExternalPayload, error) {
	var lastErr error
	attempts := h.retries + 1

	for attempt := 0; attempt < attempts; attempt++ {
		if attempt > 0 && h.retryDelay > 0 {
			timer := time.NewTimer(h.retryDelay)
			select {
			case <-ctx.Done():
				timer.Stop()
				return ExternalPayload{}, ctx.Err()
			case <-timer.C:
			}
		}

		raw, err := h.fetcher(ctx)
		if err != nil {
			lastErr = err
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				return ExternalPayload{}, err
			}
			continue
		}
		if raw == nil {
			lastErr = errors.New("upstream returned nil response")
			continue
		}
		if raw.Fact == "" {
			lastErr = errors.New("upstream returned empty fact")
			continue
		}

		length := raw.Length
		if length == 0 {
			length = len(raw.Fact)
		}

		return ExternalPayload{
			Fact:   raw.