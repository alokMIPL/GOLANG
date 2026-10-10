package response

import (
	"strings"
	"time"
)

// ---- Request DTO (wire format in) ----

// FactFeedbackRequest is the body of POST /api/feedback.
type FactFeedbackRequest struct {
	Fact    string `json:"fact"              validate:"required,max=500"`
	Rating  int    `json:"rating"            validate:"required,min=1,max=5"`
	Comment string `json:"comment,omitempty" validate:"omitempty,max=280"`
	Email   string `json:"email,omitempty"   validate:"omitempty,email"`
}

// Normalize runs after decoding and before validation, so "  a@B.com " passes
// as "a@b.com" and a whitespace-only fact fails "required".
func (r *FactFeedbackRequest) Normalize() {
	r.Fact = strings.TrimSpace(r.Fact)
	r.Comment = strings.TrimSpace(r.Comment)
	r.Email = strings.ToLower(strings.TrimSpace(r.Email))
}

// ---- Response DTO (wire format out) ----

// FactFeedbackResponse deliberately omits Email: it is personal data the
// client already has and has no reason to receive back.
type FactFeedbackResponse struct {
	ID        string    `json:"id"`
	Fact      string    `json:"fact"`
	Rating    int       `json:"rating"`
	Comment   string    `json:"comment,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

// ---- Domain model (what the store keeps) ----

type Feedback struct {
	ID        string
	Fact      string
	Rating    int
	Comment   string
	Email     string
	CreatedAt time.Time
}

// ---- Mappers: the only place DTOs and the domain touch ----

func newFeedback(req FactFeedbackRequest, id string, now time.Time) Feedback {
	return Feedback{
		ID:        id,
		Fact:      req.Fact,
		Rating:    req.Rating,
		Comment:   req.Comment,
		Email:     req.Email,
		CreatedAt: now.UTC(),
	}
}

func (f Feedback) toDTO() FactFeedbackResponse {
	return FactFeedbackResponse{
		ID:        f.ID,
		Fact:      f.Fact,
		Rating:    f.Rating,
		Comment:   f.Comment,
		CreatedAt: f.CreatedAt,
	}
}

// newExternalFact maps the upstream DTO to the public one, so the handler
// no longer builds this struct inline.
func newExternalFact(f CatFactResponse) *externalFact {
	return &externalFact{
		Source: factSource,
		Fact:   f.Fact,
		Length: f.Length,
	}
}
