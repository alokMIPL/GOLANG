package response

import (
	"math"
	"net"
	"net/http"
	"net/netip"
	"strconv"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

const (
	sweepInterval = time.Minute
	minIdleTTL    = time.Minute
	maxIdleTTL    = 7 * 24 * time.Hour
)

type clientEntry struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

// RateLimiter is a per-client token-bucket limiter. Each client key gets its
// own bucket holding up to burst tokens that refill at limit tokens/second.
type RateLimiter struct {
	limit   rate.Limit
	burst   int
	ttl     time.Duration // 0 means "never sweep"
	keyFunc func(*http.Request) string
	now     func() time.Time

	mu        sync.Mutex
	clients   map[string]*clientEntry
	lastSweep time.Time
}

// RateLimiterOption customises NewRateLimiter.
type RateLimiterOption func(*RateLimiter)

// WithKeyFunc chooses how clients are identified (default: client IP).
// Use it for API keys or user IDs. Only trust forwarded-IP headers such as
// X-Forwarded-For when a proxy you control sets them; clients can forge them.
func WithKeyFunc(f func(*http.Request) string) RateLimiterOption {
	return func(rl *RateLimiter) {
		if f != nil {
			rl.keyFunc = f
		}
	}
}

// WithClock replaces time.Now. Intended for tests.
func WithClock(now func() time.Time) RateLimiterOption {
	return func(rl *RateLimiter) {
		if now != nil {
			rl.now = now
		}
	}
}

// NewRateLimiter allows burst requests at once per client, refilling at limit
// requests per second. A burst below 1 is treated as 1.
func NewRateLimiter(limit rate.Limit, burst int, opts ...RateLimiterOption) *RateLimiter {
	if burst < 1 {
		burst = 1
	}
	rl := &RateLimiter{
		limit:   limit,
		burst:   burst,
		ttl:     idleTTL(limit, burst),
		keyFunc: clientIP,
		now:     time.Now,
		clients: make(map[string]*clientEntry),
	}
	for _, opt := range opts {
		opt(rl)
	}
	return rl
}

// idleTTL is how long a client may be silent before its entry is dropped:
// the time a drained bucket needs to refill, with sane lower and upper bounds.
func idleTTL(limit rate.Limit, burst int) time.Duration {
	switch {
	case limit == rate.Inf:
		return minIdleTTL // nothing to remember, any entry is disposable
	case limit <= 0:
		return 0 // never refills, so forgetting a client would reset its quota
	}
	secs := float64(burst) / float64(limit)
	if secs >= maxIdleTTL.Seconds() { // check before converting: avoids overflow
		return maxIdleTTL
	}
	return max(time.Duration(secs*float64(time.Second)), minIdleTTL)
}

// Middleware rejects over-limit requests with 429 and a Retry-After header
// (whole seconds, rounded up) and passes everything else to next.
func (rl *RateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		allowed, retryAfter := rl.reserve(rl.keyFunc(r))
		if !allowed {
			secs := max(int(math.Ceil(retryAfter.Seconds())), 1)
			w.Header().Set("Retry-After", strconv.Itoa(secs))
			WriteError(w, http.StatusTooManyRequests, "rate limit exceeded, retry later")
			return
		}
		next.ServeHTTP(w, r)
	})
}

// reserve takes one token for key. When none is available it returns how long
// the client must wait, and gives the speculative token back so rejected
// requests do not push the client further into debt.
func (rl *RateLimiter) reserve(key string) (allowed bool, retryAfter time.Duration) {
	now := rl.now()

	rl.mu.Lock()
	defer rl.mu.Unlock()
	rl.sweep(now)

	c, ok := rl.clients[key]
	if !ok {
		c = &clientEntry{limiter: rate.NewLimiter(rl.limit, rl.burst)}
		rl.clients[key] = c
	}
	c.lastSeen = now

	res := c.limiter.ReserveN(now, 1)
	if !res.OK() { // cannot happen: burst >= 1 and we ask for one token
		return false, time.Second
	}
	if delay := res.DelayFrom(now); delay > 0 {
		res.CancelAt(now)
		return false, delay
	}
	return true, 0
}

// sweep drops clients idle for longer than ttl. Caller must hold rl.mu.
func (rl *RateLimiter) sweep(now time.Time) {
	if rl.ttl == 0 || now.Sub(rl.lastSweep) < sweepInterval {
		return
	}
	rl.lastSweep = now
	for key, c := range rl.clients {
		if now.Sub(c.lastSeen) > rl.ttl {
			delete(rl.clients, key)
		}
	}
}

// size reports how many clients are tracked (used by tests).
func (rl *RateLimiter) size() int {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	return len(rl.clients)
}

func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}
	addr, err := netip.ParseAddr(host)
	if err != nil {
		return host
	}
	addr = addr.Unmap().WithZone("")
	if addr.Is6() {
		if p, err := addr.Prefix(64); err == nil {
			return p.String()
		}
	}
	return addr.String()
}
