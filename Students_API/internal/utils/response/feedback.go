package response

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"net/http"
	"sync"
	"time"
)

type FeedbackStore interface {
	Save(ctx context.Context, f Feedback) error
}

// MemoryFeedbackStore is a concurrency-safe store for tests and local runs.
type MemoryFeedbackStore struct {
	mu    sync.Mutex
	items []Feedback
}

func (s *MemoryFeedbackStore) Save(_ context.Context, f Feedback) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.items = append(s.items, f)
	return nil
}

func (s *MemoryFeedbackStore) All() []Feedback {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]Feedback(nil), s.items...)
}

type FeedbackHandler struct {
	store FeedbackStore
	now   func() time.Time
	newID func() string
	log   *slog.Logger
}

func NewFeedbackHandler(store FeedbackStore, log *slog.Logger) *FeedbackHandler {
	if log == nil {
		log = slog.Default()
	}
	return &FeedbackHandler{store: store, now: time.Now, newID: randomID, log: log}
}

func (h *FeedbackHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		_ = WriteJSON(w, http.StatusMethodNotAllowed, apiResponse{OK: false, Error: "method not allowed"})
		return
	}

	req, err := DecodeJSON[FactFeedbackRequest](w, r)
	if err != nil {
		writeDecodeError(w, err)
		return
	}

	fb := newFeedback(req, h.newID(), h.now())
	if err := h.store.Save(r.Context(), fb); err != nil {
		h.log.Error("feedback: save failed", "err", err)
		_ = WriteJSON(w, http.StatusInternalServerError, apiResponse{
			OK:    false,
			Error: "could not save feedback",
		})
		return
	}

	dto := fb.toDTO()
	_ = WriteJSON(w, http.StatusCreated, apiResponse{
		OK:        true,
		Timestamp: h.now().UTC(),
		Feedback:  &dto,
	})
}

func randomID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b) // never fails on supported platforms (Go 1.24+)
	return hex.EncodeToString(b)
}
