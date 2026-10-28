package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

const (
	taskColumns  = "id, title, done, created_at"
	maxBodyBytes = 1 << 20 // 1 MB
)

var errNotFound = errors.New("task not found")

// ---------------------------------------------------------------------
// Model
// ---------------------------------------------------------------------

type Task struct {
	ID        int64     `json:"id"`
	Title     string    `json:"title"`
	Done      bool      `json:"done"`
	CreatedAt time.Time `json:"created_at"`
}

// ---------------------------------------------------------------------
// Store
// ---------------------------------------------------------------------

type taskStore struct {
	db *sql.DB
}

func newTaskStore(db *sql.DB) *taskStore {
	return &taskStore{db: db}
}

func (s *taskStore) migrate(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS tasks (
			id         INTEGER PRIMARY KEY AUTOINCREMENT,
			title      TEXT    NOT NULL,
			done       INTEGER NOT NULL DEFAULT 0,
			created_at INTEGER NOT NULL
		)`)
	return err
}

// scanner is satisfied by both *sql.Row and *sql.Rows.
type scanner interface {
	Scan(dest ...any) error
}

func scanTask(sc scanner) (Task, error) {
	var (
		t       Task
		done    int
		created int64
	)
	if err := sc.Scan(&t.ID, &t.Title, &done, &created); err != nil {
		return Task{}, err
	}
	t.Done = done != 0
	t.CreatedAt = time.Unix(created, 0).UTC()
	return t, nil
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func (s *taskStore) list(ctx context.Context) ([]Task, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT `+taskColumns+` FROM tasks ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	tasks := []Task{} // non-nil so JSON encodes as [] instead of null
	for rows.Next() {
		t, err := scanTask(rows)
		if err != nil {
			return nil, err
		}
		tasks = append(tasks, t)
	}
	return tasks, rows.Err()
}

func (s *taskStore) get(ctx context.Context, id int64) (Task, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT `+taskColumns+` FROM tasks WHERE id = ?`, id)

	t, err := scanTask(row)
	if errors.Is(err, sql.ErrNoRows) {
		return Task{}, errNotFound
	}
	return t, err
}

func (s *taskStore) create(ctx context.Context, title string) (Task, error) {
	now := time.Now().Truncate(time.Second).UTC()

	res, err := s.db.ExecContext(ctx,
		`INSERT INTO tasks (title, done, created_at) VALUES (?, 0, ?)`,
		title, now.Unix())
	if err != nil {
		return Task{}, err
	}

	id, err := res.LastInsertId()
	if err != nil {
		return Task{}, err
	}
	return Task{ID: id, Title: title, CreatedAt: now}, nil
}

func (s *taskStore) update(ctx context.Context, id int64, title string, done bool) (Task, error) {
	res, err := s.db.ExecContext(ctx,
		`UPDATE tasks SET title = ?, done = ? WHERE id = ?`,
		title, boolToInt(done), id)
	if err != nil {
		return Task{}, err
	}

	n, err := res.RowsAffected()
	if err != nil {
		return Task{}, err
	}
	if n == 0 {
		return Task{}, errNotFound
	}
	return s.get(ctx, id)
}

// ---------------------------------------------------------------------
// HTTP helpers
// ---------------------------------------------------------------------

type api struct {
	store  *taskStore
	logger *slog.Logger
}

func (a *api) writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		a.logger.Error("write json failed", "error", err)
	}
}

func (a *api) writeError(w http.ResponseWriter, status int, msg string) {
	a.writeJSON(w, status, map[string]string{"error": msg})
}

// decodeJSON limits body size and rejects unknown fields.
func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	return dec.Decode(dst)
}

func parseID(r *http.Request) (int64, error) {
	return strconv.ParseInt(r.PathValue("id"), 10, 64)
}

// ---------------------------------------------------------------------
// Handlers
// ---------------------------------------------------------------------

func (a *api) listTasks(w http.ResponseWriter, r *http.Request) {
	tasks, err := a.store.list(r.Context())
	if err != nil {
		a.logger.Error("list tasks failed", "error", err)
		a.writeError(w, http.StatusInternalServerError, "failed to list tasks")
		return
	}
	a.writeJSON(w, http.StatusOK, tasks)
}

// GET /tasks/{id}
func (a *api) getTask(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		a.writeError(w, http.StatusBadRequest, "invalid task id")
		return
	}

	task, err := a.store.get(r.Context(), id)
	if errors.Is(err, errNotFound) {
		a.writeError(w, http.StatusNotFound, "task not found")
		return
	}
	if err != nil {
		a.logger.Error("get task failed", "id", id, "error", err)
		a.writeError(w, http.StatusInternalServerError, "failed to get task")
		return
	}
	a.writeJSON(w, http.StatusOK, task)
}

// POST /tasks
func (a *api) createTask(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Title string `json:"title"`
	}
	if err := decodeJSON(w, r, &body); err != nil {
		a.writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	title := strings.TrimSpace(body.Title)
	if title == "" {
		a.writeError(w, http.StatusBadRequest, "title is required")
		return
	}

	task, err := a.store.create(r.Context(), title)
	if err != nil {
		a.logger.Error("create task failed", "error", err)
		a.writeError(w, http.StatusInternalServerError, "failed to create task")
		return
	}
	a.writeJSON(w, http.StatusCreated, task)
}

// PUT /tasks/{id}
func (a *api) updateTask(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		a.writeError(w, http.StatusBadRequest, "invalid task id")
		return
	}

	var body struct {
		Title string `json:"title"`
		Done  bool   `json:"done"`
	}
	if err := decodeJSON(w, r, &body); err != nil {
		a.writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	title := strings.TrimSpace(body.Title)
	if title == "" {
		a.writeError(w, http.StatusBadRequest, "title is required")
		return
	}

	task, err := a.store.update(r.Context(), id, title, body.Done)
	if errors.Is(err, errNotFound) {
		a.writeError(w, http.StatusNotFound, "task not found")
		return
	}
	if err != nil {
		a.logger.Error("update task failed", "id", id, "error", err)
		a.writeError(w, http.StatusInternalServerError, "failed to update task")
		return
	}
	a.writeJSON(w, http.StatusOK, task)
}

func (a *api) routes() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /tasks", a.listTasks)
	mux.HandleFunc("POST /tasks", a.createTask)
	mux.HandleFunc("GET /tasks/{id}", a.getTask)
	mux.HandleFunc("PUT /tasks/{id}", a.updateTask)
	return mux
}

// ---------------------------------------------------------------------
// main
// ---------------------------------------------------------------------

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	if err := run(logger); err != nil {
		logger.Error("fatal", "error", err)
		os.Exit(1)
	}
}

func run(logger *slog.Logger) error {
	db, err := sql.Open("sqlite3", "file:tasks.db?_journal_mode=WAL&_busy_timeout=5000")
	if err != nil {
		return err
	}
	defer db.Close()

	store := newTaskStore(db)

	migrateCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := store.migrate(migrateCtx); err != nil {
		return err
	}

	a := &api{store: store, logger: logger}

	srv := &http.Server{
		Addr:              ":8080",
		Handler:           a.routes(),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	// Graceful shutdown on Ctrl+C / SIGTERM.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	errCh := make(chan error, 1)
	go func() {
		logger.Info("server listening", "addr", srv.Addr)
		errCh <- srv.ListenAndServe()
	}()

	select {
	case err := <-errCh:
		if !errors.Is(err, http.ErrServerClosed) {
			return err
		}
	case <-ctx.Done():
		logger.Info("shutting down")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		return srv.Shutdown(shutdownCtx)
	}
	return nil
}
