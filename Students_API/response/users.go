package response
package response

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-playground/validator/v10"
)

var validate = validator.New()

type CreateUserRequest struct {
	Name  string `json:"name" validate:"required"`
	Email string `json:"email" validate:"required,email"`
	Age   int    `json:"age" validate:"gte=0,lte=130"`
}

type createUserResponse struct {
	Response
	User CreateUserRequest `json:"user"`
}

func (h *Handler) CreateUser(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		_ = WriteJSON(w, http.StatusMethodNotAllowed, GeneralError(errors.New("method not allowed")))
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	var req CreateUserRequest
	if err := dec.Decode(&req); err != nil {
		_ = WriteJSON(w, http.StatusBadRequest, GeneralError(errors.New("invalid JSON body: "+err.Error())))
		return
	}

	if err := validate.Struct(req); err != nil {
		var verrs validator.ValidationErrors
		if errors.As(err, &verrs) {
			_ = WriteJSON(w, http.StatusUnprocessableEntity, ValidationError(verrs))
			return
		}
		_ = WriteJSON(w, http.StatusInternalServerError, GeneralError(errors.New("validation failed")))
		return
	}

	_ = WriteJSON(w, http.StatusCreated, createUserResponse{
		Response: Response{Status: Status},
		User:     req,
	})
}