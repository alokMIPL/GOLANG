package student

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"mime"
	"net/http"

	"github.com/alokMIPL/students-api/internal/types"
	"github.com/alokMIPL/students-api/internal/utils/response"
	"github.com/go-playground/validator/v10"
)

const maxBodyBytes = 1 << 20 // 1 MB

var (
	errEmptyBody       = errors.New("request body must not be empty")
	errInvalidJSON     = errors.New("request body contains invalid JSON")
	errTrailingData    = errors.New("request body must contain a single JSON object")
	errBodyTooLarge    = errors.New("request body too large")
	errUnsupportedType = errors.New("Content-Type must be application/json")

	// Safe for concurrent use; caches struct metadata, so create once.
	validate = validator.New(validator.WithRequiredStructEnabled())

	// Static response, no per-request map allocation.
	createdResp = struct {
		Success string `json:"success"`
	}{Success: "OK"}
)

func New() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 1. Reject wrong content type early, before reading the body.
		if mt, _, err := mime.ParseMediaType(r.Header.Get("Content-Type")); err != nil || mt != "application/json" {
			response.WriteJson(w, http.StatusUnsupportedMediaType, response.GeneralError(errUnsupportedType))
			return
		}

		// 2. Cap body size. No need for `defer r.Body.Close()`;
		//    the net/http server closes the request body itself.
		r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)

		var student types.Student

		dec := json.NewDecoder(r.Body)
		dec.DisallowUnknownFields()

		if err := dec.Decode(&student); err != nil {
			status, clientErr := decodeError(err)
			response.WriteJson(w, status, response.GeneralError(clientErr))
			return
		}

		// 3. Reject payloads like `{...}{...}` or `{...} garbage`.
		if _, err := dec.Token(); !errors.Is(err, io.EOF) {
			response.WriteJson(w, http.StatusBadRequest, response.GeneralError(errTrailingData))
			return
		}

		// 4. Pass a pointer to avoid copying the struct.
		if err := validate.Struct(&student); err != nil {
			var verrs validator.ValidationErrors
			if errors.As(err, &verrs) {
				response.WriteJson(w, http.StatusBadRequest, response.ValidationError(verrs))
				return
			}
			// Programmer error (e.g. invalid validation target): log it, don't leak it.
			slog.ErrorContext(r.Context(), "validation failed unexpectedly", "error", err)
			response.WriteJson(w, http.StatusInternalServerError, response.GeneralError(errors.New("internal server error")))
			return
		}

		slog.InfoContext(r.Context(), "creating a student")

		response.WriteJson(w, http.StatusCreated, createdResp)
	}
}

// decodeError maps decoder errors to an HTTP status and a client-safe error,
// so raw internal messages are never exposed.
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
		return http.StatusBadRequest, errors.New("invalid value for field \"" + typeErr.Field + "\"")
	default:
		// Includes unknown-field errors, which are safe and useful to return.
		return http.StatusBadRequest, err
	}
}
