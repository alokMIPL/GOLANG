package response

import (
	"context"
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

type fakeClock struct {
	mu sync.Mutex
	t  time.Time
}

func (c *fakeClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.t
}

func (c *fakeClock) Advance(d time.Duration) {
	c.mu.Lock()
	c.t = c.t.Add(d)
	c.mu.Unlock()
}

type fixture struct {
	h     *Handler
	clk   *fakeClock
	calls atomic.Int32
	fail  atomic.Bool
}

func newFixture(opts ...HandlerOption) *fixture {
	fx := &fixture{clk: &fakeClock{t: time.Unix(1_700_000_000, 0)}}
	fetch := func(ctx context.Context) (*CatFactResponse, error) {
		fx.calls.Add(1)
		if fx.fail.Load() {
			return nil, errors.New("boom")
		}
		return &CatFactResponse{Fact: "cats purr", Length: 9}, nil
	}
	base := []HandlerOption{
		withClock(fx.clk.Now),
		WithHandlerLogger(slog.New(slog.NewTextHandler(io.Discard, nil))),
		WithTTL(10 * time.Second),
		WithFailureBackoff(5 * time.Second),
	}
	fx.h = NewHandler(fetch, append(base, opts...)...)
	return fx
}

func (fx *fixture) get(t *testing.T) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	fx.h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/fact", nil))
	return rec
}

func TestMissThenHit(t *testing.T) {
	fx := newFixture()
	if got := fx.get(t).Header().Get("X-Cache"); got != "MISS" {
		t.Fatalf("first request: X-Cache = %q, want MISS", got)
	}
	if got := fx.get(t).Header().Get("X-Cache"); got != "HIT" {
		t.Fatalf("second request: X-Cache = %q, want HIT", got)
	}
	if n := fx.calls.Load(); n != 1 {
		t.Fatalf("upstream calls = %d, want 1", n)
	}
}

func TestStaleServedThenRefreshed(t *testing.T) {
	fx := newFixture(WithBackgroundRefresh(true))
	fx.get(t)
	fx.clk.Advance(11 * time.Second)

	if got := fx.get(t).Header().Get("X-Cache"); got != "STALE" {
		t.Fatalf("X-Cache = %q, want STALE", got)
	}
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if fx.get(t).Header().Get("X-Cache") == "HIT" {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("background refresh never produced a fresh entry")
}

func TestStaleFallbackOnUpstreamError(t *testing.T) {
	fx := newFixture(WithBackgroundRefresh(false))
	fx.get(t)
	fx.clk.Advance(11 * time.Second)
	fx.fail.Store(true)

	rec := fx.get(t)
	if rec.Code != http.StatusOK || rec.Header().Get("X-Cache") != "STALE" {
		t.Fatalf("got %d / %q, want 200 / STALE", rec.Code, rec.Header().Get("X-Cache"))
	}
}

func TestBackoffShieldsFailingUpstream(t *testing.T) {
	fx := newFixture()
	fx.fail.Store(true)

	if code := fx.get(t).Code; code != http.StatusBadGateway {
		t.Fatalf("first failure: code = %d, want 502", code)
	}
	rec := fx.get(t)
	if rec.Code != http.StatusServiceUnavailable || rec.Header().Get("Retry-After") == "" {
		t.Fatalf("in backoff: code = %d, Retry-After = %q", rec.Code, rec.Header().Get("Retry-After"))
	}
	if n := fx.calls.Load(); n != 1 {
		t.Fatalf("upstream calls during backoff = %d, want 1", n)
	}

	fx.clk.Advance(6 * time.Second)
	fx.fail.Store(false)
	if code := fx.get(t).Code; code != http.StatusOK {
		t.Fatalf("after backoff: code = %d, want 200", code)
	}
}

func TestConcurrentMissesCoalesce(t *testing.T) {
	fx := newFixture()
	release := make(chan struct{})
	var calls atomic.Int32
	fx.h.fetch = func(ctx context.Context) (*CatFactResponse, error) {
		calls.Add(1)
		<-release
		return &CatFactResponse{Fact: "cats purr", Length: 9}, nil
	}

	var wg sync.WaitGroup
	for range 20 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			fx.get(t)
		}()
	}
	time.Sleep(50 * time.Millisecond)
	close(release)
	wg.Wait()

	if n := calls.Load(); n != 1 {
		t.Fatalf("upstream calls = %d, want 1", n)
	}
}

func TestMethodNotAllowed(t *testing.T) {
	fx := newFixture()
	rec := httptest.NewRecorder()
	fx.h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/fact", nil))
	if rec.Code != http.StatusMethodNotAllowed || rec.Header().Get("Allow") != http.MethodGet {
		t.Fatalf("got %d, Allow = %q", rec.Code, rec.Header().Get("Allow"))
	}
}
