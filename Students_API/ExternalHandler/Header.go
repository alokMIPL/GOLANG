package response

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/sync/singleflight"
)

var _ http.Handler = (*Handler)(nil)

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
