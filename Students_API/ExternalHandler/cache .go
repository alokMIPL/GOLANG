package response

import (
	"context"
	"sync/atomic"
	"time"
)

type cacheEntry struct {
	fact      CatFactResponse
	fetchedAt time.Time
}

type failure struct {
	until time.Time
	err   error
}

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

type Stats struct {
	Hits          uint64
	Misses        uint64
	StaleServed   uint64
	Refreshes     uint64
	RefreshErrors uint64
}

type counters struct {
	hits, misses, stale, refreshes, refreshErrors atomic.Uint64
}

func (h *Handler) get(ctx context.Context) (result, error) {
	now := h.now()
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
