package response
package response

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-playground/validator/v10"
	"golang.org/x/time/rate"
)

func decode(t *testing.T, rec *httptest.ResponseRecorder) apiResponse {
	t.Helper()
	var resp apiResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response body: %v", err)
	}
	return resp
}

func do(h http.Handler, remoteAddr string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = remoteAddr
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestHandlerSuccess(t *testing.T) {
	h := NewHandler(WithFetcher(func(context.Context) (*CatFactResponse, error) {
		return &CatFactResponse{Fact: "cats purr", Length: 9}, nil
	}))

	rec := do(h, "1.2.3.4:1000")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	resp := decode(t, rec)
	if !resp.OK {
		t.Error("expected ok=true")
	}
	if resp.External == nil || resp.External.Fact == "" {
		t.Errorf("expected a non-empty fact, got %+v", resp.External)
	}
}

func TestHandlerUpstreamError(t *testing.T) {
	h := NewHandler(WithFetcher(func(context.Context) (*CatFactResponse, error) {
		return nil, errors.New("boom")
	}))

	rec := do(h, "1.2.3.4:1000")
	if rec.Code != http.StatusBadGateway {
		t.Fatalf("status = %d, want 502", rec.Code)
	}
	if resp := decode(t, rec); resp.OK {
		t.Error("expected ok=false")
	}
}

func TestValidationError(t *testing.T) {
	type input struct {
		Name string `validate:"required"`
	}
	err := validator.New().Struct(input{})

	var ve validator.ValidationErrors
	if !errors.As(err, &ve) {
		t.Fatalf("expected ValidationErrors, got %v", err)
	}
	got := ValidationError(ve)
	if got.OK || got.Error != "Field Name is required" {
		t.Errorf("unexpected response: %+v", got)
	}
}

func TestRateLimiterPerClient(t *testing.T) {
	rl := NewRateLimiter(rate.Every(time_hour), 2)
	h := rl.Middleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Same IP, different ports: must share one bucket.
	for i, addr := range []string{"9.9.9.9:1", "9.9.9.9:2"} {
		if code := do(h, addr).Code; code != http.StatusOK {
			t.Fatalf("request %d: status = %d, want 200", i, code)
		}
	}
	rec := do(h, "9.9.9.9:3")
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("status = %d, want 429", rec.Code)
	}
	if rec.Header().Get("Retry-After") == "" {
		t.Error("missing Retry-After header")
	}
	if resp := decode(t, rec); resp.OK {
		t.Error("expected ok=false")
	}

	// A different client is unaffected.
	if code := do(h, "8.8.8.8:1").Code; code != http.StatusOK {
		t.Fatalf("other client: status = %d, want 200", code)
	}
}