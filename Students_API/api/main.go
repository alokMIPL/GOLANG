package api
package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"
	"unicode/utf8"

	_ "modernc.org/sqlite" // pure-Go SQLite driver
)

const (
	maxBodyBytes = 1 << 20 // 1 MiB
	maxTitleLen  = 200
)

var errNotFound = errors.New("task not found")

// ---------- model ----------

type Task struct {
	ID        int64  `json:"id"`
	Title     string `json:"title"`
	Done      bool   `json:"done"`
	CreatedAt int64  `json:"created_at"`
}

// ---------- storage ----------

type taskStore struct{ db *sql.DB }

func newTaskStore(db *sql.DB) *taskStore { return &taskStore{db: db} }

func (s *taskStore) migrate(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS tasks (
		id         INTEGER PRIMARY KEY AUTOINCREMENT,
		title      TEXT    NOT NULL,
		done       INTEGER NOT NULL DEFAULT 0,
		created_at INTEGER NOT NULL
	)`)
	return err
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func (s *taskStore) list(ctx context.Context) ([]Task, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, title, done, created_at FROM tasks ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	tasks := []Task{} // non-nil so JSON encodes as [] rather than null
	for rows.Next() {
		var t Task
		var done int
		if err := rows.Scan(&t.ID, &t.Title, &done, &t.CreatedAt); err != nil {
			return nil, err
		}
		t.Done = done != 0
		tasks = append(tasks, t)
	}
	return tasks, rows.Err()
}

func (s *taskStore) get(ctx context.Context, id int64) (Task, error) {
	var t Task
	var done int
	err := s.db.QueryRowContext(ctx,
		`SELECT id, title, done, created_at FROM tasks WHERE id = ?`, id,
	).Scan(&t.ID, &t.Title, &done, &t.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return Task{}, errNotFound
	}
	if err != nil {
		return Task{}, err
	}
	t.Done = done != 0
	return t, nil
}

func (s *taskStore) create(ctx context.Context, title string) (Task, error) {
	now := time.Now().Unix()
	res, err := s.db.ExecContext(ctx,
		`INSERT INTO tasks (title, done, created_at) VALUES (?, 0, ?)`, title, now)
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

func (s *taskStore) delete(ctx context.Context, id int64) error {
	res, err := s.db.ExecContext(ctx, `DELETE FROM tasks WHERE id = ?`, id)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return errNotFound
	}
	return nil
}

// ---------- HTTP helpers ----------

type api struct{ store *taskStore }

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Printf("writeJSON: %v", err)
	}
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func parseID(r *http.Request) (int64, error) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		return 0, errors.New("invalid task id")
	}
	return id, nil
}

// decodeBody reads a size-limited JSON body and rejects unknown fields.
func decodeBody(w http.ResponseWriter, r *http.Request, dst any) error {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	return dec.Decode(dst)
}

func validateTitle(raw string) (string, string) {
	title := strings.TrimSpace(raw)
	if title == "" {
		return "", "title is required"
	}
	if utf8.RuneCountInString(title) > maxTitleLen {
		return "", "title must be at most 200 characters"
	}
	return title, ""
}

// ---------- handlers ----------

// GET /tasks
func (a *api) listTasks(w http.ResponseWriter, r *http.Request) {
	tasks, err := a.store.list(r.Context())
	if err != nil {
		log.Printf("listTasks: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to list tasks")
		return
	}
	writeJSON(w, http.StatusOK, tasks)
}

// POST /tasks
func (a *api) createTask(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Title string `json:"title"`
	}
	if err := decodeBody(w, r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	title, msg := validateTitle(body.Title)
	if msg != "" {
		writeError(w, http.StatusBadRequest, msg)
		return
	}

	task, err := a.store.create(r.Context(), title)
	if err != nil {
		log.Printf("createTask: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to create task")
		return
	}
	w.Header().Set("Location", "/tasks/"+strconv.FormatInt(task.ID, 10))
	writeJSON(w, http.StatusCreated, task)
}

// GET /tasks/{id}
func (a *api) getTask(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	task, err := a.store.get(r.Context(), id)
	if errors.Is(err, errNotFound) {
		writeError(w, http.StatusNotFound, "task not found")
		return
	}
	if err != nil {
		log.Printf("getTask: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to get task")
		return
	}
	writeJSON(w, http.StatusOK, task)
}

// PATCH /tasks/{id}  — send either or both of "title" and "done"
func (a *api) updateTask(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	var body struct {
		Title *string `json:"title"`
		Done  *bool   `json:"done"`
	}
	if err := decodeBody(w, r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if body.Title == nil && body.Done == nil {
		writeError(w, http.StatusBadRequest, "nothing to update")
		return
	}

	current, err := a.store.get(r.Context(), id)
	if errors.Is(err, errNotFound) {
		writeError(w, http.StatusNotFound, "task not found")
		return
	}
	if err != nil {
		log.Printf("updateTask get: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to update task")
		return
	}

	title, done := current.Title, current.Done
	if body.Title != nil {
		t, msg := validateTitle(*body.Title)
		if msg != "" {
			writeError(w, http.StatusBadRequest, msg)
			return
		}
		title = t
	}
	if body.Done != nil {
		done = *body.Done
	}

	task, err := a.store.update(r.Context(), id, title, done)
	if errors.Is(err, errNotFound) { // deleted between get and update
		writeError(w, http.StatusNotFound, "task not found")
		return
	}
	if err != nil {
		log.Printf("updateTask: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to update task")
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

	err = a.store.delete(r.Context(), id)
	if errors.Is(err, errNotFound) {
		writeError(w, http.StatusNotFound, "task not found")
		return
	}
	if err != nil {
		log.Printf("deleteTask: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to delete task")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ---------- wiring ----------

func (a *api) routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /tasks", a.listTasks)
	mux.HandleFunc("POST /tasks", a.createTask)
	mux.HandleFunc("GET /tasks/{id}", a.getTask)
	mux.HandleFunc("PATCH /tasks/{id}", a.updateTask)
	mux.HandleFunc("DELETE /tasks/{id}", a.deleteTask)
	return logRequests(mux)
}

func logRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s (%s)", r.Method, r.URL.Path, time.Since(start))
	})
}

func main() {
	dsn := os.Getenv("DB_PATH")
	if dsn == "" {
		dsn = "tasks.db"
	}
	addr := os.Getenv("ADDR")
	if addr == "" {
		addr = ":8080"
	}

	db, err := sql.Open("sqlite", dsn+"?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)")
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	defer db.Close()
	db.SetMaxOpenConns(1) // SQLite: one writer at a time

	store := newTaskStore(db)
	if err := store.migrate(context.Background()); err != nil {
		log.Fatalf("migrate: %v", err)
	}

	srv := &http.Server{
		Addr:              addr,
		Handler:           (&api{store: store}).routes(),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	// Graceful shutdown on Ctrl+C / SIGTERM.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		log.Printf("listening on %s", addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("server: %v", err)
		}
	}()

	<-ctx.Done()
	log.Println("shutting down...")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("shutdown: %v", err)
	}
}