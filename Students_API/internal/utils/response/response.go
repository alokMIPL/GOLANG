// Package response proxies a "cat fact" from an upstream API and wraps it in a
// small, production-minded toolkit:
//
//   - NewHandler: an http.Handler with an injectable Fetcher, a per-request
//     timeout, upstream payload validation and safe error mapping.
//   - NewHTTPFetcher: a Fetcher for the real upstream, with bounded reads.
//   - NewRateLimiter: per-client token-bucket middleware that returns 429 and
//     a Retry-After header, and cleans up idle clients automatically.
//   - ValidationError / WriteValidationError: turn validator errors into a
//     consistent JSON envelope.
//
// Every response, success or failure, uses the same JSON envelope:
//
//	{"ok": true,  "external": {"fact": "...", "length": 42}}
//	{"ok": false, "error": "human readable message"}
package response

import (
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"strings"

	"github.com/go-playground/validator/v10"
)

// CatFactResponse is the payload returned by the upstream API. The validate
// tags are enforced by the handler before the data is passed on to clients, so
// a misbehaving upstream can never leak garbage through us.
type CatFactResponse struct {
	Fact   string `json:"fact" validate:"required,max=2000"`
	Length int    `json:"length" validate:"gte=0"`
}

// apiResponse is the single JSON envelope used by every endpoint.
type apiResponse struct {
	OK       bool              `json:"ok"`
	Error    string            `json:"error,omitempty"`
	Fields   map[string]string `json:"fields,omitempty"` // per-field validation messages
	External *CatFactResponse  `json:"external,omitempty"`
}

func errorResponse(msg string) apiResponse { return apiResponse{Error: msg} }

// writeJSON marshals first so an encoding failure becomes a clean 500 instead
// of a half-written response. Responses are never cacheable: they are either
// per-request errors or a random fact.
func writeJSON(w http.ResponseWriter, status int, resp apiResponse) {
	body, err := json.Marshal(resp)
	if err != nil { // not reachable for these types, but never write garbage
		status = http.StatusInternalServerError
		body = []byte(`{"ok":false,"error":"internal server error"}`)
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_, _ = w.Write(append(body, '\n'))
}

// WriteError writes the standard error envelope with the given status.
func WriteError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, errorResponse(msg))
}

// ValidationError converts validator errors into the standard envelope.
// Error holds all messages joined with "; " and Fields maps each field name to
// its own message, so clients can show errors next to the right input.
func ValidationError(errs validator.ValidationErrors) apiResponse {
	if len(errs) == 0 {
		return errorResponse("validation failed")
	}
	msgs := make([]string, 0, len(errs))
	fields := make(map[string]string, len(errs))
	for _, fe := range errs {
		msg := fieldMessage(fe)
		msgs = append(msgs, msg)
		fields[fe.Field()] = msg
	}
	return apiResponse{Error: strings.Join(msgs, "; "), Fields: fields}
}

// WriteValidationError writes ValidationError as a 422 Unprocessable Entity.
func WriteValidationError(w http.ResponseWriter, errs validator.ValidationErrors) {
	writeJSON(w, http.StatusUnprocessableEntity, ValidationError(errs))
}

// fieldMessage renders one human-readable message per failed rule.
func fieldMessage(fe validator.FieldError) string {
	name := fe.Field()
	switch fe.Tag() {
	case "required":
		return fmt.Sprintf("Field %s is required", name)
	case "min":
		return fmt.Sprintf("Field %s must be at least %s%s", name, fe.Param(), unit(fe))
	case "max":
		return fmt.Sprintf("Field %s must be at most %s%s", name, fe.Param(), unit(fe))
	case "gte":
		return fmt.Sprintf("Field %s must be greater than or equal to %s%s", name, fe.Param(), unit(fe))
	case "lte":
		return fmt.Sprintf("Field %s must be less than or equal to %s%s", name, fe.Param(), unit(fe))
	case "email":
		return fmt.Sprintf("Field %s must be a valid email address", name)
	default:
		return fmt.Sprintf("Field %s failed validation (%s)", name, fe.Tag())
	}
}

// unit makes length rules on strings read naturally ("at most 5 characters").
func unit(fe validator.FieldError) string {
	if fe.Kind() == reflect.String {
		return " characters"
	}
	return ""
}
