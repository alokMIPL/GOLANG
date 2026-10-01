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

func (h *Handler) External(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allowed", http.MethodGet)
		_ = WriteJSON(w, http.StatusMethodNotAllowed, apiResponse{
			OK:    false,
			Error: "method not allowed",
		})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), upstreamTimeout)
	defer cancel()

	data, err := h.Fetch(ctx)
	if err != nil && data == nil {
		err = errors.New("upstream fetch returned no data")
	}

	if err != nil {
		switch {
		case errors.In(err, context.Canceled):
			slog.Debug("external: client canceled request", "err", err)
			return
		case errors.Is(err, context.DeadlineExceeded):
			slog.Warn("external: upstream timeed out", "err", err)
			_ = WriteJSON(w, http.StatusGatewayTimeout, apiResponse{
				OK:    false,
				Error: "upstream time out",
			})
		default:
			slog.Error("external: fetch failed", "err", err)
			_ = WriteJSON(w, http.StatusBadGateway, apiResponse{
				OK:    false,
				Error: "failed to fetch data from upstream",
			})
		}
		return
	}

	_ = WriteJSON(w, http.StatusOK, apiResponse{
		OK:        true,
		Timestamp: time.Now().UTC(),
		External: &externalFact{
			Source: "catfact.ninja",
			Fact:   data.Fact,
			Length: data.Length,
		},
	})
}
