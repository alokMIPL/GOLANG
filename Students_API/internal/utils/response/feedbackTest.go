package response

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDecodeJSON(t *testing.T) {
	tests := []struct {
		name       string
		body, ct   string
		wantStatus int // 0 = success
		wantMsg    string
	}{
		{"valid", `{"fact":"cats purr","rating":5}`, jsonCT, 0, ""},
		{"valid with charset", `{"fact":"x","rating":1}`, "application/json; charset=utf-8", 0, ""},
		{"missing content type", `{"fact":"x","rating":1}`, "", http.StatusUnsupportedMediaType, "Content-Type"},
		{"wrong content type", `{"fact":"x","rating":1}`, "text/plain", http.StatusUnsupportedMediaType, "Content-Type"},
		{"empty body", ``, jsonCT, http.StatusBadRequest, "must not be empty"},
		{"truncated JSON", `{"fact":`, jsonCT, http.StatusBadRequest, "malformed"},
		{"syntax error", `{fact:1}`, jsonCT, http.StatusBadRequest, "malformed"},
		{"unknown field", `{"fact":"x","rating":1,"admin":true}`, jsonCT, http.StatusBadRequest, `unknown field "admin"`},
		{"wrong type", `{"fact":"x","rating":"five"}`, jsonCT, http.StatusBadRequest, `"rating" must be of type int`},
		{"trailing data", `{"fact":"x","rating":1} {"a":1}`, jsonCT, http.StatusBadRequest, "single JSON object"},
		{"too large", `{"fact":"` + strings.Repeat("a", maxRequestBody) + `"}`, jsonCT, http.StatusRequestEntityTooLarge, "at most"},
		{"missing required", `{"rating":3}`, jsonCT, http.StatusUnprocessableEntity, "Field fact is required"},
		{"rating out of range", `{"fact":"x","rating":9}`, jsonCT, http.StatusUnprocessableEntity, "rating"},
		{"bad email", `{"fact":"x","rating":3,"email":"nope"}`, jsonCT, http.StatusUnprocessableEntity, "email"},
		{"whitespace-only fact", `{"fact":"   ","rating":3}`, jsonCT, http.StatusUnprocessableEntity, "Field fact is required"},
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

			de := requireDecodeError(t, err)
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
	t.Run("trims the fact", func(t *testing.T) {
		got, err := decodeFeedback(`{"fact":"  cats purr  ","rating":3}`, jsonCT)
		if err != nil {
			t.Fatal(err)
		}
		if got.Fact != "cats purr" {
			t.Errorf("Fact = %q, want %q", got.Fact, "cats purr")
		}
	})

	t.Run("lowercases the email", func(t *testing.T) {
		got, err := decodeFeedback(`{"fact":"x","rating":3,"email":"A@B.com"}`, jsonCT)
		if err != nil {
			t.Fatal(err)
		}
		if got.Email != "a@b.com" {
			t.Errorf("Email = %q, want %q", got.Email, "a@b.com")
		}
	})
}

func TestFeedbackHandlerCreated(t *testing.T) {
	store := &MemoryFeedbackStore{}
	h := newTestFeedbackHandler(store)

	rec := postFeedback(t, h, `{"fact":"cats purr","rating":5,"comment":"nice","email":"A@b.com"}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201; body %s", rec.Code, rec.Body)
	}

	resp := decode(t, rec)
	if !resp.OK || resp.Feedback == nil || resp.Feedback.ID != "fb_1" {
		t.Fatalf("unexpected response: %+v", resp)
	}
	if !resp.Timestamp.Equal(fixedTime) {
		t.Errorf("Timestamp = %v, want %v", resp.Timestamp, fixedTime)
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
		store := &MemoryFeedbackStore{}
		rec := postFeedback(t, newTestFeedbackHandler(store), `{"rating":3}`)
		if rec.Code != http.StatusUnprocessableEntity {
			t.Fatalf("status = %d, want 422", rec.Code)
		}
		if resp := decode(t, rec); resp.OK || resp.Error == "" {
			t.Errorf("unexpected response: %+v", resp)
		}
		if n := len(store.All()); n != 0 {
			t.Errorf("stored %d items, want 0 after a rejected request", n)
		}
	})
	Run("wrong content type is 415", func(t *testing.T) {
		rec := postRaw(t, newTestFeedbackHandler(&MemoryFeedbackStore{}),
			`{"fact":"x","rating":3}`, "text/plain")
		if rec.Code != http.StatusUnsupportedMediaType {
			t.Fatalf("status = %d, want 415", rec.Code)
		}
	})

	t.Run("malformed JSON is 400", func(t *testing.T) {
		rec := postFeedback(t, newTestFeedbackHandler(&MemoryFeedbackStore{}), `{"fact":`)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", rec.Code)
		}
	})

	t.Run("store failure is 500 and hides details", func(t *testing.T) {
		rec := postFeedback(t, newTestFeedbackHandler(failingStore{}), `{"fact":"x","rating":3}`)
		if rec.Code != http.StatusInternalServerError {
			t.Fatalf("status = %d, want 500", rec.Code)
		}
		if strings.Contains(rec.Body.String(), "db down") {
			t.Error("response leaked the internal error")
		}
		if resp := decode(t, rec); resp.OK {
			t.Errorf("OK = true on a failed save: %+v", resp)
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
