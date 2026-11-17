package notes

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strconv"

	"notes_api/internal/db"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

const maxBodyBytes = 1 << 20 // 1 MiB

// Repo is what the handler needs from storage. *Repository satisfies it, and
// tests can pass a fake.
type Repo interface {
	Create(ctx context.Context, n *Note) error
	Get(ctx context.Context, userID string, id primitive.ObjectID) (*Note, error)
	List(ctx context.Context, userID string, p ListParams) ([]Note, int64, error)
	Update(ctx context.Context, userID string, id primitive.ObjectID, in UpdateInput) (*Note, error)
	Delete(ctx context.Context, userID string, id primitive.ObjectID) error
	Search(ctx context.Context, userID, query string, limit int64) ([]Note, error)
}

// Handler serves the notes endpoints.
type Handler struct {
	repo Repo
	log  *slog.Logger
}

// NewHandler builds a Handler. A nil logger falls back to slog.Default.
func NewHandler(repo Repo, log *slog.Logger) *Handler {
	if log == nil {
		log = slog.Default()
	}
	return &Handler{repo: repo, log: log}
}

// Register adds the routes to mux (Go 1.22+ method/wildcard patterns).
func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("POST /notes", h.create)
	mux.HandleFunc("GET /notes", h.list)
	mux.HandleFunc("GET /notes/search", h.search) // more specific than {id}
	mux.HandleFunc("GET /notes/{id}", h.get)
	mux.HandleFunc("PATCH /notes/{id}", h.update)
	mux.HandleFunc("DELETE /notes/{id}", h.delete)
}

// ---------------------------------------------------------------------------
// Handlers
// ---------------------------------------------------------------------------

type createRequest struct {
	Title   string   `json:"title"`
	Content string   `json:"content"`
	Tags    []string `json:"tags"`
	Pinned  bool     `json:"pinned"`
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.userID(w, r)
	if !ok {
		return
	}

	var req createRequest
	if !h.decode(w, r, &req) {
		return
	}

	n := &Note{
		UserID:  userID,
		Title:   req.Title,
		Content: req.Content,
		Tags:    req.Tags,
		Pinned:  req.Pinned,
	}
	if err := h.repo.Create(r.Context(), n); err != nil {
		h.fail(w, r, err)
		return
	}

	w.Header().Set("Location", "/notes/"+n.ID.Hex())
	h.respond(w, http.StatusCreated, n)
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.userID(w, r)
	if !ok {
		return
	}
	id, err := ParseID(r.PathValue("id"))
	if err != nil {
		h.fail(w, r, err)
		return
	}

	n, err := h.repo.Get(r.Context(), userID, id)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	h.respond(w, http.StatusOK, n)
}

type listResponse struct {
	Items  []Note `json:"items"`
	Total  int64  `json:"total"`
	Limit  int64  `json:"limit"`
	Offset int64  `json:"offset"`
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.userID(w, r)
	if !ok {
		return
	}

	q := r.URL.Query()
	p := ListParams{
		Limit:  parseInt(q.Get("limit")),
		Offset: parseInt(q.Get("offset")),
		Tag:    q.Get("tag"),
	}
	p.Normalize()

	items, total, err := h.repo.List(r.Context(), userID, p)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	h.respond(w, http.StatusOK, listResponse{Items: items, Total: total, Limit: p.Limit, Offset: p.Offset})
}

func (h *Handler) search(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.userID(w, r)
	if !ok {
		return
	}

	q := r.URL.Query()
	items, err := h.repo.Search(r.Context(), userID, q.Get("q"), parseInt(q.Get("limit")))
	if err != nil {
		h.fail(w, r, err)
		return
	}
	h.respond(w, http.StatusOK, map[string]any{"items": items})
}

func (h *Handler) update(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.userID(w, r)
	if !ok {
		return
	}
	id, err := ParseID(r.PathValue("id"))
	if err != nil {
		h.fail(w, r, err)
		return
	}

	var in UpdateInput
	if !h.decode(w, r, &in) {
		return
	}

	n, err := h.repo.Update(r.Context(), userID, id, in)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	h.respond(w, http.StatusOK, n)
}

func (h *Handler) delete(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.userID(w, r)
	if !ok {
		return
	}
	id, err := ParseID(r.PathValue("id"))
	if err != nil {
		h.fail(w, r, err)
		return
	}

	if err := h.repo.Delete(r.Context(), userID, id); err != nil {
		h.fail(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// userID reads the caller's identity.
//
// PLACEHOLDER: this trusts an X-User-ID header so the example runs without an
// auth system. Replace it with the user ID from your verified session or JWT.
func (h *Handler) userID(w http.ResponseWriter, r *http.Request) (string, bool) {
	id := r.Header.Get("X-User-ID")
	if id == "" {
		h.writeError(w, http.StatusUnauthorized, "missing user identity")
		return "", false
	}
	return id, true
}

// decode reads a JSON body, rejecting unknown fields, oversized bodies and
// trailing garbage.
func (h *Handler) decode(w http.ResponseWriter, r *http.Request, dst any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)

	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	if err := dec.Decode(dst); err != nil {
		var tooBig *http.MaxBytesError
		switch {
		case errors.As(err, &tooBig):
			h.writeError(w, http.StatusRequestEntityTooLarge, "request body too large")
		case errors.Is(err, io.EOF):
			h.writeError(w, http.StatusBadRequest, "request body is empty")
		default:
			h.writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		}
		return false
	}
	if dec.More() {
		h.writeError(w, http.StatusBadRequest, "request body must contain a single JSON object")
		return false
	}
	return true
}

// fail maps domain and database errors onto HTTP status codes.
func (h *Handler) fail(w http.ResponseWriter, r *http.Request, err error) {
	var verr *ValidationError

	switch {
	case errors.As(err, &verr):
		h.respond(w, http.StatusUnprocessableEntity, map[string]any{"error": verr.Error(), "field": verr.Field})
	case errors.Is(err, ErrInvalidID):
		h.writeError(w, http.StatusBadRequest, "invalid note id")
	case errors.Is(err, ErrNotFound):
		h.writeError(w, http.StatusNotFound, "note not found")
	case db.IsTimeout(err), errors.Is(err, context.DeadlineExceeded):
		h.log.Warn("request timed out", "path", r.URL.Path, "error", err)
		h.writeError(w, http.StatusGatewayTimeout, "database timed out")
	case errors.Is(err, context.Canceled):
		// Client went away; nothing useful to send.
		h.writeError(w, 499, "client closed request")
	default:
		h.log.Error("request failed", "method", r.Method, "path", r.URL.Path, "error", err)
		h.writeError(w, http.StatusInternalServerError, "internal server error")
	}
}

func (h *Handler) writeError(w http.ResponseWriter, status int, msg string) {
	h.respond(w, status, map[string]string{"error": msg})
}

func (h *Handler) respond(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(body); err != nil {
		h.log.Error("encode response", "error", err)
	}
}

func parseInt(s string) int64 {
	n, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0
	}
	return n
}
