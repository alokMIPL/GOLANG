package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/yourname/taskapi/internal/store"
)

// newTestServer returns a fresh handler backed by an empty in-memory store,
// so every test starts from a clean slate.
func newTestServer(t *testing.T) http.Handler {
	t.Helper()
	return New(store.NewMemory()).Routes()
}

// doRequest sends a request to the handler and returns the recorded response.
func doRequest(t *testing.T, h http.Handler, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()

	var req *http.Request
	if body == "" {
		req = httptest.NewRequest(method, path, nil)
	} else {
		req = httptest.NewRequest(method, path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
	}

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

// decodeJSON decodes the response body into v and fails the test on error.
func decodeJSON(t *testing.T, rec *httptest.ResponseRecorder, v any) {
	t.Helper()
	if err := json.NewDecoder(rec.Body).Decode(v); err != nil {
		t.Fatalf("could not decode response body %q: %v", rec.Body.String(), err)
	}
}

func TestCreateTask(t *testing.T) {
	h := newTestServer(t)

	rec := doRequest(t, h, http.MethodPost, "/tasks", `{"title":"write tests"}`)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", rec.Code)
	}
}

func TestCreateTaskReturnsJSON(t *testing.T) {
	h := newTestServer(t)

	rec := doRequest(t, h, http.MethodPost, "/tasks", `{"title":"write tests"}`)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", rec.Code)
	}

	contentType := rec.Header().Get("Content-Type")
	if !strings.Contains(contentType, "application/json") {
		t.Errorf("expected JSON content type, got %q", contentType)
	}

	var got map[string]any
	decodeJSON(t, rec, &got)

	if got["title"] != "write tests" {
		t.Errorf("expected title %q, got %v", "write tests", got["title"])
	}
}

func TestCreateTaskInvalidInput(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{name: "malformed json", body: `{"title":`},
		{name: "not json at all", body: `hello`},
		{name: "wrong type for title", body: `{"title":123}`},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h := newTestServer(t)

			rec := doRequest(t, h, http.MethodPost, "/tasks", tc.body)

			if rec.Code != http.StatusBadRequest {
				t.Fatalf("expected 400, got %d (body: %s)", rec.Code, rec.Body.String())
			}
		})
	}
}

func TestGetMissingTask(t *testing.T) {
	h := newTestServer(t)

	rec := doRequest(t, h, http.MethodGet, "/tasks/42", "")

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}

func TestListTasksEmpty(t *testing.T) {
	h := newTestServer(t)

	rec := doRequest(t, h, http.MethodGet, "/tasks", "")

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var got []map[string]any
	decodeJSON(t, rec, &got)

	if len(got) != 0 {
		t.Errorf("expected an empty list, got %d tasks", len(got))
	}
}

func TestListTasksAfterCreate(t *testing.T) {
	h := newTestServer(t)

	titles := []string{"first", "second", "third"}
	for _, title := range titles {
		rec := doRequest(t, h, http.MethodPost, "/tasks", `{"title":"`+title+`"}`)
		if rec.Code != http.StatusCreated {
			t.Fatalf("creating %q: expected 201, got %d", title, rec.Code)
		}
	}

	rec := doRequest(t, h, http.MethodGet, "/tasks", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var got []map[string]any
	decodeJSON(t, rec, &got)

	if len(got) != len(titles) {
		t.Fatalf("expected %d tasks, got %d", len(titles), len(got))
	}
}

func TestUnknownRoute(t *testing.T) {
	h := newTestServer(t)

	rec := doRequest(t, h, http.MethodGet, "/does-not-exist", "")

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}

func TestMethodNotAllowed(t *testing.T) {
	h := newTestServer(t)

	rec := doRequest(t, h, http.MethodPatch, "/tasks", "")

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", rec.Code)
	}
}
