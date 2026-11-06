package response

import (
	"context"
	"errors"
	"fmt"
	"math/rand/v2"
	"runtime/debug"
	"time"

	"golang.org/x/sync/singleflight"
)

// maxRetryDelay caps exponential backoff so a large retry count can never
// produce an absurdly long sleep (or overflow the shift).
const maxRetryDelay = 30 * time.Second

// Sentinel errors returned (wrapped) by refresh. Callers can use errors.Is to
// distinguish "upstream said nothing useful" from transport-level failures.
var (
	ErrEmptyResponse = errors.New("upstream fetch: empty response")
	ErrNoFact        = errors.New("upstream fetch: response has no fact")
	ErrPanicked      = errors.New("upstream fetch panicked")
)

// permanentError marks an error that retrying cannot fix (for example a 4xx
// response or a malformed body). fetchWithRetry stops immediately on one.
type permanentError struct{ err error }

func (p *permanentError) Error() string { return p.err.Error() }
func (p *permanentError) Unwrap() error { return p.err }

// Permanent wraps err so that fetchWithRetry will not retry it. A nil error
// stays nil.
func Permanent(err error) error {
	if err == nil {
		return nil
	}
	return &permanentError{err: err}
}

// isPermanent reports whether err (or anything it wraps) was marked permanent.
func isPermanent(err error) bool {
	var p *permanentError
	return errors.As(err, &p)
}

// isFresh reports whether e is non-nil and still within the TTL.
func (h *Handler) isFresh(e *cacheEntry) bool {
	return e != nil && h.now().Sub(e.fetchedAt) < h.ttl
}

// inBackoff reports whether a recent failure is still suppressing refreshes.
func (h *Handler) inBackoff() bool {
	f := h.failure.Load()
	return f != nil && h.now().Before(f.until)
}

// flight runs refresh through singleflight so concurrent callers share one
// upstream call.
func (h *Handler) flight(ctx context.Context) <-chan singleflight.Result {
	return h.group.DoChan(flightKey, func() (any, error) {
		// A flight may have finished between the caller's cache check and now.
		if e := h.entry.Load(); h.isFresh(e) {
			return e, nil
		}
		return h.refresh(ctx)
	})
}

// refreshAsync starts at most one background refresh at a time.
//
// It is a no-op when a recent failure is still in its backoff window, when
// another background refresh is already running, or when the handler has been
// closed.
func (h *Handler) refreshAsync(ctx context.Context) {
	if h.inBackoff() {
		return
	}
	if !h.bgActive.CompareAndSwap(false, true) {
		return
	}

	// Registering with the WaitGroup under the mutex guarantees Close cannot
	// start waiting between our closed-check and wg.Add.
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

		start := h.now()
		res := <-h.flight(ctx)
		if res.Err != nil {
			h.log.Warn("external: background refresh failed",
				"err", res.Err,
				"elapsed", h.now().Sub(start),
				"shared", res.Shared)
			return
		}
		h.log.Debug("external: background refresh succeeded",
			"elapsed", h.now().Sub(start),
			"shared", res.Shared)
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
			e, err = nil, fmt.Errorf("%w: %v", ErrPanicked, r)
		}
		if err != nil {
			h.recordFailure(err)
		} else {
			h.recordSuccess()
		}
	}()

	fctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), h.timeout)
	defer cancel()

	got, err := h.fetchWithRetry(fctx)
	if err != nil {
		return nil, fmt.Errorf("upstream fetch: %w", err)
	}
	if err := validate(got); err != nil {
		return nil, err
	}

	e = &cacheEntry{fact: *got, fetchedAt: h.now()}
	h.entry.Store(e)
	return e, nil
}

// validate checks that an upstream response is usable before it is cached.
func validate(got *CatFactResponse) error {
	switch {
	case got == nil:
		return ErrEmptyResponse
	case got.Fact == "":
		return ErrNoFact
	}
	return nil
}

// fetchWithRetry calls fetch up to h.retries+1 times, sleeping with
// exponential backoff and jitter between attempts. It stops early when the
// context ends or the error is marked permanent.
func (h *Handler) fetchWithRetry(ctx context.Context) (*CatFactResponse, error) {
	var lastErr error
	for attempt := 0; attempt <= h.retries; attempt++ {
		if attempt > 0 {
			delay := h.retryBackoff(attempt)
			h.log.Debug("external: retrying upstream fetch",
				"attempt", attempt, "delay", delay, "last_err", lastErr)
			if err := sleep(ctx, delay); err != nil {
				return nil, errors.Join(lastErr, err)
			}
			h.stats.retries.Add(1)
		}

		got, err := h.fetch(ctx)
		if err == nil {
			return got, nil
		}
		lastErr = err

		if isPermanent(err) {
			h.log.Debug("external: permanent upstream error, not retrying", "err", err)
			break
		}
		if ctx.Err() != nil {
			break
		}
	}
	return nil, lastErr
}

// retryBackoff returns the delay before the given retry attempt (1-based):
// retryDelay * 2^(attempt-1), capped at maxRetryDelay, with up to 20% random
// jitter subtracted so that many instances don't retry in lockstep.
func (h *Handler) retryBackoff(attempt int) time.Duration {
	if attempt < 1 || h.retryDelay <= 0 {
		return 0
	}

	shift := attempt - 1
	// Guard against overflow: once the shift is large enough the cap applies.
	if shift > 20 {
		return maxRetryDelay
	}

	d := h.retryDelay << shift
	if d <= 0 || d > maxRetryDelay {
		d = maxRetryDelay
	}

	jitter := time.Duration(rand.Int64N(int64(d)/5 + 1))
	return d - jitter
}

// recordSuccess clears any failure state left over from earlier refreshes.
func (h *Handler) recordSuccess() {
	h.failure.Store(nil)
}

// recordFailure counts the error and, if backoff is enabled, suppresses
// further background refreshes until the backoff window passes.
func (h *Handler) recordFailure(err error) {
	h.stats.refreshErrors.Add(1)
	if h.backoff > 0 {
		h.failure.Store(&failure{until: h.now().Add(h.backoff), err: err})
	}
}

// sleep waits for d or until ctx ends, whichever comes first. A non-positive
// d returns immediately with ctx's current error (nil if still live).
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
