package response

import (
	"context"
	"errors"
	"strings"
	"sync"
	"time"
)

// ErrFeedbackNotFound is returned when a feedback item does not exist.
var ErrFeedbackNotFound = errors.New("feedback not found")

// FactFeedbackRequest is the JSON body accepted by the feedback endpoint.
type FactFeedbackRequest struct {
	FactID  string `json:"fact_id"`
	Helpful *bool  `json:"helpful"`
	Comment string `json:"comment,omitempty"`
}

// Feedback is the internal representation of a stored feedback item.
type Feedback struct {
	ID        string
	FactID    string
	Helpful   bool
	Comment   string
	CreatedAt time.Time
}

// FeedbackDTO is the public JSON representation of a Feedback item.
type FeedbackDTO struct {
	ID        string    `json:"id"`
	FactID    string    `json:"fact_id"`
	Helpful   bool      `json:"helpful"`
	Comment   string    `json:"comment,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

// newFeedback builds a Feedback from a validated request.
func newFeedback(req FactFeedbackRequest, id string, now time.Time) Feedback {
	helpful := false
	if req.Helpful != nil {
		helpful = *req.Helpful
	}
	return Feedback{
		ID:        id,
		FactID:    strings.TrimSpace(req.FactID),
		Helpful:   helpful,
		Comment:   strings.TrimSpace(req.Comment),
		CreatedAt: now.UTC(),
	}
}

func (f Feedback) toDTO() FeedbackDTO {
	return FeedbackDTO{
		ID:        f.ID,
		FactID:    f.FactID,
		Helpful:   f.Helpful,
		Comment:   f.Comment,
		CreatedAt: f.CreatedAt,
	}
}

// FeedbackStore persists and retrieves feedback.
type FeedbackStore interface {
	Save(ctx context.Context, f Feedback) error
	Get(ctx context.Context, id string) (Feedback, error)
	ListByFact(ctx context.Context, factID string) ([]Feedback, error)
}

// MemoryFeedbackStore is a concurrency-safe store for tests and local runs.
// The zero value is ready to use.
type MemoryFeedbackStore struct {
	mu    sync.RWMutex
	items []Feedback
	byID  map[string]int // id -> index in items
}

func (s *MemoryFeedbackStore) Save(ctx context.Context, f Feedback) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.byID == nil {
		s.byID = make(map[string]int)
	}
	if idx, ok := s.byID[f.ID]; ok {
		// Same ID saved twice: overwrite rather than duplicate.
		s.items[idx] = f
		return nil
	}

	s.byID[f.ID] = len(s.items)
	s.items = append(s.items, f)
	return nil
}

func (s *MemoryFeedbackStore) Get(ctx context.Context, id string) (Feedback, error) {
	if err := ctx.Err(); err != nil {
		return Feedback{}, err
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	idx, ok := s.byID[id]
	if !ok {
		return Feedback{}, ErrFeedbackNotFound
	}
	return s.items[idx], nil
}

func (s *MemoryFeedbackStore) ListByFact(ctx context.Context, factID string) ([]Feedback, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	var out []Feedback
	for _, it := range s.items {
		if it.FactID == factID {
			out = append(out, it)
		}
	}
	return out, nil
}

// All returns a copy of every stored item, in insertion order.
func (s *MemoryFeedbackStore) All() []Feedback {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return append([]Feedback(nil), s.items...)
}

// Len returns the number of stored items.
func (s *MemoryFeedbackStore) Len() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.items)
}

// Reset removes all items. Useful between tests.
func (s *MemoryFeedbackStore) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.items = nil
	s.byID = nil
}
