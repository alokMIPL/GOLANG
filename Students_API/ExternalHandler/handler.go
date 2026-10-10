package response

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"golang.org/x/sync/singleflight"
)

const (
	defaultTTL     = 30 * time.Second
	defaultTimeout = 5 * time.Second
	defaultURL     = "https://catfact.ninja/fact"
	factSource     = "catfact.ninja"
	flightKey      = "fact"
)

// FactFetcher lets tests swap in a fake upstream.
type FactFetcher func(ctx context.Context) (*CatFactResponse, error)

type cacheEntry struct {
	fact      CatFactResponse
	fetchedAt time.Time
}

// Handler serves a cat fact, cached for ttl. Concurrent misses share a single
// upstream call, and a stale value is served if a refresh fails.
type Handler struct {
	fetch   FactFetcher
	ttl     time.Duration
	timeout time.Duration
	now     func() time.Time
	log     *slog.Logger

	group singleflight.Group

	mu    sync.RWMutex
	entry *cacheEntry
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
		fetch:   NewCatFactFetcher(nil, defaultURL),
		ttl:     defaultTTL,
		timeout: defaultTimeout,
		now:     time.Now,
		log:     slog.Default(),
	}
	for _, opt := range opts {
		opt(h)
	}
	return h
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
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
		// Client went away; nobody is left to answer.
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
		// A flight may have finished between our cache check and now.
		if fact, ok := h.fresh(); ok {
			return fact, nil
		}

		// Detach from the first caller's cancellation so one impatient client
		// can't fail the fetch for everyone waiting on it; the timeout still bounds it.
		fctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), h.timeout)
		defer cancel()

		fact, err := h.fetch(fctx)
		if err != nil {
			return nil, fmt.Errorf("upstream fetch: %w", err)
		}
		if fact == nil {
			return nil, errors.New("upstream fetch: empty response")
		}

		h.store(*fact)
		return *fact, nil
	})

	select {
	case <-ctx.Done():
		return CatFactResponse{}, ctx.Err()
	case res := <-ch:
		if res.Err != nil {
			if stale, ok := h.stale(); ok {
				h.log.Warn("external: serving stale fact", "err", res.Err)
				return stale, nil
			}
			return CatFactResponse{}, res.Err
		}
		return res.Val.(CatFactResponse), nil
	}
}

func (h *Handler) store(fact CatFactResponse) {
	h.mu.Lock()
	h.entry = &cacheEntry{fact: fact, fetchedAt: h.now()}
	h.mu.Unlock()
}

func (h *Handler) fresh() (CatFactResponse, bool) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	if h.entry != nil && h.now().Sub(h.entry.fetchedAt) < h.ttl {
		return h.entry.fact, true
	}
	return CatFactResponse{}, false
}

func (h *Handler) stale() (CatFactResponse, bool) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	if h.entry == nil {
		return CatFactResponse{}, false
	}
	return h.entry.fact, true
}
