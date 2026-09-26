package response

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
)

type Response struct {
	Status string
	Error  string
}

const (
	catFaceURL      = "https://catfact.ninja/fact"
	upstreamTimeout = 5 * http.DefaultClient.Timeout
	maxUpstreamBody = 1 << 20
)

const (
	Status      = "OK"
	StatusError = "Error"
)

type CatFactResponse struct {
	Fact string `json:"fact"`
}

type apiResponse struct {
	OK        bool          `json:"OK"`
	Timestamp time.Time     `json:"timestamp,omitempty"`
	Error     string        `json:"error, omitempty"`
	External  *externalFact `json:"external, omniempty"`
}

type externalFact struct {
	Source string `json:"source"`
	Fact   string `json:"fact"`
	Length string `json:"length"`
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
}

func WriteJson(w http.ResponseWriter, status int, data interface{}) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	return json.NewEncoder(w).Encode(data)

}

func GeneralError(err error) Response {
	return Response{
		Status: StatusError,
		Error:  err.Error(),
	}
}

func ValidationError(errs validator.ValidationErrors) Response {
	var errMsgs []string

	for _, err := range errs {
		switch err.ActualTag() {
		case "required":
			errMsgs = append(errMsgs, fmt.Sprintf("Field %s is required field", err.Field()))
		default:
			errMsgs = append(errMsgs, fmt.Sprintf("Field %s is invalid", err.Field()))
		}
	}
	return Response{
		Status: StatusError,
		Error:  strings.Join(errMsgs, ", "),
	}
}
