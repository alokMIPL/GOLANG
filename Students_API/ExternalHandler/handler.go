package response

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"
)

const upstreamTimeout = 5 * time.Second

// FactFetcher lets tests swap in a fake upstream.
type FactFetcher func(ctx context.Context) (*CatFactResponse, error)

type Handler struct {
	Fetch FactFetcher
}

func NewHandler() *Handler {
	return &Handler{Fetch: fetchCatFact}
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
