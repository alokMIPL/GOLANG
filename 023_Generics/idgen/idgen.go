package idgen

import "github.com/google/uuid"

// New returns a new random UUID (v4) as a string.
func New() string {
	return uuid.New().String()
}

// IsValid reports whether s is a syntactically valid UUID.
func IsValid(s string) bool {
	_, err := uuid.Parse(s)
	return err == nil
}
