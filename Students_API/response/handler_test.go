package response
package response

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-playground/validator/v10"
)

func decode(t *testing.T, rec *httptest.ResponseRecorder) apiResponse {
	t.Helper()
	var resp apiResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response body: %v", err)
	}
	return resp
}

func TestExternal_Success(t *testing.T) {
	h := &Handler{Fetch: func(ctx context.Context) (*CatFactResponse, error) {
		return &CatFactResponse{Fact: "Cats sleep a lot", Length: 16}, nil
	}}

	rec := httptest.NewRecorder()
	h.External(rec, httptest.NewRequest(http.MethodGet, "/external", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	resp := decode(t, rec)
	if !resp.OK {
		t.Errorf("expected ok=true")
	}
	if resp.External == nil || resp.External.Fact != "Cats sleep a lot" {
		t.Errorf("unexpected external payload: %+v", resp.External)
	}
}

func TestExternal_UpstreamFailure(t *testing.T) {
	h := &Handler{Fetch: func(ctx context.Context) (*CatFactResponse, error) {
		return nil, errors.New("boom")
	}}

	rec := httptest.NewRecorder()
	h.External(rec, httptest.NewRequest(http.MethodGet, "/external", nil))

	if rec.Code != http.StatusBadGateway {
		t.Fatalf("expected 502, got %d", rec.Code)
	}
	resp := decode(t, rec)
	if resp.OK {
		t.Errorf("expected ok=false, got true")
	}
	if resp.Error == "" {
		t.Errorf("expected an error message")
	}
}

func TestExternal_MethodNotAllowed(t *testing.T) {
	h := NewHandler()
	rec := httptest.NewRecorder()
	h.External(rec, httptest.NewRequest(http.MethodPost, "/external", nil))

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", rec.Code)
	}
}

func TestFetchCatFact_Upstream(t *testing.T) {
	// Fake upstream server so we test the real fetchCatFact offline.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"fact":"Cats purr","length":9}`))
	}))
	defer srv.Close()

	// To test this, make catFactURL a var instead of a const, then:
	// old := catFactURL; catFactURL = srv.URL; defer func() { catFactURL = old }()
	t.Skip("requires catFactURL to be a variable")
}

func TestRateLimiter(t *testing.T) {
	rl := &RateLimiter{
		requests: make(map[string][]time.Time),
		limit:    2,
		window:   time.Minute,
	}

	if !rl.Allow("a") || !rl.Allow("a") {
		t.Fatal("first two requests should be allowed")
	}
	if rl.Allow("a") {
		t.Fatal("third request should be blocked")
	}
	if !rl.Allow("b") {
		t.Fatal("different key should have its own budget")
	}
}

func TestRateLimiter_Middleware(t *testing.T) {
	rl := &RateLimiter{
		requests: make(map[string][]time.Time),
		limit:    1,
		window:   time.Minute,
	}
	handler := rl.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	do := func() int {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.RemoteAddr = "1.2.3.4:5555"
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		return rec.Code
	}

	if code := do(); code != http.StatusOK {
		t.Fatalf("first request: expected 200, got %d", code)
	}
	if code := do(); code != http.StatusTooManyRequests {
		t.Fatalf("second request: expected 429, got %d", code)
	}
}

func TestValidationError(t *testing.T) {
	type input struct {
		Name  string `validate:"required"`
		Email string `validate:"email"`
	}

	err := validator.New().Struct(input{Email: "not-an-email"})
	var verrs validator.ValidationErrors
	if !errors.As(err, &verrs) {
		t.Fatalf("expected ValidationErrors, got %v", err)
	}

	resp := ValidationError(verrs)
	if resp.Status != StatusError {
		t.Errorf("expected status %q, got %q", StatusError, resp.Status)
	}
	want := "Field Name is a required field, Field Email is invalid"
	if resp.Error != want {
		t.Errorf("got %q, want %q", resp.Error, want)
	}
}

func TestGeneralError(t *testing.T) {
	resp := GeneralError(errors.New("oops"))
	if resp.Status != StatusError || resp.Error != "oops" {
		t.Errorf("unexpected response: %+v", resp)
	}
}