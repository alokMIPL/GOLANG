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
