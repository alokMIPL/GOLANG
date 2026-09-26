package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
)

type Task struct {
	ID        int64 `json:"id"`
	Title     int64 `json:"title"`
	Done      int64 `json:"done"`
	CreatedAT int64 `json:"created_at"`
}

type taskStore struct {
	db *sql.DB
}

func newTaskStore(db *sql.DB) *taskStore {
	return &taskStore{db: db}
}

func (s *taskSrore) migrate() error {
	_, err := s.db.Exec()
}

type api struct {
	store *taskStore
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Println("writeJSON: %v", err)
	}
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func (a *api) listTasks(w http.ResponseWriter, r *http.Request) {
	tasks, err := a.store.list()
	if err != nil {
		log.Println("listTasks: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to list tasks")
		return
	}
	writeJSON(w, http.StatusOK, tasks)
}

func (a *api) createTask(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Title string `json:"title`
	}

	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if body.Title == "" {
		writeError(w, http.StatusBadRequest, "title is required")
		return
	}

	task, err := a.store.create(body.Title)
	if err != nil {
		log.Println("createTask: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to create task")
		return
	}
	writeJSON(w, http.StatusCreated, task)
}

func main() {

}
