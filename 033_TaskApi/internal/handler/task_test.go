package handler

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/yourname/taskapi/internal/store"
)

func TestCreateTask(t *testing.T) {
	h := New(store.NewMemory()).Routes()

	req := httptest.NewRequest(http.MethodPost, "/tasks", strings.NewReader(`{"title":"write tests"}`))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", rec.Code)
	}
}

func TestGetMissingTask(t *testing.T) {
	h := New(store.NewMemory()).Routes()

	req := httptest.NewRequest(http.MethodGet, "/tasks/42", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}
