package response

import (
	"context"
	"log"
	"net/http"
	"time"
)

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
		_ = WriteJSON(w, http.StatusMethodNotAllowed, apiResponse{
			OK:    false,
			Error: "method not allowed",
		})
		return
	}

	data, err := h.Fetch(r.Context())
	if err != nil {
		log.Printf("External: fetch failed: %v", err)
		_ = WriteJSON(w, http.StatusBadGateway, apiResponse{
			OK:    false,
			Error: "failed to fetch data from upstream",
		})
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
