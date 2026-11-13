package response

import (
	"context"
	"errors"
	"log/slog"
	"math"
	"net/http"
	"strconv"
	"sync/atomic"
	"time"

	"golang.org/x/sync/singleflight"
)

const (
	defaultTTL            = 30 * time.Second
	defaultMaxStale       = 5 * time.Minute
	defaultFailureBackoff = 5 * time.Second
	defaultFetchTimeout   = 8 * time.Second

	flightKey = "cat-fact"
)

type Handler struct {
	fetch        Fetcher
	ttl          time.Duration // how long an entry is fresh
	maxStale     time.Duration // how long past ttl an entry may still be served
	failBackoff  time.Duration // how long to leave a failing upstream alone
	fetchTimeout time.Duration // per upstream call, independent of any caller
	bgRefresh    bool          // serve stale immediately and refresh in background
	log          *slog.Logger
	now          func() time.Time // injectable clock for tests

	entry   atomic.Pointer[cacheEntry]
	failure atomic.Pointer[failure]
	group   singleflight.Group
	stats   counters
}

type HandlerOption func(*Handler)

func WithTTL(d time.Duration) HandlerOption {
	return func(h *Handler) {
		if d > 0 {
			h.ttl = d
		}
	}
}

func WithMaxStale(d time.Duration) HandlerOption {
	return func(h *Handler) {
		if d >= 0 {
			h.maxStale = d
		}
	}
}

func WithBackgroundRefresh(on bool) HandlerOption {
	return func(h *Handler) { h.bgRefresh = on }
}

func WithFailureBackoff(d time.Duration) HandlerOption {
	return func(h *Handler) {
		if d >= 0 {
			h.failBackoff = d
		}
	}
}

func WithFetchTimeout(d time.Duration) HandlerOption {
	return func(h *Handler) {
		if d > 0 {
			h.fetchTimeout = d
		}
	}
}

func WithHandlerLogger(l *slog.Logger) HandlerOption {
	return func(h *Handler) {
		if l != nil {
			h.log = l
		}
	}
}

func withClock(now func() time.Time) HandlerOption {
	return func(h *Handler) { h.now = now }
}

func NewHandler(f Fetcher, opts ...HandlerOption) *Handler {
	h := &Handler{
		fetch:        f,
		ttl:          defaultTTL,
		maxStale:     defaultMaxStale,
		failBackoff:  defaultFailureBackoff,
		fetchTimeout: defaultFetchTimeout,
		bgRefresh:    true,
		log:          slog.Default(),
		now:          time.Now,
	}
	for _, o := range opts {
		o(h)
	}
	return h
}

func (h *Handler) flight(ctx context.Context) <-chan singleflight.Result {
	detached := context.WithoutCancel(ctx)
	return h.group.DoChan(flightKey, func() (any, error) {
		fctx, cancel := context.WithTimeout(detached, h.fetchTimeout)
		defer cancel()

		h.stats.refreshes.Add(1)
		fact, err := h.fetch(fctx)
		if err != nil {
			h.stats.refreshErrors.Add(1)
			h.failure.Store(&failure{until: h.now().Add(h.failBackoff), err: err})
			return nil, err
		}
		e := &cacheEntry{fact: *fact, fetchedAt: h.now()}
		h.entry.Store(e)
		h.failure.Store(nil)
		return e, nil
	})
}

func (h *Handler) refreshAsync(ctx context.Context) {
	if f := h.failure.Load(); f != nil && h.now().Before(f.until) {
		return
	}
	_ = h.flight(ctx)
}

func (h *Handler) Stats() Stats {
	return Stats{
		Hits:          h.stats.hits.Load(),
		Misses:        h.stats.misses.Load(),
		StaleServed:   h.stats.stale.Load(),
		Refreshes:     h.stats.refreshes.Load(),
		RefreshErrors: h.stats.refreshErrors.Load(),
	}
}

func (h *Handler) StatsHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
			return
		}
		writeJSON(w, http.StatusOK, h.Stats())
	})
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), h.fetchTimeout+2*time.Second)
	defer cancel()

	res, err := h.get(ctx)
	if err != nil {
		h.writeError(ctx, w, err)
		return
	}

	hdr := w.Header()
	hdr.Set("X-Cache", res.state.String())
	hdr.Set("Age", strconv.Itoa(int(res.age/time.Second)))
	if remaining := h.ttl - res.age; remaining > 0 {
		hdr.Set("Cache-Control", "public, max-age="+strconv.Itoa(int(remaining/time.Second)))
	} else {
		hdr.Set("Cache-Control", "no-cache")
	}
	writeJSON(w, http.StatusOK, res.fact)
}

func (h *Handler) writeError(ctx context.Context, w http.ResponseWriter, err error) {
	var be *backoffError
	switch {
	case errors.Is(err, context.Canceled):
		return // client went away; nobody to answer
	case errors.As(err, &be):
		secs := max(int(math.Ceil(be.until.Sub(h.now()).Seconds())), 1)
		w.Header().Set("Retry-After", strconv.Itoa(secs))
		h.log.WarnContext(ctx, "cat fact unavailable (backoff)", "retry_after", secs, "err", err)
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "upstream unavailable"})
	case errors.Is(err, context.DeadlineExceeded):
		h.log.ErrorContext(ctx, "cat fact timed out", "err", err)
		writeJSON(w, http.StatusGatewayTimeout, map[string]string{"error": "upstream timed out"})
	default:
		h.log.ErrorContext(ctx, "cat fact failed", "err", err)
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": "upstream unavailable"})
	}
}
