package api

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"taskmanager/tasks"
)

type Server struct {
	mgr *tasks.Manager
}

func NewServer(mgr *tasks.Manager) *Server {
	return &Server{mgr: mgr}
}

func (s *Server) Routes() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("/tasks", s.handleTasks)
	mux.HandleFunc("/tasks/run", s.handleRun)
	mux.HandleFunc("/tasks/summary", s.handleSummary)
	return mux
}

// GET /tasks -> list all tasks
// POST /tasks -> create a new task {"name": "..."}
func (s *Server) handleTasks(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		writeJSON(w, http.StatusOK, s.mgr.All())

	case http.MethodPost:
		var body struct {
			Name string `json:"name"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeError(w, http.StatusBadRequest, "invalid JSON body")
			return
		}
		id := len(s.mgr.All()) + 1
		t, err := tasks.NewTask(id, body.Name)
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		s.mgr.Add(t)
		writeJSON(w, http.StatusCreated, t)

	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

// POST /tasks/run -> runs all pending tasks with a 2s timeout, 3 workers
func (s *Server) handleRun(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "use POST")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	completed := s.mgr.RunAll(ctx, 3)
	writeJSON(w, http.StatusOK, completed)
}

// GET /tasks/summary -> status counts
func (s *Server) handleSummary(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, s.mgr.Summary())
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
