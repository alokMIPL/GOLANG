package notes

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
)

// ---- helpers --------------------------------------------------------------

func newTestAPI(t *testing.T) http.Handler {
	t.Helper()
	quiet := slog.New(slog.NewTextHandler(io.Discard, nil))
	h := NewHandler(newTestService(), quiet)
	return Chain(h.Routes(), HeaderUserAuth("X-User-ID"))
}

func do(t *testing.T, h http.Handler, method, target, user, body string) *httptest.ResponseRecorder {
	t.Helper()
	var rd io.Reader
	if body != "" {
		rd = strings.NewReader(body)
	}
	req := httptest.NewRequest(method, target, rd)
	if user != "" {
		req.Header.Set("X-User-ID", user)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func decodeBody[T any](t *testing.T, rec *httptest.ResponseRecorder) T {
	t.Helper()
	var v T
	if err := json.Unmarshal(rec.Body.Bytes(), &v); err != nil {
		t.Fatalf("decode %q: %v", rec.Body.String(), err)
	}
	return v
}

func wantStatus(t *testing.T, rec *httptest.ResponseRecorder, want int) {
	t.Helper()
	if rec.Code != want {
		t.Fatalf("status = %d, want %d; body = %s", rec.Code, want, rec.Body.String())
	}
}

// ---- tests ----------------------------------------------------------------

func TestHTTPLifecycle(t *testing.T) {
	api := newTestAPI(t)

	rec := do(t, api, "POST", "/notes", "alice", `{"title":"Hello","tags":["A","a"]}`)
	wantStatus(t, rec, http.StatusCreated)
	created := decodeBody[Note](t, rec)
	path := "/notes/" + created.ID.Hex()

	if loc := rec.Header().Get("Location"); loc != path {
		t.Errorf("Location = %q, want %q", loc, path)
	}
	if !reflect.DeepEqual(created.Tags, []string{"a"}) {
		t.Errorf("tags = %v", created.Tags)
	}

	rec = do(t, api, "GET", path, "alice", "")
	wantStatus(t, rec, http.StatusOK)

	rec = do(t, api, "PATCH", path, "alice", `{"content":"body","pinned":true}`)
	wantStatus(t, rec, http.StatusOK)
	patched := decodeBody[Note](t, rec)
	if patched.Content != "body" || !patched.Pinned || patched.Title != "Hello" {
		t.Errorf("unexpected patched note: %+v", patched)
	}

	rec = do(t, api, "DELETE", path+"/pin", "alice", "")
	wantStatus(t, rec, http.StatusOK)
	if decodeBody[Note](t, rec).Pinned {
		t.Error("note should be unpinned")
	}

	rec = do(t, api, "PUT", path+"/pin", "alice", "")
	wantStatus(t, rec, http.StatusOK)
	if !decodeBody[Note](t, rec).Pinned {
		t.Error("note should be pinned")
	}

	wantStatus(t, do(t, api, "DELETE", path, "alice", ""), http.StatusNoContent)
	wantStatus(t, do(t, api, "GET", path, "alice", ""), http.StatusNotFound)
}

func TestHTTPValidationReturns422WithFields(t *testing.T) {
	api := newTestAPI(t)

	rec := do(t, api, "POST", "/notes", "alice", `{"title":"   ","tags":["ok"]}`)
	wantStatus(t, rec, http.StatusUnprocessableEntity)

	body := decodeBody[errorBody](t, rec)
	if len(body.Fields) != 1 || body.Fields[0].Field != "title" {
		t.Fatalf("unexpected fields: %+v", body.Fields)
	}
}

func TestHTTPRequiresAuthentication(t *testing.T) {
	api := newTestAPI(t)
	wantStatus(t, do(t, api, "GET", "/notes", "", ""), http.StatusUnauthorized)
	wantStatus(t, do(t, api, "POST", "/notes", "", `{"title":"x"}`), http.StatusUnauthorized)
}

func TestHTTPTenantIsolation(t *testing.T) {
	api := newTestAPI(t)

	rec := do(t, api, "POST", "/notes", "alice", `{"title":"private"}`)
	wantStatus(t, rec, http.StatusCreated)
	path := "/notes/" + decodeBody[Note](t, rec).ID.Hex()

	wantStatus(t, do(t, api, "GET", path, "bob", ""), http.StatusNotFound)
	wantStatus(t, do(t, api, "PATCH", path, "bob", `{"pinned":true}`), http.StatusNotFound)
	wantStatus(t, do(t, api, "DELETE", path, "bob", ""), http.StatusNotFound)

	list := decodeBody[ListResult](t, do(t, api, "GET", "/notes", "bob", ""))
	if list.Total != 0 {
		t.Errorf("bob should see no notes, got %d", list.Total)
	}
}

func TestHTTPRejectsBadBodies(t *testing.T) {
	api := newTestAPI(t)

	tests := []struct {
		name string
		body string
		want int
	}{
		{"empty", "", http.StatusBadRequest},
		{"malformed", `{`, http.StatusBadRequest},
		{"unknown field", `{"title":"x","bogus":1}`, http.StatusBadRequest},
		{"trailing data", `{"title":"x"}{"title":"y"}`, http.StatusBadRequest},
		{"wrong type", `{"title":5}`, http.StatusBadRequest},
		{"too large", `{"title":"` + strings.Repeat("a", maxBodyBytes) + `"}`, http.StatusRequestEntityTooLarge},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			wantStatus(t, do(t, api, "POST", "/notes", "alice", tc.body), tc.want)
		})
	}
}

func TestHTTPInvalidIDAndEmptyUpdate(t *testing.T) {
	api := newTestAPI(t)

	wantStatus(t, do(t, api, "GET", "/notes/zzz", "alice", ""), http.StatusBadRequest)

	rec := do(t, api, "POST", "/notes", "alice", `{"title":"x"}`)
	path := "/notes/" + decodeBody[Note](t, rec).ID.Hex()
	wantStatus(t, do(t, api, "PATCH", path, "alice", `{}`), http.StatusBadRequest)
}

func TestHTTPListParamsAndSearch(t *testing.T) {
	api := newTestAPI(t)

	for _, body := range []string{
		`{"title":"Buy milk","tags":["home"]}`,
		`{"title":"Ideas","content":"oat MILK?","tags":["home","blog"]}`,
		`{"title":"Taxes"}`,
	} {
		wantStatus(t, do(t, api, "POST", "/notes", "alice", body), http.StatusCreated)
	}

	wantStatus(t, do(t, api, "GET", "/notes?limit=abc", "alice", ""), http.StatusUnprocessableEntity)
	wantStatus(t, do(t, api, "GET", "/notes?offset=x", "alice", ""), http.StatusUnprocessableEntity)

	page := decodeBody[ListResult](t, do(t, api, "GET", "/notes?limit=2", "alice", ""))
	if len(page.Notes) != 2 || page.Total != 3 || !page.HasMore {
		t.Errorf("unexpected page: %+v", page)
	}

	byTag := decodeBody[ListResult](t, do(t, api, "GET", "/notes?tag=Blog", "alice", ""))
	if byTag.Total != 1 {
		t.Errorf("tag filter total = %d, want 1", byTag.Total)
	}

	search := decodeBody[ListResult](t, do(t, api, "GET", "/notes?q=milk", "alice", ""))
	if search.Total != 2 {
		t.Errorf("search total = %d, want 2", search.Total)
	}
}

func TestHTTPMethodNotAllowed(t *testing.T) {
	api := newTestAPI(t)
	wantStatus(t, do(t, api, "PUT", "/notes", "alice", `{"title":"x"}`), http.StatusMethodNotAllowed)
}

// ---- middleware tests -----------------------------------------------------

func TestChainOrder(t *testing.T) {
	var order []string
	mark := func(name string) Middleware {
		return func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				order = append(order, name)
				next.ServeHTTP(w, r)
			})
		}
	}
	h := Chain(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		order = append(order, "handler")
	}), mark("A"), mark("B"), mark("C"))

	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/", nil))

	if want := []string{"A", "B", "C", "handler"}; !reflect.DeepEqual(order, want) {
		t.Fatalf("order = %v, want %v", order, want)
	}
}

func TestRecovererReturns500(t *testing.T) {
	quiet := slog.New(slog.NewTextHandler(io.Discard, nil))
	h := Chain(
		http.HandlerFunc(func(http.ResponseWriter, *http.Request) { panic("boom") }),
		RequestLogger(quiet), Recoverer(quiet),
	)

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest("GET", "/", nil))

	wantStatus(t, rec, http.StatusInternalServerError)
	if strings.Contains(rec.Body.String(), "boom") {
		t.Error("panic value must not leak to the client")
	}
}

func TestRequestIDPropagation(t *testing.T) {
	quiet := slog.New(slog.NewTextHandler(io.Discard, nil))
	h := RequestLogger(quiet)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	// A sane incoming ID is echoed back.
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("X-Request-ID", "abc-123")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if got := rec.Header().Get("X-Request-ID"); got != "abc-123" {
		t.Errorf("X-Request-ID = %q, want abc-123", got)
	}

	// An unsafe one is replaced with a generated ID.
	req = httptest.NewRequest("GET", "/", nil)
	req.Header.Set("X-Request-ID", "bad id\nwith newline")
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if got := rec.Header().Get("X-Request-ID"); got == "" || strings.ContainsAny(got, " \n") {
		t.Errorf("unsafe request id was not replaced: %q", got)
	}
}
