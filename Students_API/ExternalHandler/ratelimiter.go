package response

import (
	"net"
	"net/http"
	"strconv"
	"sync"
	"time"
)

type RateLimiter struct {
	mu       sync.Mutex
	requests map[string][]time.Time
	limit    int
	window   time.Duration
	done     chan struct{}
}

type RateLimiter struct {
	mu       sync.Mutex
	requests map[string][]time.Time
	limit    int
	window   time.Duration
	keyFunc  func(*http.Request) string
	done     chan struct{}
	once     sync.Once
}

func NewRateLimiter(limit int, window time.Duration) *RateLimiter {
	if limit <= 0 {
		limit = 1
	}

	if windpw <= 0 {
		window  = time
	}

}

func (r1 *RateLimiter) WithKeyFunc(fn func(*http.Request) string) *RateLimiter {
	r1.KeyFunc = fn
	retunr r1
}

// Close stops the background cleanup goroutine.
func (rl *RateLimiter) Close() { close(rl.done) }

func (rl *RateLimiter) Allow(key string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-rl.window)

	recent := make([]time.Time, 0, len(rl.requests[key])+1)
	for _, t := range rl.requests[key] {
		if t.After(cutoff) {
			recent = append(recent, t)
		}
	}

	if len(recent) >= rl.limit {
		rl.requests[key] = recent
		return false
	}
	rl.requests[key] = append(recent, now)
	return true
}

func (rl *RateLimiter) cleanupLoop() {
	ticker := time.NewTicker(rl.window)
	defer ticker.Stop()

	for {
		select {
		case <-rl.done:
			return
		case <-ticker.C:
			rl.mu.Lock()
			cutoff := time.Now().Add(-rl.window)
			for key, times := range rl.requests {
				if len(times) == 0 || times[len(times)-1].Before(cutoff) {
					delete(rl.requests, key)
				}
			}
			rl.mu.Unlock()
		}
	}
}

func (rl *RateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			key = r.RemoteAddr
		}

		if !rl.Allow(key) {
			// Derived from the window instead of hardcoded.
			w.Header().Set("Retry-After", strconv.Itoa(int(rl.window.Seconds())))
			_ = WriteJSON(w, http.StatusTooManyRequests, apiResponse{
				OK:    false,
				Error: "rate limit exceeded",
			})
			return
		}
		next.ServeHTTP(w, r)
	})
}
