package response

import (
	"context"
	"errors"
	"fmt"
	"runtime/debug"
	"time"

	"golang.org/x/sync/singleflight"
)

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
