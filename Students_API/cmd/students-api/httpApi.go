// Package httpapi exposes the task service as a JSON REST API.
package httpapi

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"runtime/debug"
	"strconv"
	"time"

	"github.com/example/taskapi/internal/task"
)

// Options configures the router.
type Options struct {
	Service        *task.Service
	Logger         *slog.Logger
	MaxBodyBytes   int64
	RequestTimeout time.Duration
	Ready          func(context.Context) error // readiness probe, e.g. db.PingContext
}

type api struct {
	svc     *task.Service
	log     *slog.Logger
	maxBody int64
}

// NewRouter builds the full handler: routes wrapped in middleware.
func NewRouter(opts Options) http.Handler {
	if opts.Logger == nil {
		opts.Logger = slog.Default()
	}
	if opts.MaxBodyBytes <= 0 {
		opts.MaxBodyBytes = 1 << 20
	}
	a := &api{svc: opts.Service, log: opts.Logger, maxBody: opts.MaxBodyBytes}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", a.healthz)
	mux.HandleFunc("GET /readyz", a.readyz(opts.Ready))

	mux.HandleFunc("POST /v1/tasks", a.createTask)
	mux.HandleFunc("GET /v1/tasks", a.listTasks)
	mux.HandleFunc("GET /v1/tasks/{id}", a.getTask)
	mux.HandleFunc("PATCH /v1/tasks/{id}", a.updateTask)
	mux.HandleFunc("DELETE /v1/tasks/{id}", a.deleteTask)

	// Applied inside-out: the last wrapper is the outermost one, so a panic in
	// any layer below is caught and every response carries a request ID.
	var h http.Handler = mux
	h = withTimeout(opts.RequestTimeout)(h)
	h = withLogging(opts.Logger)(h)
	h = withRequestID(h)
	h = withRecover(opts.Logger)(h)
	return h
}

// ---------- handlers ----------

func (a *api) healthz(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (a *api) readyz(ready func(context.Context) error) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if ready != nil {
			ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
			defer cancel()
			if err := ready(ctx); err != nil {
				a.log.WarnContext(r.Context(), "readiness check failed", "err", err)
				writeError(w, http.StatusServiceUnavailable, "not_ready", "service is not ready", nil)
				return
			}
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "ready"})
	}
}

type createRequest struct {
	Title       string     `json:"title"`
	Description string     `json:"description"`
	DueAt       *time.Time `json:"due_at"`
}

func (a *api) createTask(w http.ResponseWriter, r *http.Request) {
	var req createRequest
	if err := a.decodeJSON(w, r, &req); err != nil {
		a.fail(w, r, err)
		return
	}
	t, err := a.svc.Create(r.Context(), task.CreateInput{
		Title:       req.Title,
		Description: req.Description,
		DueAt:       req.DueAt,
	})
	if err != nil {
		a.fail(w, r, err)
		return
	}
	w.Header().Set("Location", "/v1/tasks/"+strconv.FormatInt(t.ID, 10))
	writeJSON(w, http.StatusCreated, t)
}

func (a *api) getTask(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		a.fail(w, r, err)
		return
	}
	t, err := a.svc.Get(r.Context(), id)
	if err != nil {
		a.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, t)
}

func (a *api) listTasks(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	var f task.ListFilter

	if raw := q.Get("status"); raw != "" {
		s := task.Status(raw)
		f.Status = &s
	}
	var err error
	if f.Limit, err = intParam(q.Get("limit")); err != nil {
		a.fail(w, r, badParam("limit"))
		return
	}
	if f.Offset, err = intParam(q.Get("offset")); err != nil {
		a.fail(w, r, badParam("offset"))
		return
	}

	res, err := a.svc.List(r.Context(), f)
	if err != nil {
		a.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"items":  res.Items,
		"total":  res.Total,
		"limit":  res.Limit,
		"offset": res.Offset,
	})
}

type updateRequest struct {
	Title       *string      `json:"title"`
	Description *string      `json:"description"`
	Status      *task.Status `json:"status"`
	DueAt       *time.Time   `json:"due_at"`
	ClearDueAt  bool         `json:"clear_due_at"`
}

func (a *api) updateTask(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		a.fail(w, r, err)
		return
	}
	var req updateRequest
	if err := a.decodeJSON(w, r, &req); err != nil {
		a.fail(w, r, err)
		return
	}
	t, err := a.svc.Update(r.Context(), id, task.UpdateInput{
		Title:       req.Title,
		Description: req.Description,
		Status:      req.Status,
		DueAt:       req.DueAt,
		ClearDueAt:  req.ClearDueAt,
	})
	if err != nil {
		a.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, t)
}

func (a *api) deleteTask(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		a.fail(w, r, err)
		return
	}
	if err := a.svc.Delete(r.Context(), id); err != nil {
		a.fail(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ---------- request parsing ----------

// requestError is a client mistake with a ready-made HTTP mapping.
type requestError struct {
	status int
	code   string
	msg    string
	fields map[string]string
}

func (e *requestError) Error() string { return e.msg }

func badParam(name string) error {
	return &requestError{
		status: http.StatusUnprocessableEntity,
		code:   "validation_failed",
		msg:    "one or more fields are invalid",
		fields: map[string]string{name: "must be an integer"},
	}
}

func pathID(r *http.Request) (int64, error) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		return 0, &requestError{
			status: http.StatusBadRequest,
			code:   "bad_id",
			msg:    "task id must be a positive integer",
		}
	}
	return id, nil
}

func intParam(raw string) (int, error) {
	if raw == "" {
		return 0, nil
	}
	return strconv.Atoi(raw)
}

// decodeJSON reads exactly one JSON value, rejecting unknown fields, empty
// bodies, trailing garbage and bodies over the configured limit.
func (a *api) decodeJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	r.Body = http.MaxBytesReader(w, r.Body, a.maxBody)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	if err := dec.Decode(dst); err != nil {
		var (
			syntaxErr *json.SyntaxError
			typeErr   *json.UnmarshalTypeError
			tooBig    *http.MaxBytesError
		)
		switch {
		case errors.As(err, &tooBig):
			return &requestError{http.StatusRequestEntityTooLarge, "body_too_large",
				fmt.Sprintf("request body must not exceed %d bytes", tooBig.Limit), nil}
		case errors.Is(err, io.EOF):
			return &requestError{http.StatusBadRequest, "empty_body", "request body must not be empty", nil}
		case errors.As(err, &syntaxErr):
			return &requestError{http.StatusBadRequest, "malformed_json",
				fmt.Sprintf("malformed JSON at offset %d", syntaxErr.Offset), nil}
		case errors.As(err, &typeErr):
			return &requestError{http.StatusBadRequest, "wrong_type",
				fmt.Sprintf("field %q has the wrong type (expected %s)", typeErr.Field, typeErr.Type), nil}
		default:
			// Includes "unknown field" errors, which are safe and useful to echo.
			return &requestError{http.StatusBadRequest, "invalid_json", err.Error(), nil}
		}
	}

	if _, err := dec.Token(); !errors.Is(err, io.EOF) {
		return &requestError{http.StatusBadRequest, "trailing_data",
			"request body must contain a single JSON value", nil}
	}
	return nil
}

// ---------- responses ----------

type errorBody struct {
	Error errorDetail `json:"error"`
}

type errorDetail struct {
	Code    string            `json:"code"`
	Message string            `json:"message"`
	Fields  map[string]string `json:"fields,omitempty"`
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, code, msg string, fields map[string]string) {
	writeJSON(w, status, errorBody{Error: errorDetail{Code: code, Message: msg, Fields: fields}})
}

// fail maps any error from the service or request parsing to an HTTP response.
// Unexpected errors are logged in full but never leaked to the client.
func (a *api) fail(w http.ResponseWriter, r *http.Request, err error) {
	var (
		reqErr *requestError
		valErr *task.ValidationError
	)
	switch {
	case errors.As(err, &reqErr):
		writeError(w, reqErr.status, reqErr.code, reqErr.msg, reqErr.fields)
	case errors.As(err, &valErr):
		writeError(w, http.StatusUnprocessableEntity, "validation_failed",
			"one or more fields are invalid", valErr.Fields)
	case errors.Is(err, task.ErrNotFound):
		writeError(w, http.StatusNotFound, "not_found", "task not found", nil)
	case errors.Is(err, context.Canceled):
		// Client disconnected; there is nobody to respond to.
	case errors.Is(err, context.DeadlineExceeded):
		writeError(w, http.StatusServiceUnavailable, "timeout", "request timed out", nil)
	default:
		a.log.ErrorContext(r.Context(), "unhandled error", "err", err, "request_id", RequestID(r.Context()))
		writeError(w, http.StatusInternalServerError, "internal", "internal server error", nil)
	}
}

// ---------- middleware ----------

type middleware func(http.Handler) http.Handler

type ctxKey int

const requestIDKey ctxKey = iota

// RequestID returns the request ID stored in ctx, or "" if there is none.
func RequestID(ctx context.Context) string {
	id, _ := ctx.Value(requestIDKey).(string)
	return id
}

// withRequestID honours a sane incoming X-Request-ID or generates one, and
// echoes it on the response so clients can quote it in bug reports.
func withRequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get("X-Request-ID")
		if id == "" || len(id) > 64 {
			id = newRequestID()
		}
		w.Header().Set("X-Request-ID", id)
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), requestIDKey, id)))
	})
}

func newRequestID() string {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		return strconv.FormatInt(time.Now().UnixNano(), 36)
	}
	return hex.EncodeToString(b[:])
}

// statusRecorder captures what the handler wrote, for access logging.
type statusRecorder struct {
	http.ResponseWriter
	status int
	bytes  int
	wrote  bool
}

func (s *statusRecorder) WriteHeader(code int) {
	if !s.wrote {
		s.status, s.wrote = code, true
	}
	s.ResponseWriter.WriteHeader(code)
}

func (s *statusRecorder) Write(p []byte) (int, error) {
	if !s.wrote {
		s.status, s.wrote = http.StatusOK, true
	}
	n, err := s.ResponseWriter.Write(p)
	s.bytes += n
	return n, err
}

// Unwrap lets http.ResponseController reach the underlying writer.
func (s *statusRecorder) Unwrap() http.ResponseWriter { return s.ResponseWriter }

func withLogging(log *slog.Logger) middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
			next.ServeHTTP(rec, r)

			level := slog.LevelInfo
			switch {
			case rec.status >= 500:
				level = slog.LevelError
			case rec.status >= 400:
				level = slog.LevelWarn
			}
			log.Log(r.Context(), level, "http request",
				"method", r.Method,
				"path", r.URL.Path,
				"status", rec.status,
				"bytes", rec.bytes,
				"duration", time.Since(start).String(),
				"remote", r.RemoteAddr,
				"request_id", RequestID(r.Context()),
			)
		})
	}
}

// withTimeout bounds how long a handler's context lives. Handlers and the
// store honour it, so slow queries are abandoned instead of piling up.
func withTimeout(d time.Duration) middleware {
	return func(next http.Handler) http.Handler {
		if d <= 0 {
			return next
		}
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx, cancel := context.WithTimeout(r.Context(), d)
			defer cancel()
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// withRecover turns a panic into a logged 500 instead of a dropped connection.
func withRecover(log *slog.Logger) middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				rec := recover()
				if rec == nil {
					return
				}
				if rec == http.ErrAbortHandler {
					panic(rec) // deliberate abort; let net/http handle it
				}
				log.ErrorContext(r.Context(), "panic recovered",
					"panic", fmt.Sprint(rec),
					"stack", string(debug.Stack()),
					"request_id", RequestID(r.Context()),
				)
				writeError(w, http.StatusInternalServerError, "internal", "internal server error", nil)
			}()
			next.ServeHTTP(w, r)
		})
	}
}
