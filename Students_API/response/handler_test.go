package response

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// ---------- shared helpers ----------

var discardLog = slog.New(slog.NewTextHandler(io.Discard, nil))

// fakeClock gives tests full control over time, so nothing needs to sleep.
type fakeClock struct {
	mu  sync.Mutex
	now time.Time
}

func newFakeClock() *fakeClock {
	return &fakeClock{now: time.Date(2026, time.January, 1, 12, 0, 0, 0, time.UTC)}
}

func (c *fakeClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *fakeClock) Advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = c.now.Add(d)
}

// countingFetcher wraps a fake upstream and counts how often it is called.
type countingFetcher struct {
	calls atomic.Int32
	fn    FactFetcher
}

func (c *countingFetcher) fetch(ctx context.Context) (*CatFactResponse, error) {
	c.calls.Add(1)
	return c.fn(ctx)
}

func factFetcher(fact string) FactFetcher {
	return func(context.Context) (*CatFactResponse, error) {
		return &CatFactResponse{Fact: fact, Length: 16}, nil
	}
}

func newTestHandler(t *testing.T, clk *fakeClock, fetch FactFetcher, opts ...Option) *Handler {
	t.Helper()
	base := []Option{WithFetcher(fetch), WithClock(clk.Now), WithLogger(discardLog)}
	h := NewHandler(append(base, opts...)...)
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_ = h.Close(ctx)
	})
	return h
}

func serve(h http.Handler, method string) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(method, "/external", nil))
	return rec
}

func decodeResp(t *testing.T, rec *httptest.ResponseRecorder) apiResponse {
	t.Helper()
	var resp apiResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response body %q: %v", rec.Body.String(), err)
	}
	return resp
}

func assertStatus(t *testing.T, rec *httptest.ResponseRecorder, want int) {
	t.Helper()
	if rec.Code != want {
		t.Fatalf("status = %d, want %d (body: %s)", rec.Code, want, rec.Body.String())
	}
}

func assertHeader(t *testing.T, rec *httptest.ResponseRecorder, key, want string) {
	t.Helper()
	if got := rec.Header().Get(key); got != want {
		t.Errorf("header %s = %q, want %q", key, got, want)
	}
}

func assertCalls(t *testing.T, cf *countingFetcher, want int32) {
	t.Helper()
	if got := cf.calls.Load(); got != want {
		t.Errorf("upstream calls = %d, want %d", got, want)
	}
}

// ---------- tests ----------

func TestHandler_Success(t *testing.T) {
	t.Parallel()
	h := newTestHandler(t, newFakeClock(), factFetcher("Cats sleep a lot"))

	rec := serve(h, http.MethodGet)
	assertStatus(t, rec, http.StatusOK)
	assertHeader(t, rec, "X-Cache", "MISS")

	resp := decodeResp(t, rec)
	if !resp.OK {
		t.Error("expected ok=true")
	}
	if resp.External == nil {
		t.Fatal("expected external payload")
	}
	if resp.External.Fact != "Cats sleep a lot" || resp.External.Source != factSource {
		t.Errorf("unexpected external payload: %+v", resp.External)
	}
}

func TestHandler_CacheHit(t *testing.T) {
	t.Parallel()
	clk := newFakeClock()
	cf := &countingFetcher{fn: factFetcher("Cats purr")}
	h := newTestHandler(t, clk, cf.fetch, WithTTL(30*time.Second))

	assertStatus(t, serve(h, http.MethodGet), http.StatusOK)
	clk.Advance(10 * time.Second)

	rec := serve(h, http.MethodGet)
	assertStatus(t, rec, http.StatusOK)
	assertHeader(t, rec, "X-Cache", "HIT")
	assertHeader(t, rec, "Age", "10")
	assertHeader(t, rec, "Cache-Control", "public, max-age=20")
	assertCalls(t, cf, 1)

	want := Stats{Hits: 1, Misses: 1, Refreshes: 1}
	if got := h.Stats(); got != want {
		t.Errorf("stats = %+v, want %+v", got, want)
	}
}

func TestHandler_TTLExpiry(t *testing.T) {
	t.Parallel()
	clk := newFakeClock()
	cf := &countingFetcher{fn: factFetcher("Cats purr")}
	h := newTestHandler(t, clk, cf.fetch, WithTTL(10*time.Second))

	serve(h, http.MethodGet)
	clk.Advance(11 * time.Second)

	rec := serve(h, http.MethodGet)
	assertStatus(t, rec, http.StatusOK)
	assertHeader(t, rec, "X-Cache", "MISS")
	assertCalls(t, cf, 2)
}

func TestHandler_ServesStaleOnUpstreamFailure(t *testing.T) {
	t.Parallel()
	clk := newFakeClock()
	var fail atomic.Bool
	cf := &countingFetcher{fn: func(context.Context) (*CatFactResponse, error) {
		if fail.Load() {
			return nil, errors.New("boom")
		}
		return &CatFactResponse{Fact: "Cats purr", Length: 9}, nil
	}}
	h := newTestHandler(t, clk, cf.fetch, WithTTL(10*time.Second), WithMaxStale(time.Minute))

	assertStatus(t, serve(h, http.MethodGet), http.StatusOK) // warm the cache
	fail.Store(true)
	clk.Advance(11 * time.Second)

	rec := serve(h, http.MethodGet)
	assertStatus(t, rec, http.StatusOK)
	assertHeader(t, rec, "X-Cache", "STALE")
	assertHeader(t, rec, "Cache-Control", "public, max-age=0")
	if got := decodeResp(t, rec).External.Fact; got != "Cats purr" {
		t.Errorf("stale fact = %q", got)
	}
	assertCalls(t, cf, 2)

	// Still inside the backoff window: stale is served without touching upstream.
	rec = serve(h, http.MethodGet)
	assertStatus(t, rec, http.StatusOK)
	assertHeader(t, rec, "X-Cache", "STALE")
	assertCalls(t, cf, 2)
}

func TestHandler_StaleTooOld(t *testing.T) {
	t.Parallel()
	clk := newFakeClock()
	var fail atomic.Bool
	cf := &countingFetcher{fn: func(context.Context) (*CatFactResponse, error) {
		if fail.Load() {
			return nil, errors.New("boom")
		}
		return &CatFactResponse{Fact: "Cats purr", Length: 9}, nil
	}}
	h := newTestHandler(t, clk, cf.fetch, WithTTL(10*time.Second), WithMaxStale(10*time.Second))

	serve(h, http.MethodGet)
	fail.Store(true)
	clk.Advance(25 * time.Second) // older than ttl + maxStale

	assertStatus(t, serve(h, http.MethodGet), http.StatusBadGateway)
}

func TestHandler_UpstreamErrors(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		fn   FactFetcher
		want int
	}{
		{"generic error", func(context.Context) (*CatFactResponse, error) { return nil, errors.New("boom") }, http.StatusBadGateway},
		{"timeout", func(context.Context) (*CatFactResponse, error) { return nil, context.DeadlineExceeded }, http.StatusGatewayTimeout},
		{"nil response", func(context.Context) (*CatFactResponse, error) { return nil, nil }, http.StatusBadGateway},
		{"empty fact", func(context.Context) (*CatFactResponse, error) { return &CatFactResponse{}, nil }, http.StatusBadGateway},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			h := newTestHandler(t, newFakeClock(), tt.fn)

			rec := serve(h, http.MethodGet)
			assertStatus(t, rec, tt.want)

			resp := decodeResp(t, rec)
			if resp.OK {
				t.Error("expected ok=false")
			}
			if resp.Error == "" {
				t.Error("expected an error message")
			}
		})
	}
}

func TestHandler_Backoff(t *testing.T) {
	t.Parallel()
	clk := newFakeClock()
	var fail atomic.Bool
	fail.Store(true)
	cf := &countingFetcher{fn: func(context.Context) (*CatFactResponse, error) {
		if fail.Load() {
			return nil, errors.New("boom")
		}
		return &CatFactResponse{Fact: "Cats purr", Length: 9}, nil
	}}
	h := newTestHandler(t, clk, cf.fetch, WithBackoff(2*time.Second))

	assertStatus(t, serve(h, http.MethodGet), http.StatusBadGateway)

	// Inside the backoff window upstream must not be hit again.
	rec := serve(h, http.MethodGet)
	assertStatus(t, rec, http.StatusServiceUnavailable)
	assertHeader(t, rec, "Retry-After", "2")
	assertCalls(t, cf, 1)

	// After the window, upstream is tried again and recovers.
	fail.Store(false)
	clk.Advance(2 * time.Second)
	assertStatus(t, serve(h, http.MethodGet), http.StatusOK)
	assertCalls(t, cf, 2)
}

func TestHandler_Retries(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		failFirst int32 // number of initial calls that fail
		retries   int
		wantCode  int
		wantCalls int32
	}{
		{"recovers after retries", 2, 2, http.StatusOK, 3},
		{"retries exhausted", 99, 1, http.StatusBadGateway, 2},
		{"no retries", 99, 0, http.StatusBadGateway, 1},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			cf := &countingFetcher{}
			cf.fn = func(context.Context) (*CatFactResponse, error) {
				if cf.calls.Load() <= tt.failFirst {
					return nil, errors.New("flaky")
				}
				return &CatFactResponse{Fact: "Cats purr", Length: 9}, nil
			}
			h := newTestHandler(t, newFakeClock(), cf.fetch, WithRetries(tt.retries, time.Millisecond))

			assertStatus(t, serve(h, http.MethodGet), tt.wantCode)
			assertCalls(t, cf, tt.wantCalls)
		})
	}
}

func TestHandler_BackgroundRefresh(t *testing.T) {
	t.Parallel()
	clk := newFakeClock()
	var fact atomic.Value
	fact.Store("old fact")
	cf := &countingFetcher{fn: func(context.Context) (*CatFactResponse, error) {
		return &CatFactResponse{Fact: fact.Load().(string), Length: 8}, nil
	}}
	h := newTestHandler(t, clk, cf.fetch, WithTTL(10*time.Second), WithBackgroundRefresh(true))

	serve(h, http.MethodGet)
	clk.Advance(11 * time.Second)
	fact.Store("new fact")

	// Expired entry is returned immediately while a refresh runs in the background.
	rec := serve(h, http.MethodGet)
	assertStatus(t, rec, http.StatusOK)
	assertHeader(t, rec, "X-Cache", "STALE")
	if got := decodeResp(t, rec).External.Fact; got != "old fact" {
		t.Errorf("stale fact = %q, want old fact", got)
	}

	// Close waits for the background refresh to finish.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := h.Close(ctx); err != nil {
		t.Fatalf("Close: %v", err)
	}

	rec = serve(h, http.MethodGet)
	assertHeader(t, rec, "X-Cache", "HIT")
	if got := decodeResp(t, rec).External.Fact; got != "new fact" {
		t.Errorf("refreshed fact = %q, want new fact", got)
	}
	assertCalls(t, cf, 2)
}

func TestHandler_SingleFlight(t *testing.T) {
	t.Parallel()
	const n = 20

	started := make(chan struct{})
	release := make(chan struct{})
	var once sync.Once
	cf := &countingFetcher{fn: func(context.Context) (*CatFactResponse, error) {
		once.Do(func() { close(started) })
		<-release
		return &CatFactResponse{Fact: "Cats purr", Length: 9}, nil
	}}
	h := newTestHandler(t, newFakeClock(), cf.fetch)

	var wg sync.WaitGroup
	codes := make([]int, n)
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			codes[i] = serve(h, http.MethodGet).Code
		}(i)
	}

	select {
	case <-started:
	case <-time.After(5 * time.Second):
		t.Fatal("upstream was never called")
	}
	close(release)
	wg.Wait()

	assertCalls(t, cf, 1)
	for i, code := range codes {
		if code != http.StatusOK {
			t.Errorf("request %d: status = %d, want 200", i, code)
		}
	}
}

func TestHandler_Methods(t *testing.T) {
	t.Parallel()
	tests := []struct {
		method string
		want   int
	}{
		{http.MethodGet, http.StatusOK},
		{http.MethodHead, http.StatusOK},
		{http.MethodPost, http.StatusMethodNotAllowed},
		{http.MethodPut, http.StatusMethodNotAllowed},
		{http.MethodDelete, http.StatusMethodNotAllowed},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.method, func(t *testing.T) {
			t.Parallel()
			h := newTestHandler(t, newFakeClock(), factFetcher("Cats purr"))

			rec := serve(h, tt.method)
			assertStatus(t, rec, tt.want)
			if tt.want == http.StatusMethodNotAllowed {
				assertHeader(t, rec, "Allow", "GET, HEAD")
				if decodeResp(t, rec).OK {
					t.Error("expected ok=false")
				}
			}
		})
	}
}

func TestHandler_Prime(t *testing.T) {
	t.Parallel()

	t.Run("warms the cache", func(t *testing.T) {
		t.Parallel()
		cf := &countingFetcher{fn: factFetcher("Cats purr")}
		h := newTestHandler(t, newFakeClock(), cf.fetch)

		if err := h.Prime(context.Background()); err != nil {
			t.Fatalf("Prime: %v", err)
		}
		rec := serve(h, http.MethodGet)
		assertStatus(t, rec, http.StatusOK)
		assertHeader(t, rec, "X-Cache", "HIT")
		assertCalls(t, cf, 1)
	})

	t.Run("returns upstream error", func(t *testing.T) {
		t.Parallel()
		h := newTestHandler(t, newFakeClock(), func(context.Context) (*CatFactResponse, error) {
			return nil, errors.New("boom")
		})
		if err := h.Prime(context.Background()); err == nil {
			t.Fatal("expected an error")
		}
	})
}
