package response

import (
	"context"
	"sync/atomic"
	"time"
)

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
