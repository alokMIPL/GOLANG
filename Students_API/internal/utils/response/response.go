package response

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
)

// apiResponse is the single envelope for every JSON response.
type apiResponse struct {
	OK        bool          `json:"ok"`
	Timestamp time.Time     `json:"timestamp,omitzero"` // Go 1.24+
	Error     string        `json:"error,omitempty"`
	External  *externalFact `json:"external,omitempty"`
}

type externalFact struct {
	Source string `json:"source"`
	Fact   string `json:"fact"`
	Length int    `json:"length"`
}

// WriteJSON marshals first so an encoding failure can still produce a 500.
func WriteJSON(w http.ResponseWriter, status int, data any) error {
	body, err := json.Marshal(data)
	if err != nil {
		http.Error(w, `{"ok":false,"error":"internal error"}`, http.StatusInternalServerError)
		return fmt.Errorf("marshal response: %w", err)
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_, err = w.Write(append(body, '\n'))
	return err
}

// GeneralError wraps err for the client. Log the real error and pass a
// generic one here if its text should not be exposed.
func GeneralError(err error) apiResponse {
	return apiResponse{OK: false, Error: err.Error()}
}

func ValidationError(errs validator.ValidationErrors) apiResponse {
	msgs := make([]string, 0, len(errs))
	for _, e := range errs {
		switch e.ActualTag() {
		case "required":
			msgs = append(msgs, fmt.Sprintf("Field %s is required", e.Field()))
		case "min", "max", "len":
			msgs = append(msgs, fmt.Sprintf("Field %s must satisfy %s=%s", e.Field(), e.ActualTag(), e.Param()))
		case "email":
			msgs = append(msgs, fmt.Sprintf("Field %s must be a valid email", e.Field()))
		default:
			msgs = append(msgs, fmt.Sprintf("Field %s is invalid", e.Field()))
		}
	}
	return apiResponse{OK: false, Error: strings.Join(msgs, ", ")}
}
