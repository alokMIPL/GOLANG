package response

import (
	"encoding/json"
	"math"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	defaultRateLimit  = 1
	defaultRateWindow = time.Minute
)

type bucket struct {
	entries []time.Time
	head    int
	count   int
}

func newBucket(capacity int) *bucket {
	return &bucket{
		entries: make([]time.Time, capacity),
	}
}

func (b *bucket) evictExpired(cutoff time.Time) {
	cap := len(b.entries)
	for b.count > 0 {
		oldest := b.entries[b.head]
		if !oldest.After(cutoff) {
			b.head = (b.head + 1) % cap
			b.count--
		} else {
			break
		}
	}
}

func (b *bucket) push(t time.Time) {
	cap := len(b.entries)
	tail := (b.head + b.count) % cap
	b.entries[tail] = t
	b.count++
}

func (b *bucket) oldest() time.Time {
	return b.entries[b.head]
}

type KeyFunc func(r *http.Request) string

type RateLimiter struct {
	mu      sync.Mutex
	buckets map[string]*bucket
	limit   int
	window  time.Duration
	keyFunc KeyFunc
	now     func() time.Time

	done      chan struct{}
	closeOnce sync.Once
	wg        sync.WaitGroup
}

func NewRateLimiter(limit int, window time.Duration) *RateLimiter {
	if limit <= 0 {
		limit = defaultRateLimit
	}
	if window <= 0 {
		window = defaultRateWindow
	}

	rl := &RateLimiter{
		buckets: make(map[string]*bucket),
		limit:   limit,
		window:  window,
		keyFunc: remoteIP,
		now:     time.Now,
		done:    make(chan struct{}),
	}

	rl.wg.Add(1)
	go rl.sweepLoop()

	return rl
}

func (rl *RateLimiter) WithKeyFunc(fn KeyFunc) *RateLimiter {
	if fn != nil {
		rl.mu.Lock()
		rl.keyFunc = fn
		rl.mu.Unlock()
	}
	return rl
}

func (rl *RateLimiter) WithClock(now func() time.Time) *RateLimiter {
	if now != nil {
		rl.mu.Lock()
		rl.now = now
		rl.mu.Unlock()
	}
	return rl
}

func (rl *RateLimiter) Allow(key string) bool {
	ok, _ := rl.allow(key)
	return ok
}

func (rl *RateLimiter) allow(key string) (bool, time.Duration) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := rl.now()
	cutoff := now.Add(-rl.window)

	b, exists := rl.buckets[key]
	if !exists {
		b = newBucket(rl.limit)
		rl.buckets[key] = b
	}

	b.evictExpired(cutoff)

	if b.count >= rl.limit {
		retryAfter := b.oldest().Add(rl.window).Sub(now)
		if retryAfter < 0 {
			retryAfter = 0
		}
		return false, retryAfter
	}

	b.push(now)
	return true, 0
}

func (rl *RateLimiter) allowWithMeta(key string) (allowed bool, remaining int, resetAt time.Time, retryAfter time.Duration) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := rl.now()
	cutoff := now.Add(-rl.window)

	b, exists := rl.buckets[key]
	if !exists {
		b = newBucket(rl.limit)
		rl.buckets[key] = b
	}

	b.evictExpired(cutoff)

	if b.count >= rl.limit {
		oldest := b.oldest()
		resetAt = oldest.Add(rl.window)
		retryAfter = resetAt.Sub(now)
		if retryAfter < 0 {
			retryAfter = 0
		}
		return false, 0, resetAt, retryAfter
	}

	b.push(now)
	remaining = rl.limit - b.count
	resetAt = b.oldest().Add(rl.window)
	return true, remaining, resetAt, 0
}

func (rl *RateLimiter) cleanup() {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	cutoff := rl.now().Add(-rl.window)
	for key, b := range rl.buckets {
		b.evictExpired(cutoff)
		if b.count == 0 {
			delete(rl.buckets, key)
		}
	}
}

func (rl *RateLimiter) sweepLoop() {
	defer rl.wg.Done()

	interval := rl.window
	if interval <= 0 {
		interval = defaultRateWindow
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

func (rl *RateLimiter) Close() {
	rl.closeOnce.Do(func() {
		close(rl.done)
	})
	rl.wg.Wait()
}
  
func (rl *RateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rl.mu.Lock()
		keyFn := rl.keyFunc
		limit := rl.limit
		rl.mu.Unlock()

		key := keyFn(r)
		allowed, remaining, resetAt, retryAfter := rl.allowWithMeta(key)

		w.Header().Set("X-RateLimit-Limit", strconv.Itoa(limit))
		w.Header().Set("X-RateLimit-Remaining", strconv.Itoa(remaining))
		w.Header().Set("X-RateLimit-Reset", strconv.FormatInt(resetAt.Unix(), 10))

		if !allowed {
			retrySeconds := int(math.Ceil(retryAfter.Seconds()))
			if retrySec````````````````````````onds < 1 {
				retrySeconds = 1
			}
			w.Header().Set("Retry-After", strconv.Itoa(retrySeconds))
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			w.Header().Set("X-Content-Type-Options", "nosniff")
			w.WriteHeader(http.StatusTooManyRequests)
			_ = json.NewEncoder(w).Encode(apiResponse{
				OK:    false,
				Error: "rate limit exceeded",
			})
			return
		}

		next.ServeHTTP(w, r)
	})
}

func remoteIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		if first, _, ok := strings.Cut(xff, ","); ok {
			xff = first
		}
		if trimmed := strings.TrimSpace(xff); trimmed != "" {
			return trimmed
		}
	}
	if xrip := strings.TrimSpace(r.Header.Get("X-Real-IP")); xrip != "" {
		return xrip
	}

	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
