package response

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"
)

const (
	maxFactIDLen   = 128
	maxCommentLen  = 2000
	defaultTimeout = 5 * time.Second
)

// ValidationError describes why a feedback request was rejected.
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("invalid %s: %s", e.Field, e.Message)
}

// validateFeedbackRequest checks a decoded request and returns a
// *ValidationError describing the first problem found.
func validateFeedbackRequest(req FactFeedbackRequest) error {
	factID := strings.TrimSpace(req.FactID)
	switch {
	case factID == "":
		return &ValidationError{Field: "fact_id", Message: "is required"}
	case len(factID) > maxFactIDLen:
		return &ValidationError{Field: "fact_id", Message: "is too long"}
	}

	if req.Helpful == nil {
		return &ValidationError{Field: "helpful", Message: "is required"}
	}

	if utf8.RuneCountInString(req.Comment) > maxCommentLen {
		return &ValidationError{
			Field:   "comment",
			Message: fmt.Sprintf("must be at most %d characters", maxCommentLen),
		}
	}
	return nil
}

// Option customizes a FeedbackHandler.
type Option func(*FeedbackHandler)

// WithClock overrides the time source (handy in tests).
func WithClock(now func() time.Time) Option {
	return func(h *FeedbackHandler) {
		if now != nil {
			h.now = now
		}
	}
}

// WithIDGenerator overrides how feedback IDs are created.
func WithIDGenerator(fn func() string) Option {
	return func(h *FeedbackHandler) {
		if fn != nil {
			h.newID = fn
		}
	}
}

// WithSaveTimeout sets the maximum time allowed for a store write.
func WithSaveTimeout(d time.Duration) Option {
	return func(h *FeedbackHandler) {
		if d > 0 {
			h.saveTimeout = d
		}
	}
}

type FeedbackHandler struct {
	store       FeedbackStore
	now         func() time.Time
	newID       func() string
	log         *slog.Logger
	saveTimeout time.Duration
}

func NewFeedbackHandler(store FeedbackStore, log *slog.Logger, opts ...Option) *FeedbackHandler {
	if log == nil {
		log = slog.Default()
	}
	h := &FeedbackHandler{
		store:       store,
		now:         time.Now,
		newID:       randomID,
		log:         log,
		saveTimeout: defaultTimeout,
	}
	for _, opt := range opts {
		opt(h)
	}
	return h
}

func (h *FeedbackHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		h.writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	req, err := DecodeJSON[FactFeedbackRequest](w, r)
	if err != nil {
		writeDecodeError(w, err)
		return
	}

	if err := validateFeedbackRequest(req); err != nil {
		var verr *ValidationError
		if errors.As(err, &verr) {
			h.log.Warn("feedback: validation failed", "field", verr.Field, "reason", verr.Message)
			h.writeError(w, http.StatusUnprocessableEntity, verr.Error())
			return
		}
		h.writeError(w, http.StatusBadRequest, "invalid request")
		return
	}

	fb := newFeedback(req, h.newID(), h.now())

	ctx, cancel := context.WithTimeout(r.Context(), h.saveTimeout)
	defer cancel()

	if err := h.store.Save(ctx, fb); err != nil {
		h.handleSaveError(w, fb, err)
		return
	}

	h.log.Info("feedback: saved", "id", fb.ID, "fact_id", fb.FactID, "helpful", fb.Helpful)

	dto := fb.toDTO()
	_ = WriteJSON(w, http.StatusCreated, apiResponse{
		OK:        true,
		Timestamp: h.now().UTC(),
		Feedback:  &dto,
	})
}

func (h *FeedbackHandler) handleSaveError(w http.ResponseWriter, fb Feedback, err error) {
	h.log.Error("feedback: save failed", "id", fb.ID, "fact_id", fb.FactID, "err", err)

	switch {
	case errors.Is(err, context.DeadlineExceeded):
		h.writeError(w, http.StatusGatewayTimeout, "saving feedback timed out")
	case errors.Is(err, context.Canceled):
		// Client went away; nothing useful to send, but reply anyway.
		h.writeError(w, http.StatusRequestTimeout, "request canceled")
	default:
		h.writeError(w, http.StatusInternalServerError, "could not save feedback")
	}
}

func (h *FeedbackHandler) writeError(w http.ResponseWriter, status int, msg string) {
	_ = WriteJSON(w, status, apiResponse{OK: false, Error: msg})
}

func randomID() string {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		// Extremely unlikely; fall back to a time-based ID so we never return "".
		return fmt.Sprintf("fb-%x", time.Now().UnixNano())
	}
	return hex.EncodeToString(b)
}
