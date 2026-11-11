package response

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"mime"
	"net/http"
	"strings"

	"github.com/go-playground/validator/v10"
)

const maxBodyBytes = 1 << 20 // 1 MiB

var validate = validator.New()

// CreateUserRequest is the expected JSON payload for creating a user.
type CreateUserRequest struct {
	Name  string `json:"name" validate:"required"`
	Email string `json:"email" validate:"required,email"`
	Age   int    `json:"age" validate:"gte=0,lte=130"`
}

// normalize trims whitespace and lowercases the email.
func (r *CreateUserRequest) normalize() {
	r.Name = strings.TrimSpace(r.Name)
	r.Email = strings.ToLower(strings.TrimSpace(r.Email))
}

type createUserResponse struct {
	Response
	User CreateUserRequest `json:"user"`
}

// hasJSONContentType reports whether the request declares a JSON body.
func hasJSONContentType(r *http.Request) bool {
	ct := r.Header.Get("Content-Type")
	if ct == "" {
		return false
	}
	mt, _, err := mime.ParseMediaType(ct)
	if err != nil {
		return false
	}
	return mt == "application/json"
}

// decodeJSONBody decodes the request body into dst and returns a
// descriptive error plus the HTTP status that should be sent.
func decodeJSONBody(w http.ResponseWriter, r *http.Request, dst any) (int, error) {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)

	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	if err := dec.Decode(dst); err != nil {
		var (
			syntaxErr    *json.SyntaxError
			unmarshalErr *json.UnmarshalTypeError
			maxBytesErr  *http.MaxBytesError
		)

		switch {
		case errors.As(err, &syntaxErr):
			return http.StatusBadRequest,
				fmt.Errorf("malformed JSON at position %d", syntaxErr.Offset)

		case errors.Is(err, io.ErrUnexpectedEOF):
			return http.StatusBadRequest, errors.New("malformed JSON: unexpected end of body")

		case errors.As(err, &unmarshalErr):
			return http.StatusBadRequest,
				fmt.Errorf("field %q must be of type %s (position %d)",
					unmarshalErr.Field, unmarshalErr.Type, unmarshalErr.Offset)

		case strings.HasPrefix(err.Error(), "json: unknown field "):
			field := strings.TrimPrefix(err.Error(), "json: unknown field ")
			return http.StatusBadRequest, fmt.Errorf("unknown field %s", field)

		case errors.Is(err, io.EOF):
			return http.StatusBadRequest, errors.New("request body must not be empty")

		case errors.As(err, &maxBytesErr):
			return http.StatusRequestEntityTooLarge,
				fmt.Errorf("request body must not exceed %d bytes", maxBodyBytes)

		default:
			return http.StatusBadRequest, errors.New("invalid JSON body: " + err.Error())
		}
	}

	// Make sure the body contains only a single JSON value.
	if err := dec.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return http.StatusBadRequest, errors.New("request body must contain a single JSON object")
	}

	return http.StatusOK, nil
}

// CreateUser handles POST requests that create a new user.
func (h *Handler) CreateUser(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		_ = WriteJSON(w, http.StatusMethodNotAllowed, GeneralError(errors.New("method not allowed")))
		return
	}

	if !hasJSONContentType(r) {
		_ = WriteJSON(w, http.StatusUnsupportedMediaType,
			GeneralError(errors.New("Content-Type must be application/json")))
		return
	}

	var req CreateUserRequest
	if status, err := decodeJSONBody(w, r, &req); err != nil {
		log.Printf("create user: decode failed: %v", err)
		_ = WriteJSON(w, status, GeneralError(err))
		return
	}

	req.normalize()

	if err := validate.Struct(req); err != nil {
		var verrs validator.ValidationErrors
		if errors.As(err, &verrs) {
			log.Printf("create user: validation failed: %v", verrs)
			_ = WriteJSON(w, http.StatusUnprocessableEntity, ValidationError(verrs))
			return
		}

		log.Printf("create user: unexpected validation error: %v", err)
		_ = WriteJSON(w, http.StatusInternalServerError, GeneralError(errors.New("validation failed")))
		return
	}

	log.Printf("create user: created user with email %s", req.Email)

	_ = WriteJSON(w, http.StatusCreated, createUserResponse{
		Response: Response{Status: Status},
		User:     req,
	})
}
