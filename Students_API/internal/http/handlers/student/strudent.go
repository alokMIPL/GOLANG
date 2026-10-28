package student

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"

	"github.com/alokMIPL/students-api/internal/types"
	"github.com/alokMIPL/students-api/internal/utils/response"
	"github.com/go-playground/validator/v10"
)

// maxBodyBytes caps the request body to protect against oversized payloads.
const maxBodyBytes = 1 << 20 // 1 MB

// errEmptyBody is a sentinel error, allocated once instead of on every request.
var errEmptyBody = errors.New("empty body")

// validate is safe for concurrent use and caches struct metadata,
// so it should be created once, not per request.
var validate = validator.New(validator.WithRequiredStructEnabled())

func New() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
		defer r.Body.Close()

		var student types.Student

		dec := json.NewDecoder(r.Body)
		dec.DisallowUnknownFields() // optional: reject unexpected fields

		if err := dec.Decode(&student); err != nil {
			if errors.Is(err, io.EOF) {
				err = errEmptyBody
			}
			response.WriteJson(w, http.StatusBadRequest, response.GeneralError(err))
			return
		}

		if err := validate.Struct(student); err != nil {
			var validationErrs validator.ValidationErrors
			if errors.As(err, &validationErrs) {
				response.WriteJson(w, http.StatusBadRequest, response.ValidationError(validationErrs))
				return
			}
			response.WriteJson(w, http.StatusInternalServerError, response.GeneralError(err))
			return
		}

		slog.Info("creating a student")

		response.WriteJson(w, http.StatusCreated, map[string]string{"success": "OK"})
	}
}
