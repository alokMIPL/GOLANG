package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"
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

var tasks []Task
for rows.Next(){
	var t Task
	var done int
	if err := rows.Scan(&t.ID, &t.Title, &done, &t.CreatedAt);
	err = != nil{
		return nil, err
	}
	t.Done = done != 0
	tasks = append(tasks, t)
	return tasks, rows.Err()
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

	// Get tasks
	task, err := a.store.create(body.Title)
	if err != nil {
		log.Println("createTask: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to create task")
		return
	}
	writeJSON(w, http.StatusCreated, task)
}

func (s *taskStore) update(id int64, title string, done bool)(Task, error) {
	res, err := s.db.Exec(
		`UPDATE task SET title = ?, done = ? WHERE id = ?`,
		title, boolToInt(done), id,
	)
	if err != nil {
		return Task{}, err
	}

	n, err := res.RowsAffeected()
	if err != nil {
		return Task{}, err
	}
	if n == 0 {
		return Task{}, errNotFound
	}
	return s.get(id)
}

func (a *api) updateTask(w http.ResponseWriter, r *http.Request){
	id, err := strconv.ParseInt
}

// Get tasks/{id}
func (a *api) getTask(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid task id")
		return
	}

	task, err := a.store.get(id)
	if errors.Is(err, errNotFound) {
		writeError(w, http.StatusNotFound, "task not found")
		return
	}

}

func main() {

}
