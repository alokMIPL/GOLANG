package notes

import (
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

const (
	maxTitleLen   = 200
	maxContentLen = 10_000
	maxTags       = 20
	maxTagLen     = 40
)

// Errors returned by this package. Use errors.Is to check for them.
var (
	ErrNotFound  = errors.New("notes: not found")
	ErrInvalidID = errors.New("notes: invalid id")
)

// ValidationError describes a single invalid field.
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("notes: invalid %s: %s", e.Field, e.Message)
}

// Note is one stored note.
type Note struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	UserID    string             `bson:"user_id"       json:"user_id"`
	Title     string             `bson:"title"         json:"title"`
	Content   string             `bson:"content"       json:"content"`
	Tags      []string           `bson:"tags"          json:"tags"`
	Pinned    bool               `bson:"pinned"        json:"pinned"`
	CreatedAt time.Time          `bson:"created_at"    json:"created_at"`
	UpdatedAt time.Time          `bson:"updated_at"    json:"updated_at"`
}

// UpdateInput holds optional changes for a partial update. A nil field means
// "leave unchanged", which lets callers clear a value (e.g. empty Content)
// without ambiguity.
type UpdateInput struct {
	Title   *string   `json:"title"`
	Content *string   `json:"content"`
	Tags    *[]string `json:"tags"`
	Pinned  *bool     `json:"pinned"`
}

// IsEmpty reports whether the input changes nothing.
func (u UpdateInput) IsEmpty() bool {
	return u.Title == nil && u.Content == nil && u.Tags == nil && u.Pinned == nil
}

// ListParams controls pagination and filtering for List.
type ListParams struct {
	Limit  int64
	Offset int64
	Tag    string
}

// Normalize applies defaults and caps.
func (p *ListParams) Normalize() {
	if p.Limit <= 0 {
		p.Limit = 20
	}
	if p.Limit > 100 {
		p.Limit = 100
	}
	if p.Offset < 0 {
		p.Offset = 0
	}
	p.Tag = strings.TrimSpace(strings.ToLower(p.Tag))
}

// ParseID converts a hex string into an ObjectID.
func ParseID(s string) (primitive.ObjectID, error) {
	id, err := primitive.ObjectIDFromHex(s)
	if err != nil {
		return primitive.NilObjectID, ErrInvalidID
	}
	return id, nil
}

// Validate checks a note before it is stored.
func (n *Note) Validate() error {
	if strings.TrimSpace(n.UserID) == "" {
		return &ValidationError{"user_id", "is required"}
	}
	if err := validateTitle(n.Title); err != nil {
		return err
	}
	if err := validateContent(n.Content); err != nil {
		return err
	}
	if _, err := normalizeTags(n.Tags); err != nil {
		return err
	}
	return nil
}

func validateTitle(title string) error {
	title = strings.TrimSpace(title)
	if title == "" {
		return &ValidationError{"title", "is required"}
	}
	if utf8.RuneCountInString(title) > maxTitleLen {
		return &ValidationError{"title", fmt.Sprintf("must be at most %d characters", maxTitleLen)}
	}
	return nil
}

func validateContent(content string) error {
	if utf8.RuneCountInString(content) > maxContentLen {
		return &ValidationError{"content", fmt.Sprintf("must be at most %d characters", maxContentLen)}
	}
	return nil
}

// normalizeTags lower-cases, trims and de-duplicates tags, and enforces limits.
func normalizeTags(tags []string) ([]string, error) {
	if len(tags) > maxTags {
		return nil, &ValidationError{"tags", fmt.Sprintf("at most %d tags allowed", maxTags)}
	}

	seen := make(map[string]struct{}, len(tags))
	out := make([]string, 0, len(tags))

	for _, t := range tags {
		t = strings.ToLower(strings.TrimSpace(t))
		if t == "" {
			continue
		}
		if utf8.RuneCountInString(t) > maxTagLen {
			return nil, &ValidationError{"tags", fmt.Sprintf("each tag must be at most %d characters", maxTagLen)}
		}
		if _, dup := seen[t]; dup {
			continue
		}
		seen[t] = struct{}{}
		out = append(out, t)
	}
	return out, nil
}
