package response

import (
	"math"
	"net"
	"net/http"
	"strconv"
	"sync"
	"time"
)

// maxCleanupInterval caps how long idle keys can linger, even for large windows.
const maxCleanupInterval = time.Minute

// bucket holds the request timestamps for one key. It grows up to `limit`
// entries and is then used as a ring buffer, so every operation is O(1).
type bucket struct {
	times []time.Time
	head  int // index of the oldest entry once the ring is full
}

// newest returns the most recent timestamp. Only call on a non-empty bucket.
func (b *bucket) newest() time.Time {
	return b.times[(b.head+len(b.times)-1)%len(b.times)]
}

// RateLimiter is a per-key sliding-window limiter.
type RateLimiter struct {
	mu      sync.Mutex
	buckets map[string]*bucket
	limit   int
	window  time.Duration
	keyFunc func(*http.Request) string
	now     func() time.Time // replaceable in tests
	done    chan struct{}
	once    sync.Once
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
		buckets: make(map[string]*bucket),
		limit:   limit,
		window:  window,
		keyFunc: remoteIP,
		now:     time.Now,
		done:    make(chan struct{}),
	}
	go rl.cleanupLoop()
	return rl
}

// WithKeyFunc overrides how the client key is derived (e.g. API key, or
// X-Forwarded-For when behind a trusted proxy). Call before serving.
// Never derive the key from headers a client can spoof unless a trusted proxy
// sets them, otherwise attackers can dodge the limit and bloat memory.
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
	now := rl.now()
	cutoff := now.Add(-rl.window)

	rl.mu.Lock()
	defer rl.mu.Unlock()

	b := rl.buckets[key]
	if b == nil {
		b = &bucket{}
		rl.buckets[key] = b
	}

	// Fewer than `limit` entries ever recorded: always room.
	if len(b.times) < rl.limit {
		b.times = append(b.times, now)
		return true, 0
	}

	// Ring is full: the oldest entry decides.
	oldest := b.times[b.head]
	if oldest.After(cutoff) {
		return false, oldest.Add(rl.window).Sub(now)
	}

	// Oldest has expired, so overwrite it with this request.
	b.times[b.head] = now
	b.head = (b.head + 1) % rl.limit
	return true, 0
}

func (rl *RateLimiter) cleanupLoop() {
	interval := rl.window
	if interval > maxCleanupInterval {
		interval = maxCleanupInterval
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-rl.done:
			return
		case <-ticker.C:
			rl.cleanup()
		}
	}
}

// cleanup removes keys whose most recent request is outside the window.
func (rl *RateLimiter) cleanup() {
	cutoff := rl.now().Add(-rl.window)

	rl.mu.Lock()
	defer rl.mu.Unlock()

	for key, b := range rl.buckets {
		if len(b.times) == 0 || !b.newest().After(cutoff) {
			delete(rl.buckets, key)
		}
	}
}

// Middleware rejects requests over the limit with 429 and a Retry-After header.
func (rl *RateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ok, retry := rl.allow(rl.keyFunc(r))
		if !ok {
			secs := int(math.Ceil(retry.Seconds()))
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
