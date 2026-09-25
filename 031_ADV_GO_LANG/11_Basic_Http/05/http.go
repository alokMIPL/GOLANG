package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"sync"
)

// ---- Shared helpers ----

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]any{"ok": false, "error": msg})
}

// ---- In-memory "database" ----

type User struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type UserStore struct {
	mu     sync.Mutex
	users  map[int]User
	nextID int
}

func NewUserStore() *UserStore {
	return &UserStore{users: make(map[int]User), nextID: 1}
}

func (s *UserStore) Create(name string) User {
	s.mu.Lock()
	defer s.mu.Unlock()
	u := User{ID: s.nextID, Name: name}
	s.users[u.ID] = u
	s.nextID++
	return u
}

func (s *UserStore) Get(id int) (User, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	u, ok := s.users[id]
	return u, ok
}

func (s *UserStore) List() []User {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]User, 0, len(s.users))
	for _, u := range s.users {
		out = append(out, u)
	}
	return out
}

// ---- Handlers (CRUD with path params) ----

func usersHandler(store *UserStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			writeJSON(w, http.StatusOK, store.List())
		case http.MethodPost:
			var body struct {
				Name string `json:"name"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Name == "" {
				writeError(w, http.StatusBadRequest, "name is required")
				return
			}
			writeJSON(w, http.StatusCreated, store.Create(body.Name))
		default:
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		}
	}
}

// Uses Go 1.22+ path params: registered as "/users/{id}"
func userByIDHandler(store *UserStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.Atoi(r.PathValue("id"))
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid id")
			return
		}

		switch r.Method {
		case http.MethodGet:
			u, ok := store.Get(id)
			if !ok {
				writeError(w, http.StatusNotFound, "user not found")
				return
			}
			writeJSON(w, http.StatusOK, u)
		case http.MethodDelete:
			if !store.Delete(id) {
				writeError(w, http.StatusNotFound, "user not found")
				return
			}
			w.WriteHeader(http.StatusNoContent)
		default:
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		}
	}
}

// ---- File upload handler ----

func uploadHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "only POST is allowed")
		return
	}

	// limit request body to 10MB
	r.Body = http.MaxBytesReader(w, r.Body, 10<<20)

	file, header, err := r.FormFile("file")
	if err != nil {
		writeError(w, http.StatusBadRequest, "missing file field")
		return
	}
	defer file.Close()

	dst, err := os.Create("./uploads/" + header.Filename)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not save file")
		return
	}
	defer dst.Close()

	if _, err := io.Copy(dst, file); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to write file")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"ok":       true,
		"filename": header.Filename,
		"size":     header.Size,
	})
}

// ---- Middleware chaining ----

type Middleware func(http.Handler) http.Handler

func recoverMW(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				log.Printf("panic recovered: %v", err)
				writeError(w, http.StatusInternalServerError, "internal server error")
			}
		}()
		next.ServeHTTP(w, r)
	})
}

func corsMW(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// ---- Main ----

func main() {
	os.MkdirAll("./uploads", 0755)

	store := NewUserStore()
	mux := http.NewServeMux()

	mux.HandleFunc("/users", usersHandler(store))
	mux.HandleFunc("/users/{id}", userByIDHandler(store))
	mux.HandleFunc("/upload", uploadHandler)

	handler := Chain(mux, recoverMW, loggingMW, corsMW)

	fmt.Println("listening on :5000")
	log.Fatal(http.ListenAndServe(":5000", handler))
}
