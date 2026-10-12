package response

import (
	"net"
	"net/http"
	"strconv"
	"sync"
	"time"
)

// RateLimiter is a per-key sliding-window limiter.
type RateLimiter struct {
	mu       sync.Mutex
	requests map[string][]time.Time
	limit    int	window   time.Duration
	keyFunc  func(*http.Request) string
	done     chan struct{}
	once     sync.Once
}

// NewRateLimiter allows `limit` requests per `window` for each key.
// It starts a background cleanup goroutine; call Close when finished.
func NewRateLimiter(limit int, window time.Duration) *RateLimiter {
	if limit <= 0 {
		limit = 1
	}
	if window <= 0 {
		window = time.Minute
	}

	rl := &RateLimiter{
		requests: make(map[string][]time.Time),
		limit:    limit,
		window:   window,
		keyFunc:  remoteIP,
		done:     make(chan struct{}),
	}
	go rl.cleanupLoop()
	return rl
}

// WithKeyFunc overrides how the client key is derived (e.g. API key,
// or X-Forwarded-For when behind a trusted proxy). Call before serving.
func (rl *RateLimiter) WithKeyFunc(fn func(*http.Request) string) *RateLimiter {
	if fn != nil {
		rl.keyFunc = fn
	}
	return rl
}

// Close stops the cleanup goroutine. Safe to call multiple times.
func (rl *RateLimiter) Close() { rl.once.Do(func() { close(rl.done) }) }

// Allow reports whether a request for key is permitted right now.
func (rl *RateLimiter) Allow(key string) bool {
	ok, _ := rl.allow(key)
	return ok
}

// allow returns whether the request is permitted and, if not,
// how long until the next slot frees up.
func (rl *RateLimiter) allow(key string) (bool, time.Duration) {
	now := time.Now()
	cutoff := now.Add(-rl.window)

	rl.mu.Lock()
	defer rl.mu.Unlock()

	times := rl.requests[key]

	// Timestamps are appended in order, so expired entries are a prefix.
	// Trim in place instead of allocating a new slice on every call.
	drop := 0
	for drop < len(times) && !times[drop].After(cutoff) {
		drop++
	}
	if drop > 0 {
		n := copy(times, times[drop:])
		times = times[:n]
	}

	if len(times) >= rl.limit {
		rl.requests[key] = times
		return false, times[0].Add(rl.window).Sub(now)
	}

	rl.requests[key] = append(times, now)
	return true, 0
}

func (rl *RateLimiter) cleanupLoop() {
	ticker := time.NewTicker(rl.window)
	defer ticker.Stop()

	for {
		select {
		case <-rl.done:
			return
		case <-ticker.C:
			cutoff := time.Now().Add(-rl.window)
			rl.mu.Lock()
			for key, times := range rl.requests {
				if len(times) == 0 || !times[len(times)-1].After(cutoff) {
					delete(rl.requests, key)
				}
			}
			rl.mu.Unlock()
		}
	}
}

// Middleware rejects requests over the limit with 429 and a Retry-After header.
func (rl *RateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ok, retry := rl.allow(rl.keyFunc(r))
		if !ok {
			secs := int((retry + time.Second - 1) / time.Second) // round up
			if secs < 1 {
				secs = 1
			}
			w.Header().Set("Retry-After", strconv.Itoa(secs))
			_ = WriteJSON(w, http.StatusTooManyRequests, apiResponse{
				OK:    false,
				Error: "rate limit exceeded",
			})
			return
		}
		next.ServeHTTP(w, r)
	})
}

func remoteIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
