package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// newTestHandler builds the full router on top of a fresh in-memory DB.
func newTestHandler(t *testing.T) http.Handler {
	t.Helper()

	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	// Each new connection to :memory: gets its own empty database,
	// so pin the pool to a single connection.
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { db.Close() })

	store := newTaskStore(db)
	if err := store.migrate(context.Background()); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return (&api{store: store}).routes()
}

func do(t *testing.T, h http.Handler, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	var req *http.Request
	if body == "" {
		req = httptest.NewRequest(method, path, nil)
	} else {
		req = httptest.NewRequest(method, path, strings.NewReader(body))
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func decodeTask(t *testing.T, rec *httptest.ResponseRecorder) Task {
	t.Helper()
	var task Task
	if err := json.NewDecoder(rec.Body).Decode(&task); err != nil {
		t.Fatalf("decode task: %v (body=%q)", err, rec.Body.String())
	}
	return task
}

func TestListEmptyReturnsArray(t *testing.T) {
	h := newTestHandler(t)

	rec := do(t, h, "GET", "/tasks", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if got := strings.TrimSpace(rec.Body.String()); got != "[]" {
		t.Fatalf("body = %q, want []", got)
	}
}

func TestCreateAndGet(t *testing.T) {
	h := newTestHandler(t)

	rec := do(t, h, "POST", "/tasks", `{"title":"  write tests  "}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create status = %d, want 201", rec.Code)
	}
	created := decodeTask(t, rec)
	if created.Title != "write tests" {
		t.Errorf("title = %q, want trimmed %q", created.Title, "write tests")
	}
	if created.Done {
		t.Error("new task should not be done")
	}
	if loc := rec.Header().Get("Location"); loc != "/tasks/1" {
		t.Errorf("Location = %q, want /tasks/1", loc)
	}

	rec = do(t, h, "GET", "/tasks/1", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("get status = %d, want 200", rec.Code)
	}
	got := decodeTask(t, rec)
	if got != created {
		t.Errorf("got %+v, want %+v", got, created)
	}
}

func TestCreateValidation(t *testing.T) {
	h := newTestHandler(t)

	tests := []struct {
		name string
		body string
	}{
		{"malformed json", `{"title":`},
		{"empty title", `{"title":""}`},
		{"whitespace title", `{"title":"   "}`},
		{"missing title", `{}`},
		{"unknown field", `{"title":"x","bogus":1}`},
		{"title too long", `{"title":"` + strings.Repeat("a", maxTitleLen+1) + `"}`},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			rec := do(t, h, "POST", "/tasks", tc.body)
			if rec.Code != http.StatusBadRequest {
				t.Errorf("status = %d, want 400 (body=%s)", rec.Code, rec.Body.String())
			}
		})
	}
}

func TestGetErrors(t *testing.T) {
	h := newTestHandler(t)

	tests := []struct {
		name   string
		path   string
		status int
	}{
		{"not found", "/tasks/999", http.StatusNotFound},
		{"non-numeric id", "/tasks/abc", http.StatusBadRequest},
		{"zero id", "/tasks/0", http.StatusBadRequest},
		{"negative id", "/tasks/-5", http.StatusBadRequest},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			rec := do(t, h, "GET", tc.path, "")
			if rec.Code != tc.status {
				t.Errorf("status = %d, want %d", rec.Code, tc.status)
			}
		})
	}
}

func TestPatch(t *testing.T) {
	h := newTestHandler(t)
	do(t, h, "POST", "/tasks", `{"title":"original"}`)

	// Mark done only; title must be preserved.
	rec := do(t, h, "PATCH", "/tasks/1", `{"done":true}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	task := decodeTask(t, rec)
	if !task.Done || task.Title != "original" {
		t.Errorf("got %+v, want done=true title=original", task)
	}

	// Change title only; done must be preserved.
	rec = do(t, h, "PATCH", "/tasks/1", `{"title":"renamed"}`)
	task = decodeTask(t, rec)
	if !task.Done || task.Title != "renamed" {
		t.Errorf("got %+v, want done=true title=renamed", task)
	}

	// Un-done explicitly with false (pointer field distinguishes from absent).
	rec = do(t, h, "PATCH", "/tasks/1", `{"done":false}`)
	task = decodeTask(t, rec)
	if task.Done {
		t.Errorf("done = true, want false")
	}
}

func TestPatchErrors(t *testing.T) {
	h := newTestHandler(t)
	do(t, h, "POST", "/tasks", `{"title":"x"}`)

	tests := []struct {
		name   string
		path   string
		body   string
		status int
	}{
		{"empty patch", "/tasks/1", `{}`, http.StatusBadRequest},
		{"blank title", "/tasks/1", `{"title":"  "}`, http.StatusBadRequest},
		{"bad json", "/tasks/1", `nope`, http.StatusBadRequest},
		{"bad id", "/tasks/x", `{"done":true}`, http.StatusBadRequest},
		{"missing task", "/tasks/42", `{"done":true}`, http.StatusNotFound},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			rec := do(t, h, "PATCH", tc.path, tc.body)
			if rec.Code != tc.status {
				t.Errorf("status = %d, want %d (body=%s)", rec.Code, tc.status, rec.Body.String())
			}
		})
	}
}

func TestDelete(t *testing.T) {
	h := newTestHandler(t)
	do(t, h, "POST", "/tasks", `{"title":"bye"}`)

	rec := do(t, h, "DELETE", "/tasks/1", "")
	if rec.Code != http.StatusNoContent {
		t.Fatalf("delete status = %d, want 204", rec.Code)
	}
	if rec.Body.Len() != 0 {
		t.Errorf("204 response should have empty body, got %q", rec.Body.String())
	}

	if rec = do(t, h, "GET", "/tasks/1", ""); rec.Code != http.StatusNotFound {
		t.Errorf("get after delete = %d, want 404", rec.Code)
	}
	if rec = do(t, h, "DELETE", "/tasks/1", ""); rec.Code != http.StatusNotFound {
		t.Errorf("second delete = %d, want 404", rec.Code)
	}
}

func TestListOrdering(t *testing.T) {
	h := newTestHandler(t)
	for _, title := range []string{"a", "b", "c"} {
		do(t, h, "POST", "/tasks", `{"title":"`+title+`"}`)
	}

	rec := do(t, h, "GET", "/tasks", "")
	var tasks []Task
	if err := json.NewDecoder(rec.Body).Decode(&tasks); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(tasks) != 3 {
		t.Fatalf("len = %d, want 3", len(tasks))
	}
	for i, want := range []string{"a", "b", "c"} {
		if tasks[i].Title != want {
			t.Errorf("tasks[%d].Title = %q, want %q", i, tasks[i].Title, want)
		}
	}
}

func TestMethodNotAllowed(t *testing.T) {
	h := newTestHandler(t)

	// Go 1.22+ ServeMux returns 405 automatically for known paths
	// registered under other methods.
	rec := do(t, h, "PUT", "/tasks/1", `{}`)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want 405", rec.Code)
	}
}
