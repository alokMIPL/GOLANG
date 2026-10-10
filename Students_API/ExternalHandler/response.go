package response

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"sync/atomic"
	"time"

	"golang.org/x/sync/singleflight"
)

const (
	defaultTTL      = 30 * time.Second
	defaultMaxStale = 5 * time.Minute
	defaultTimeout  = 5 * time.Second
	defaultURL      = "https://catfact.ninja/fact"
	factSource      = "catfact.ninja"
	flightKey       = "fact"
)

// FactFetcher lets tests swap in a fake upstream.
type FactFetcher func(ctx context.Context) (*CatFactResponse, error)

// cacheEntry is immutable once published, so readers never need a lock.
type cacheEntry struct {
	fact      CatFactResponse
	fetchedAt time.Time
}

// Handler serves a cat fact.
//
//   - A fact stays fresh for ttl.
//   - Concurrent cache misses share one upstream call.
//   - If a refresh fails, a fact up to ttl+maxStale old is served instead.
type Handler struct {
	fetch    FactFetcher
	ttl      time.Duration
	maxStale time.Duration
	timeout  time.Duration
	now      func() time.Time
	log      *slog.Logger

	group singleflight.Group
	entry atomic.Pointer[cacheEntry]
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

// WithTimeout bounds each upstream call.
func WithTimeout(d time.Duration) Option {
	return func(h *Handler) {
		if d > 0 {
			h.timeout = d
		}
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
		now:      time.Now,
		log:      slog.Default(),
	}
	for _, opt := range opts {
		opt(h)
	}
	return h
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

	fact, err := h.get(r.Context())
	if err != nil {
		h.writeError(w, err)
		return
	}

	_ = WriteJSON(w, http.StatusOK, apiResponse{
		OK:        true,
		Timestamp: h.now().UTC(),
		External: &externalFact{
			Source: factSource,
			Fact:   fact.Fact,
			Length: fact.Length,
		},
	})
}

func (h *Handler) writeError(w http.ResponseWriter, err error) {
	switch {
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

func (h *Handler) get(ctx context.Context) (CatFactResponse, error) {
	if fact, ok := h.fresh(); ok {
		return fact, nil
	}

	ch := h.group.DoChan(flightKey, func() (any, error) {
		// A flight may have finished between our cache check and this call.
		if fact, ok := h.fresh(); ok {
			return fact, nil
		}
		return h.refresh(ctx)

	})

	select {
	case <-ctx.Done():
		return CatFactResponse{}, ctx.Err()
	case res := <-ch:
		if res.Err != nil {
			if fact, age, ok := h.stale(); ok {
				h.log.Warn("external: serving stale fact",
					"err", res.Err, "age", age)
				return fact, nil
			}
			return CatFactResponse{}, res.Err
		}
		return res.Val.(CatFactResponse), nil
	}
}

// refresh calls upstream and publishes the result. It runs inside a
// singleflight, so it is detached from the triggering request's cancellation:
// one impatient client must not fail the fetch for everyone waiting on it.
// The timeout still bounds it, and context values (trace IDs) are kept.
func (h *Handler) refresh(ctx context.Context) (fact CatFactResponse, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("upstream fetch panicked: %v", r)
		}
	}()

	fctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), h.timeout)
	defer cancel()

	got, err := h.fetch(fctx)
	switch {
	case err != nil:
		return CatFactResponse{}, fmt.Errorf("upstream fetch: %w", err)
	case got == nil:
		return CatFactResponse{}, errors.New("upstream fetch: empty response")
	case got.Fact == "":
		return CatFactResponse{}, errors.New("upstream fetch: response has no fact")
	}

	h.entry.Store(&cacheEntry{fact: *got, fetchedAt: h.now()})
	return *got, nil
}

func (h *Handler) fresh() (CatFactResponse, bool) {
	e := h.entry.Load()
	if e == nil || h.now().Sub(e.fetchedAt) >= h.ttl {
		return CatFactResponse{}, false
	}
	return e.fact, true
}

// stale returns the cached fact if it is still within the stale window.
func (h *Handler) stale() (CatFactResponse, time.Duration, bool) {
	e := h.entry.Load()
	if e == nil {
		return CatFactResponse{}, 0, false
	}
	age := h.now().Sub(e.fetchedAt)
	if age >= h.ttl+h.maxStale {
		return CatFactResponse{}, age, false
	}
	return e.fact, age, true
}
