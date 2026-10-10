package response

import (
	"math"
	"net"
	"net/http"
	"strconv"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

type visitor struct {
	lim  *rate.Limiter
	seen time.Time
}

// RateLimiter is a per-client token bucket. Idle clients are swept
// periodically so the map cannot grow without bound.
type RateLimiter struct {
	limit rate.Limit
	burst int
	ttl   time.Duration
	now   func() time.Time

	mu        sync.Mutex
	visitors  map[string]*visitor
	lastSweep time.Time
}

func NewRateLimiter(limit rate.Limit, burst int) *RateLimiter {
	return &RateLimiter{
		limit:    limit,
		burst:    burst,
		ttl:      10 * time.Minute,
		now:      time.Now,
		visitors: make(map[string]*visitor),
	}
}

func (rl *RateLimiter) allow(key string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := rl.now()
	if now.Sub(rl.lastSweep) > rl.ttl {
		for k, v := range rl.visitors {
			if now.Sub(v.seen) > rl.ttl {
				delete(rl.visitors, k)
			}
		}
		rl.lastSweep = now
	}

	v, ok := rl.visitors[key]
	if !ok {
		v = &visitor{lim: rate.NewLimiter(rl.limit, rl.burst)}
		rl.visitors[key] = v
	}
	v.seen = now
	return v.lim.AllowN(now, 1)
}

func (rl *RateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !rl.allow(clientIP(r)) {
			if rl.limit > 0 {
				secs := int(math.Ceil(1 / float64(rl.limit)))
				w.Header().Set("Retry-After", strconv.Itoa(secs))
			}
			_ = WriteJSON(w, http.StatusTooManyRequests, apiResponse{
				OK:    false,
				Error: "rate limit exceeded",
			})
			return
		}
		next.ServeHTTP(w, r)
	})
}

// clientIP uses the connection's address without the port. It deliberately
// ignores X-Forwarded-For: trust that header only behind a proxy you control,
// and resolve it in that proxy-aware layer, not here.
func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
