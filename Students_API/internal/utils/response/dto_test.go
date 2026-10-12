package response

import (
	"context"
	"errors"
	"n/t/http/httptest"
	"net/http"
	"strings"
	"testing"
	"time"
)

func decodeFeedback(body, contentType string) (FactFeedbackRequest, error) {
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	return DecodeJSON[FactFeedbackRequest](httptest.NewRecorder(), req)
}

func TestDecodeJSON(t *testing.T) {
	const ct = "application/json"
	tests := []struct {
		name       string
		body, ct   string
		wantStatus int // 0 = success
		wantMsg    string
	}{
		{"valid", `{"fact":"cats purr","rating":5}`, ct, 0, ""},
		{"valid with charset", `{"fact":"x","rating":1}`, "application/json; charset=utf-8", 0, ""},
		{"missing content type", `{"fact":"x","rating":1}`, "", http.StatusUnsupportedMediaType, "Content-Type"},
		{"wrong content type", `{"fact":"x","rating":1}`, "text/plain", http.StatusUnsupportedMediaType, "Content-Type"},
		{"empty body", ``, ct, http.StatusBadRequest, "must not be empty"},
		{"truncated JSON", `{"fact":`, ct, http.StatusBadRequest, "malformed"},
		{"syntax error", `{fact:1}`, ct, http.StatusBadRequest, "malformed"},
		{"unknown field", `{"fact":"x","rating":1,"admin":true}`, ct, http.StatusBadRequest, `unknown field "admin"`},
		{"wrong type", `{"fact":"x","rating":"five"}`, ct, http.StatusBadRequest, `"rating" must be of type int`},
		{"trailing data", `{"fact":"x","rating":1} {"a":1}`, ct, http.StatusBadRequest, "single JSON object"},
		{"too large", `{"fact":"` + strings.Repeat("a", maxRequestBody) + `"}`, ct, http.StatusRequestEntityTooLarge, "at most"},
		{"missing required", `{"rating":3}`, ct, http.StatusUnprocessableEntity, "Field fact is required"},
		{"rating out of range", `{"fact":"x","rating":9}`, ct, http.StatusUnprocessableEntity, "rating"},
		{"bad email", `{"fact":"x","rating":3,"email":"nope"}`, ct, http.StatusUnprocessableEntity, "email"},
		{"whitespace-only fact", `{"fact":"   ","rating":3}`, ct, http.StatusUnprocessableEntity, "Field fact is required"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := decodeFeedback(tc.body, tc.ct)
			if tc.wantStatus == 0 {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				return
			}
			var de *decodeError
			if !errors.As(err, &de) {
				t.Fatalf("err = %v, want *decodeError", err)
			}
			if de.status != tc.wantStatus {
				t.Errorf("status = %d, want %d (%s)", de.status, tc.wantStatus, de.msg)
			}
			if !strings.Contains(de.msg, tc.wantMsg) {
				t.Errorf("msg = %q, want it to contain %q", de.msg, tc.wantMsg)
			}
		})
	}
}

func TestNormalizeRunsBeforeValidation(t *testing.T) {
	got, err := decodeFeedback(
		`{"fact":"  cats purr ","rating":4,"email":"  Ann@Example.COM "}`, "application/json")
	if err != nil {
		t.Fatal(err)
	}
	if got.Fact != "cats purr" || got.Email != "ann@example.com" {
		t.Errorf("not normalized: %+v", got)
	}
}

type failingStore struct{}

func (failingStore) Save(context.Context, Feedback) error { return errors.New("db down") }

func newTestFeedbackHandler(store FeedbackStore) *FeedbackHandler {
	h := NewFeedbackHandler(store, nil)
	h.now = func() time.Time { return time.Unix(1_700_000_000, 0) }
	h.newID = func() string { return "fb_1" }
	return h
}

func postFeedback(h http.Handler, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestFeedbackHandlerCreated(t *testing.T) {
	store := &MemoryFeedbackStore{}
	h := newTestFeedbackHandler(store)

	rec := postFeedback(h, `{"fact":"cats purr","rating":5,"comment":"nice","email":"A@b.com"}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201; body %s", rec.Code, rec.Body)
	}

	resp := decode(t, rec)
	if !resp.OK || resp.Feedback == nil || resp.Feedback.ID != "fb_1" {
		t.Fatalf("unexpected response: %+v", resp)
	}
	if strings.Contains(rec.Body.String(), "a@b.com") {
		t.Error("response leaked the email address")
	}

	saved := store.All()
	if len(saved) != 1 || saved[0].Email != "a@b.com" {
		t.Errorf("stored = %+v, want one item with the normalized email", saved)
	}
}

func TestFeedbackHandlerErrors(t *testing.T) {
	t.Run("validation is 422 with ok=false", func(t *testing.T) {
		rec := postFeedback(newTestFeedbackHandler(&MemoryFeedbackStore{}), `{"rating":3}`)
		if rec.Code != http.StatusUnprocessableEntity {
			t.Fatalf("status = %d, want 422", rec.Code)
		}
		if resp := decode(t, rec); resp.OK || resp.Error == "" {
			t.Errorf("unexpected response: %+v", resp)
		}
	})

	t.Run("store failure is 500 and hides details", func(t *testing.T) {
		rec := postFeedback(newTestFeedbackHandler(failingStore{}), `{"fact":"x","rating":3}`)
		if rec.Code != http.StatusInternalServerError {
			t.Fatalf("status = %d, want 500", rec.Code)
		}
		if strings.Contains(rec.Body.String(), "db down") {
			t.Error("response leaked the internal error")
		}
	})

	t.Run("GET is 405", func(t *testing.T) {
		rec := httptest.NewRecorder()
		newTestFeedbackHandler(&MemoryFeedbackStore{}).
			ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
		if rec.Code != http.StatusMethodNotAllowed || rec.Header().Get("Allow") != "POST" {
			t.Fatalf("status = %d, Allow = %q", rec.Code, rec.Header().Get("Allow"))
		}
	})
}
