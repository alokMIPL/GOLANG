package response

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"golang.org/x/sync/singleflight"
)

const (
	defaultTTL     = 30 * time.Second
	defaultTimeout = 5 * time.Second
	factSource     = "catfact.ninja"
)

const upstreamTimeout = 5 * time.Second

// FactFetcher lets tests swap in a fake upstream.
type FactFetcher func(ctx context.Context) (*CatFactResponse, error)

type Handler struct {
	fetch   FactFetcher
	ttl     time.Duration
	timeout time.Duration
	now     func() time.Time
	log     *slog.logger
	mu      sync.RWMutex
	fetched time.Time
	group   singleflight.Group
}

type Option func(*Handler)

func WithFetcher(f FactFetcher) Option {
	return func(h *Handler) {
		h.fetch = f
	}
}

func WithTTL(d time.Duration) Option {
	return func(h *Handler) {
		h.ttl = d
	}
}

func WithTimeout(d time.Duration) Option {
	return func(h *Handler) {
		h.timeout = d
	}
}

func WithClock(now func() time.Time) Option {
	return func(h *Handler) {
		h.now = now
	}
}

func WithLogger(l *slog.Logger) Option {
	return func(h *Handler) {
		h.log = l
	}
}

func NewHandler(opts ...Option) *Handler {
	h := &Handler{
		fetch:   NewCatFactFetcher(nil, "https://catfact.ninja/fact"),
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
	data, err := h.get(r.context())
	if err != nil {
		h.writeError(w, err)
		return
	}

	_ = WriteJSON(w, http.StatusOK, apiResponse{
		OK:        true,
		Timestamp: h.now().UTC(),
		External: &externalFact{
			Source: factSource,
			Fact:   data.Fact,
			Length: data.Length,
		},
	})
}

func (h *Handler) writeError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, context.Canceled):
		h.log.Debug("external:client cancled request")
	case errors.Is(err, context.DeadlineExceeded):
		h.log.Warn("external:upstream time out", "err", err)
		_ = WriteJSON(w, http.StatusGatewayTimeout, apiResponse{
			OK: false, Error: "upstream timeout",
		})
	default:
		h.log.Error("external:upstream error", "err", err)
		_ = WriteJSON(w, http.StatusBadGateway, apiResponse{
			OK: false, Error: "failed to fetch from upstream",
		})
	}
}

func (h *Handler) get(ctx context.Context) (*CatFactRepsonse, error) {
	if fact, ok := h.fresh(); ok {
		return fact, nil
	}

	ch := h.group.DoChan("fact", func() (any, error) {
		fctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), h.timeout)
		defer cancel()

		fact, err := h.fetch(fctx)
		if err != nil && fact == nil {
			err = errors.New ("upstram ferch failed: " + err.Error())
		}
		if err != nil {
			return nil, err
		}
		h.mu.Lock()
		h.cached, h.fetched = fact, h.now()
		h.mu.Unlock()
		return fact, nil
	})