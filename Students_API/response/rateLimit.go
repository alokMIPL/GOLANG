package response

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// newTestLimiter builds a limiter without the background goroutine and with a
// fake clock, so tests are deterministic. Call rl.cleanup() to sweep manually.
func newTestLimiter(limit int, window time.Duration, clk *fakeClock) *RateLimiter {
	return &RateLimiter{
		buckets: make(map[string]*bucket),
		limit:   limit,
		window:  window,
		keyFunc: remoteIP,
		now:     clk.Now,
		done:    make(chan struct{}),
	}
}

func TestRateLimiter_Allow(t *testing.T) {
	t.Parallel()
	rl := newTestLimiter(2, time.Minute, newFakeClock())

	steps := []struct {
		key  string
		want bool
	}{
		{"a", true},
		{"a", true},
		{"a", false}, // over budget
		{"b", true},  // separate key, separate budget
	}
	for i, s := range steps {
		if got := rl.Allow(s.key); got != s.want {
			t.Fatalf("step %d: Allow(%q) = %v, want %v", i, s.key, got, s.want)
		}
	}
}

func TestRateLimiter_SlidingWindow(t *testing.T) {
	t.Parallel()
	clk := newFakeClock()
	rl := newTestLimiter(2, time.Minute, clk)

	rl.Allow("a") // t+0
	clk.Advance(30 * time.Second)
	rl.Allow("a") // t+30s

	if ok, retry := rl.allow("a"); ok || retry != 30*time.Second {
		t.Fatalf("at t+30s: ok=%v retry=%v, want blocked with 30s", ok, retry)
	}

	clk.Advance(30 * time.Second) // t+60s: the first request just left the window
	if !rl.Allow("a") {
		t.Fatal("at t+60s: expected request to be allowed")
	}

	// Now the oldest entry is the t+30s one, so the next slot frees at t+90s.
	if ok, retry := rl.allow("a"); ok || retry != 30*time.Second {
		t.Fatalf("at t+60s (2nd): ok=%v retry=%v, want blocked with 30s", ok, retry)
	}
}

func TestRateLimiter_RingBufferReuse(t *testing.T) {
	t.Parallel()
	clk := newFakeClock()
	rl := newTestLimiter(3, time.Minute, clk)

	for cycle := 0; cycle < 10; cycle++ {
		for i := 0; i < 3; i++ {
			if !rl.Allow("a") {
				t.Fatalf("cycle %d request %d should be allowed", cycle, i)
			}
		}
		if rl.Allow("a") {
			t.Fatalf("cycle %d: 4th request should be blocked", cycle)
		}
		clk.Advance(time.Minute)
	}
}

func TestRateLimiter_Cleanup(t *testing.T) {
	t.Parallel()
	clk := newFakeClock()
	rl := newTestLimiter(2, time.Minute, clk)

	rl.Allow("a")
	rl.Allow("b")
	clk.Advance(30 * time.Second)
	rl.Allow("c")
	clk.Advance(40 * time.Second) // a and b are now 70s old, c is 40s old

	rl.cleanup()

	if len(rl.buckets) != 1 || rl.buckets["c"] == nil {
		t.Fatalf("expected only key c to survive cleanup, got %d keys", len(rl.buckets))
	}
}

func TestRateLimiter_Middleware(t *testing.T) {
	t.Parallel()
	clk := newFakeClock()
	rl := newTestLimiter(1, time.Minute, clk)
	handler := rl.Middleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	do := func(addr string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.RemoteAddr = addr
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		return rec
	}

	assertStatus(t, do("1.2.3.4:5555"), http.StatusOK)

	rec := do("1.2.3.4:6666") // same IP, different port: still the same client
	assertStatus(t, rec, http.StatusTooManyRequests)
	assertHeader(t, rec, "Retry-After", "60")
	resp := decodeResp(t, rec)
	if resp.OK || resp.Error != "rate limit exceeded" {
		t.Errorf("unexpected body: %+v", resp)
	}

	assertStatus(t, do("5.6.7.8:1111"), http.StatusOK) // different client

	clk.Advance(time.Minute)
	assertStatus(t, do("1.2.3.4:7777"), http.StatusOK) // window has passed
}

func TestRateLimiter_WithKeyFunc(t *testing.T) {
	t.Parallel()
	rl := newTestLimiter(1, time.Minute, newFakeClock()).
		WithKeyFunc(func(r *http.Request) string { return r.Header.Get("X-API-Key") })
	handler := rl.Middleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {}))

	do := func(key string) int {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.RemoteAddr = "1.2.3.4:5555" // identical for every request
		req.Header.Set("X-API-Key", key)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		return rec.Code
	}

	if code := do("key-1"); code != http.StatusOK {
		t.Fatalf("key-1 first: %d", code)
	}
	if code := do("key-2"); code != http.StatusOK {
		t.Fatalf("key-2 first: %d", code)
	}
	if code := do("key-1"); code != http.StatusTooManyRequests {
		t.Fatalf("key-1 second: %d, want 429", code)
	}
}

func TestNewRateLimiter_Defaults(t *testing.T) {
	t.Parallel()
	rl := NewRateLimiter(0, 0)
	if rl.limit != 1 || rl.window != time.Minute {
		t.Errorf("defaults = limit %d window %v, want 1 and 1m", rl.limit, rl.window)
	}
	rl.Close()
	rl.Close() // must be safe to call twice
}
