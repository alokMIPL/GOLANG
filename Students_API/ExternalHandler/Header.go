package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"mime"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

const (
	maxBodyBytes  = 1 << 20 // 1 MB
	maxQueryRunes = 100
)

type api struct {
	store  TaskStore
	logger *slog.Logger
}

// newHandler wires routes and middleware. Order matters: the first entry is
// outermost, so request IDs exist for the logger, and the logger sees the 500
// written by the panic recoverer.
func newHandler(store TaskStore, logger *slog.Logger, cfg config) http.Handler {
	a := &api{store: store, logger: logger}
	return chain(a.routes(),
		requestID,
		logRequests(logger),
		recoverPanics(logger),
		cors(cfg.CORSOrigins),
		withTimeout(cfg.RequestTimeout),
		requireToken(cfg.APIToken),
	)
}

func (a *api) routes() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", a.health)
	mux.HandleFunc("GET /readyz", a.ready)

	mux.HandleFunc("GET /tasks", a.listTasks)
	mux.HandleFunc("POST /tasks", a.createTask)
	mux.HandleFunc("DELETE /tasks/completed", a.deleteCompleted)

	mux.HandleFunc("GET /tasks/{id}", a.getTask)
	mux.HandleFunc("PUT /tasks/{id}", a.replaceTask)
	mux.HandleFunc("PATCH /tasks/{id}", a.patchTask)
	mux.HandleFunc("DELETE /tasks/{id}", a.deleteTask)
	mux.HandleFunc("POST /tasks/{id}/toggle", a.toggleTask)
	return mux
}

// ---------------------------------------------------------------------
// Response helpers
// ---------------------------------------------------------------------

func writeJSON(w http.ResponseWriter, status int, data any) {
	// Marshal first so an encoding failure becomes a clean 500 rather than a
	// truncated 200.
	buf, err := json.Marshal(data)
	if err != nil {
		slog.Error("marshal response failed", "error", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error":"internal server error"}` + "\n"))
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write(append(buf, '\n'))
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

// fail maps store errors onto HTTP responses and logs unexpected ones.
func (a *api) fail(w http.ResponseWriter, r *http.Request, op string, err error) {
	switch {
	case errors.Is(err, errNotFound):
		writeError(w, http.StatusNotFound, "task not found")
	case errors.Is(err, errEmptyTitle), errors.Is(err, errTitleTooLong):
		writeError(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, context.DeadlineExceeded):
		a.logger.Warn(op+" timed out", "request_id", requestIDFrom(r.Context()))
		writeError(w, http.StatusServiceUnavailable, "request timed out")
	case errors.Is(err, context.Canceled):
		// The client went away; there is nobody to tell, but keep the log honest.
		a.logger.Debug(op+" canceled", "request_id", requestIDFrom(r.Context()))
		writeError(w, http.StatusServiceUnavailable, "request canceled")
	default:
		a.logger.Error(op+" failed", "request_id", requestIDFrom(r.Context()), "error", err)
		writeError(w, http.StatusInternalServerError, "internal server error")
	}
}

// ---------------------------------------------------------------------
// Request helpers
// ---------------------------------------------------------------------

type apiError struct {
	Status int
	Msg    string
}

func (e *apiError) Error() string { return e.Msg }

// respondDecodeError writes the status carried by a decodeJSON error.
func respondDecodeError(w http.ResponseWriter, err error) {
	var ae *apiError
	if errors.As(err, &ae) {
		writeError(w, ae.Status, ae.Msg)
		return
	}
	writeError(w, http.StatusBadRequest, "invalid JSON body")
}

// decodeJSON enforces a JSON content type and body size limit, rejects unknown
// fields and trailing data, and returns *apiError with a precise status.
func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	mt, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || mt != "application/json" {
		return &apiError{http.StatusUnsupportedMediaType, "Content-Type must be application/json"}
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	if err := dec.Decode(dst); err != nil {
		var (
			maxErr *http.MaxBytesError
			synErr *json.SyntaxError
			typErr *json.UnmarshalTypeError
		)
		switch {
		case errors.As(err, &maxErr):
			return &apiError{http.StatusRequestEntityTooLarge, "request body too large"}
		case errors.Is(err, io.EOF):
			return &apiError{http.StatusBadRequest, "request body must not be empty"}
		case errors.As(err, &synErr), errors.Is(err, io.ErrUnexpectedEOF):
			return &apiError{http.StatusBadRequest, "malformed JSON"}
		case errors.As(err, &typErr):
			return &apiError{http.StatusBadRequest, fmt.Sprintf("field %q has the wrong type", typErr.Field)}
		default:
			// e.g. `json: unknown field "x"`; safe and useful to echo.
			return &apiError{http.StatusBadRequest, err.Error()}
		}
	}
	if err := dec.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return &apiError{http.StatusBadRequest, "body must contain a single JSON value"}
	}
	return nil
}

func parseID(r *http.Request) (int64, error) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		return 0, errors.New("invalid task id")
	}
	return id, nil
}

func parseListParams(q url.Values) (ListParams, error) {
	p := ListParams{Limit: defaultLimit}

	if v := q.Get("done"); v != "" {
		b, err := strconv.ParseBool(v)
		if err != nil {
			return p, errors.New("done must be true or false")
		}
		p.Done = &b
	}
	if v := q.Get("limit"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 1 || n > maxLimit {
			return p, fmt.Errorf("limit must be between 1 and %d", maxLimit)
		}
		p.Limit = n
	}
	if v := q.Get("offset"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 0 {
			return p, errors.New("offset must be a non-negative integer")
		}
		p.Offset = n
	}
	switch strings.ToLower(q.Get("order")) {
	case "", "asc":
	case "desc":
		p.Desc = true
	default:
		return p, errors.New("order must be asc or desc")
	}
	p.Query = strings.TrimSpace(q.Get("q"))
	if utf8.RuneCountInString(p.Query) > maxQueryRunes {
		return p, fmt.Errorf("q must be at most %d characters", maxQueryRunes)
	}
	return p, nil
}

// ---------------------------------------------------------------------
// Handlers
// ---------------------------------------------------------------------

func (a *api) health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// ready reports whether dependencies (the database) are reachable.
func (a *api) ready(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()
	if err := a.store.Ping(ctx); err != nil {
		a.logger.Warn("readiness check failed", "error", err)
		writeError(w, http.StatusServiceUnavailable, "database unavailable")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ready"})
}

type listResponse struct {
	Data   []Task `json:"data"`
	Total  int    `json:"total"`
	Limit  int    `json:"limit"`
	Offset int    `json:"offset"`
}

// GET /tasks?done=&q=&limit=&offset=&order=
func (a *api) listTasks(w http.ResponseWriter, r *http.Request) {
	params, err := parseListParams(r.URL.Query())
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	tasks, total, err := a.store.List(r.Context(), params)
	if err != nil {
		a.fail(w, r, "list tasks", err)
		return
	}
	writeJSON(w, http.StatusOK, listResponse{
		Data: tasks, Total: total, Limit: params.Limit, Offset: params.Offset,
	})
}

// POST /tasks
func (a *api) createTask(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Title string `json:"title"`
	}
	if err := decodeJSON(w, r, &body); err != nil {
		respondDecodeError(w, err)
		return
	}
	task, err := a.store.Create(r.Context(), body.Title)
	if err != nil {
		a.fail(w, r, "create task", err)
		return
	}
	w.Header().Set("Location", "/tasks/"+strconv.FormatInt(task.ID, 10))
	writeJSON(w, http.StatusCreated, task)
}

// DELETE /tasks/completed
func (a *api) deleteCompleted(w http.ResponseWriter, r *http.Request) {
	n, err := a.store.DeleteCompleted(r.Context())
	if err != nil {
		a.fail(w, r, "delete completed tasks", err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]int64{"deleted": n})
}

// GET /tasks/{id}
func (a *api) getTask(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	task, err := a.store.Get(r.Context(), id)
	if err != nil {
		a.fail(w, r, "get task", err)
		return
	}
	writeJSON(w, http.StatusOK, task)
}

// PUT /tasks/{id}: full replacement . A missing "done" means false.
func (a *api) replaceTask(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	var body struct {
		Title string `json:"title"`
		Done  bool   `json:"done"`
	}
	if err := decodeJSON(w, r, &body); err != nil {
		respondDecodeError(w, err)
		return
	}
	task, err := a.store.Replace(r.Context(), id, body.Title, body.Done)
	if err != nil {
		a.fail(w, r, "replace task", err)
		return
	}
	writeJSON(w, http.StatusOK, task)
}

// PATCH /tasks/{id}: partial update; only supplied fields change.
func (a *api) patchTask(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	var body struct {
		Title *string `json:"title"`
		Done  *bool   `json:"done"`
	}
	if err := decodeJSON(w, r, &body); err != nil {
		respondDecodeError(w, err)
		return
	}
	patch := TaskPatch{Title: body.Title, Done: body.Done}
	if patch.isEmpty() {
		writeError(w, http.StatusBadRequest, "provide at least one of: title, done")
		return
	}
	task, err := a.store.Patch(r.Context(), id, patch)
	if err != nil {
		a.fail(w, r, "patch task", err)
		return
	}
	writeJSON(w, http.StatusOK, task)
}

// POST /tasks/{id}/toggle
func (a *api) toggleTask(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	task, err := a.store.Toggle(r.Context(), id)
	if err != nil {
		a.fail(w, r, "toggle task", err)
		return
	}
	writeJSON(w, http.StatusOK, task)
}

// DELETE /tasks/{id}
func (a *api) deleteTask(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := a.store.Delete(r.Context(), id); err != nil {
		a.fail(w, r, "delete task", err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
