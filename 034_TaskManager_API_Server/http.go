package notes

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
)

// maxBodyBytes caps request bodies. Notes are at most ~10k characters, so 1 MiB
// leaves generous room for multibyte text and JSON overhead.
const maxBodyBytes = 1 << 20

type userCtxKey struct{}

// ContextWithUserID returns a context carrying the authenticated user's ID.
// Authentication middleware calls this; handlers read it back.
func ContextWithUserID(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, userCtxKey{}, userID)
}

// UserIDFromContext returns the authenticated user's ID, if any.
func UserIDFromContext(ctx context.Context) (string, bool) {
	id, ok := ctx.Value(userCtxKey{}).(string)
	return id, ok && id != ""
}

// errorBody is the JSON shape of every error response.
type errorBody struct {
	Error  string             `json:"error"`
	Fields []*ValidationError `json:"fields,omitempty"`
}

// Handler exposes a Service over HTTP.
//
// Routes (all require an authenticated user in the request context):
//
//	POST   /notes            create a note
//	GET    /notes            list notes (?limit=&offset=&tag=&q=)
//	GET    /notes/{id}       fetch one note
//	PATCH  /notes/{id}       partial update
//	DELETE /notes/{id}       delete
//	PUT    /notes/{id}/pin   pin
//	DELETE /notes/{id}/pin   unpin
type Handler struct {
	svc *Service
	log *slog.Logger
}

// NewHandler builds a Handler. A nil logger falls back to slog.Default().
func NewHandler(svc *Service, log *slog.Logger) *Handler {
	return &Handler{svc: svc, log: orDefault(log)}
}

// Routes returns the router for all note endpoints.
func (h *Handler) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /notes", h.create)
	mux.HandleFunc("GET /notes", h.list)
	mux.HandleFunc("GET /notes/{id}", h.get)
	mux.HandleFunc("PATCH /notes/{id}", h.update)
	mux.HandleFunc("DELETE /notes/{id}", h.delete)
	mux.HandleFunc("PUT /notes/{id}/pin", h.pin(true))
	mux.HandleFunc("DELETE /notes/{id}/pin", h.pin(false))
	return mux
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.user(w, r)
	if !ok {
		return
	}
	var in CreateInput
	if !h.decode(w, r, &in) {
		return
	}
	n, err := h.svc.Create(r.Context(), userID, in)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	w.Header().Set("Location", "/notes/"+n.ID.Hex())
	writeJSON(w, http.StatusCreated, n)
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.user(w, r)
	if !ok {
		return
	}

	q := r.URL.Query()
	p := ListParams{Tag: q.Get("tag"), Query: q.Get("q")}

	var errs ValidationErrors
	if v := q.Get("limit"); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err != nil {
			errs.add("limit", "must be an integer")
		} else {
			p.Limit = n
		}
	}
	if v := q.Get("offset"); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err != nil {
			errs.add("offset", "must be an integer")
		} else {
			p.Offset = n
		}
	}
	if err := errs.orNil(); err != nil {
		h.fail(w, r, err)
		return
	}

	res, err := h.svc.List(r.Context(), userID, p)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, res)
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.user(w, r)
	if !ok {
		return
	}
	n, err := h.svc.Get(r.Context(), userID, r.PathValue("id"))
	if err != nil {
		h.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, n)
}

func (h *Handler) update(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.user(w, r)
	if !ok {
		return
	}
	var in UpdateInput
	if !h.decode(w, r, &in) {
		return
	}
	n, err := h.svc.Update(r.Context(), userID, r.PathValue("id"), in)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, n)
}

func (h *Handler) delete(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.user(w, r)
	if !ok {
		return
	}
	if err := h.svc.Delete(r.Context(), userID, r.PathValue("id")); err != nil {
		h.fail(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// pin returns a handler that sets the pinned flag to the given value.
func (h *Handler) pin(pinned bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := h.user(w, r)
		if !ok {
			return
		}
		n, err := h.svc.SetPinned(r.Context(), userID, r.PathValue("id"), pinned)
		if err != nil {
			h.fail(w, r, err)
			return
		}
		writeJSON(w, http.StatusOK, n)
	}
}

// user extracts the authenticated user or responds 401.
func (h *Handler) user(w http.ResponseWriter, r *http.Request) (string, bool) {
	id, ok := UserIDFromContext(r.Context())
	if !ok {
		writeMessage(w, http.StatusUnauthorized, "authentication required")
	}
	return id, ok
}

// decode reads a single strict JSON object from the body into dst. On failure it
// writes the error response itself and returns false.
func (h *Handler) decode(w http.ResponseWriter, r *http.Request, dst any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)

	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	if err := dec.Decode(dst); err != nil {
		var (
			syntaxErr *json.SyntaxError
			typeErr   *json.UnmarshalTypeError
			tooBig    *http.MaxBytesError
		)
		switch {
		case errors.Is(err, io.EOF):
			writeMessage(w, http.StatusBadRequest, "request body must not be empty")
		case errors.As(err, &tooBig):
			writeMessage(w, http.StatusRequestEntityTooLarge,
				fmt.Sprintf("request body must not exceed %d bytes", tooBig.Limit))
		case errors.As(err, &syntaxErr):
			writeMessage(w, http.StatusBadRequest,
				fmt.Sprintf("malformed JSON at offset %d", syntaxErr.Offset))
		case errors.Is(err, io.ErrUnexpectedEOF):
			writeMessage(w, http.StatusBadRequest, "malformed JSON")
		case errors.As(err, &typeErr):
			writeMessage(w, http.StatusBadRequest,
				fmt.Sprintf("field %q has the wrong type (got JSON %s)", typeErr.Field, typeErr.Value))
		default: // includes "json: unknown field ..."
			writeMessage(w, http.StatusBadRequest, err.Error())
		}
		return false
	}

	// Reject trailing data such as `{"a":1}{"b":2}`.
	if _, err := dec.Token(); !errors.Is(err, io.EOF) {
		writeMessage(w, http.StatusBadRequest, "request body must contain a single JSON object")
		return false
	}
	return true
}

// fail maps a service error to an HTTP response. Unknown errors are logged and
// reported as an opaque 500 so internals never leak to clients.
func (h *Handler) fail(w http.ResponseWriter, r *http.Request, err error) {
	var verrs ValidationErrors
	switch {
	case errors.As(err, &verrs):
		writeJSON(w, http.StatusUnprocessableEntity, errorBody{
			Error:  "validation failed",
			Fields: verrs,
		})
	case errors.Is(err, ErrNotFound):
		writeMessage(w, http.StatusNotFound, "note not found")
	case errors.Is(err, ErrInvalidID):
		writeMessage(w, http.StatusBadRequest, "invalid note id")
	case errors.Is(err, ErrEmptyUpdate):
		writeMessage(w, http.StatusBadRequest, "no fields to update")
	default:
		h.log.ErrorContext(r.Context(), "request failed",
			"method", r.Method, "path", r.URL.Path, "err", err)
		writeMessage(w, http.StatusInternalServerError, "internal server error")
	}
}

func writeMessage(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, errorBody{Error: msg})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		// The status line is already sent; all we can do is record it.
		slog.Default().Error("notes: encode response", "err", err)
	}
}

func orDefault(log *slog.Logger) *slog.Logger {
	if log == nil {
		return slog.Default()
	}
	return log
}
