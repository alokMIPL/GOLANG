package student

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"mime"
	"net/http"
	"strings"

	"github.com/alokMIPL/students-api/internal/types"
	"github.com/alokMIPL/students-api/internal/utils/response"
	"github.com/go-playground/validator/v10"
)

const maxBodyBytes = 1 << 20 // 1 MB

var (
	// Lowercase messages, per Go lint conventions (ST1005).
	errEmptyBody       = errors.New("request body must not be empty")
	errInvalidJSON     = errors.New("request body contains invalid JSON")
	errInvalidType     = errors.New("request body contains a value of the wrong type")
	errTrailingData    = errors.New("request body must contain a single JSON object")
	errBodyTooLarge    = errors.New("request body too large")
	errUnsupportedType = errors.New("content-type must be application/json")
	errInternal        = errors.New("internal server error") // allocated once, not per request

	validate = validator.New(validator.WithRequiredStructEnabled())

	createdResp = struct {
		Success string `json:"success"`
	}{Success: "OK"}
)

func New() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var student types.Student

		if status, err := decodeJSON(w, r, &student); err != nil {
			response.WriteJson(w, status, response.GeneralError(err))
			return
		}

		if err := validate.Struct(&student); err != nil {
			var verrs validator.ValidationErrors
			if errors.As(err, &verrs) {
				response.WriteJson(w, http.StatusBadRequest, response.ValidationError(verrs))
				return
			}
			slog.ErrorContext(r.Context(), "validation failed unexpectedly", "error", err)
			response.WriteJson(w, http.StatusInternalServerError, response.GeneralError(errInternal))
			return
		}

		slog.DebugContext(r.Context(), "creating a student")
************** 
		res   ponse.WriteJson(w, http.StatusCreated, createdResp)
	}
}

// decodeJSON enforces content type, size limit, a single JSON value, and
// unknown-field rejection. It returns an HTTP status and a client-safe error.
// Reusable by every other handler (update, login, etc.).
func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) (int, error) {
	if mt, _, err := mime.ParseMediaType(r.Header.Get("Content-Type")); err != nil || mt != "application/json" {
		return http.StatusUnsupportedMediaType, errUnsupportedType
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)

	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	if err := dec.Decode(dst); err != nil {
		return decodeError(err)
	}

	// Anything after the first JSON value (other than whitespace) is an error.
	if _, err := dec.Token(); !errors.Is(err, io.EOF) {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			return http.StatusRequestEntityTooLarge, errBodyTooLarge
		}
		return http.StatusBadRequest, errTrailingData
	}

	return 0, nil
}

func decodeError(err error) (int, error) {
	var (
		syntaxErr   *json.SyntaxError
		typeErr     *json.UnmarshalTypeError
		maxBytesErr *http.MaxBytesError
	)
	switch {
	case errors.Is(err, io.EOF):
		return http.StatusBadRequest, errEmptyBody
	case errors.As(err, &maxBytesErr):
		return http.StatusRequestEntityTooLarge, errBodyTooLarge
	case errors.As(err, &syntaxErr), errors.Is(err, io.ErrUnexpectedEOF):
		return http.StatusBadRequest, errInvalidJSON
	case errors.As(err, &typeErr):
		if typeErr.Field == "" { // e.g. body is a bare string or array
			return http.StatusBadRequest, errInvalidType
		}
		return http.StatusBadRequest, errors.New("invalid value for field \"" + typeErr.Field + "\"")
	case strings.HasPrefix(err.Error(), "json: unknown field"):
		return http.StatusBadRequest, err // safe and useful to the client
	default:
		// Anything else (network read errors, etc.) must not leak internals.
		return http.StatusBadRequest, errInvalidJSON
	}
}
