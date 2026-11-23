package main

import (
	"bufio"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net"
	"net/http"
	"runtime/debug"
	"strconv"
	"sync"
	"time"
)

// Tunables for the middleware chain.
const (
	maxRequestBodyBytes = 1 << 20          // 1 MiB
	requestTimeout      = 15 * time.Second // per-request handler deadline
	rateLimitPerSecond  = 20.0             // sustained requests per second per client
	rateLimitBurst      = 40.0             // short bursts allowed above the sustained rate
	rateLimiterIdleTTL  = 5 * time.Minute  // forget clients that have gone quiet
	maxRequestIDLength  = 64
	requestIDHeader     = "X-Request-ID"
)

// middleware wraps an http.Handler with extra behavior.
type middleware func(http.Handler) http.Handler

// chain applies middlewares so the first one listed is the outermost.
// chain(h, a, b, c) behaves like a(b(c(h))).
func chain(h http.Handler, mws ...middleware) http.Handler {
	for i := len(mws) - 1; i >= 0; i-- {
		h = mws[i](h)
	}
	return h
}

func buildHandler(routes http.Handler) http.Handler {
	mux := http.NewServeMux()
	mux.Handle("/", routes)
	mux.HandleFunc("/healthz", healthHandler)

	limiter := newRateLimiter(rateLimitPerSecond, rateLimitBurst, rateLimiterIdleTTL)

	return chain(mux,
		recoverMiddleware,         // outermost: catches panics from everything below
		requestIDMiddleware,       // tag the request before anything logs it
		loggingMiddleware,         // log after the response is written
		securityHeadersMiddleware, // add defensive headers to every response
		rateLimitMiddleware(limiter),

		bodyLimitMiddleware(maxRequestBodyBytes),
		timeoutMiddleware(requestTimeout),
	)
}

func healthHandler(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"status":"ok"}`))
}

type statusRecorder struct {
	http.ResponseWriter
	status      int
	bytes       int
	wroteHeader bool
}

func (r *statusRecorder) WriteHeader(code int) {
	if r.wroteHeader {
		return
	}
	r.wroteHeader = true
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}

func (r *statusRecorder) Write(b []byte) (int, error) {
	if !r.wroteHeader {
		r.WriteHeader(http.StatusOK)
	}
	n, err := r.ResponseWriter.Write(b)
	r.bytes += n
	return n, err
}

func (r *statusRecorder) Flush() {
	if f, ok := r.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

func (r *statusRecorder) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	h, ok := r.ResponseWriter.(http.Hijacker)
	if !ok {
		return nil, nil, errors.New("underlying ResponseWriter does not support hijacking")
	}
	return h.Hijack()
}

func (r *statusRecorder) Unwrap() http.ResponseWriter {
	return r.ResponseWriter
}

type ctxKey int

const requestIDKey ctxKey = iota

func requestIDFrom(ctx context.Context) string {
	id, _ := ctx.Value(requestIDKey).(string)
	return id
}

func newRequestID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return strconv.FormatInt(time.Now().UnixNano(), 16)
	}
	return hex.EncodeToString(b[:])
}

func validRequestID(id string) bool {
	if id == "" || len(id) > maxRequestIDLength {
		return false
	}
	for _, c := range id {
		switch {
		case c >= 'a' && c <= 'z':
		case c >= 'A' && c <= 'Z':
		case c >= '0' && c <= '9':
		case c == '-', c == '_', c == '.':
		default:
			return false
		}
	}
	return true
}

func requestIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get(requestIDHeader)
		if !validRequestID(id) {
			id = newRequestID()
		}

		w.Header().Set(requestIDHeader, id)
		ctx := context.WithValue(r.Context(), requestIDKey, id)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}

		next.ServeHTTP(rec, r)

		logger.Printf("id=%s %s %s -> %d %dB (%s) from=%s",
			requestIDFrom(r.Context()),
			r.Method,
			r.URL.Path,
			rec.status,
			rec.bytes,
			time.Since(start).Round(time.Microsecond),
			clientIP(r),
		)
	})
}

func recoverMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			rec := recover()
			if rec == nil {
				return
			}
			if rec == http.ErrAbortHandler {
				panic(rec)
			}

			logger.Printf("id=%s panic: %v\n%s", requestIDFrom(r.Context()), rec, debug.Stack())
			writeJSONError(w, http.StatusInternalServerError, "internal server error")
		}()
		next.ServeHTTP(w, r)
	})
}

func writeJSONError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(map[string]string{"error": msg}); err != nil {
		logger.Printf("failed to write error response: %v", err)
	}
}

func securityHeadersMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("X-Frame-Options", "DENY")
		h.Set("Referrer-Policy", "no-referrer")
		h.Set("Content-Security-Policy", "default-src 'none'; frame-ancestors 'none'")
		h.Set("Cache-Control", "no-store")
		next.ServeHTTP(w, r)
	})
}

func bodyLimitMiddleware(limit int64) middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.ContentLength > limit {
				writeJSONError(w, http.StatusRequestEntityTooLarge,
					fmt.Sprintf("request body exceeds %d bytes", limit))
				return
			}
			if r.Body != nil {
				r.Body = http.MaxBytesReader(w, r.Body, limit)
			}
			next.ServeHTTP(w, r)
		})
	}
}

func timeoutMiddleware(d time.Duration) middleware {
	return func(next http.Handler) http.Handler {
		return http.TimeoutHandler(next, d, `{"error":"request timed out"}`)
	}
}

type visitor struct {
	tokens float64
	last   time.Time
}

type rateLimiter struct {
	mu        sync.Mutex
	rate      float64 // tokens added per second
	burst     float64 // bucket capacity
	ttl       time.Duration
	visitors  map[string]*visitor
	lastSweep time.Time
}

func newRateLimiter(rate, burst float64, ttl time.Duration) *rateLimiter {
	return &rateLimiter{
		rate:      rate,
		burst:     burst,
		ttl:       ttl,
		visitors:  make(map[string]*visitor),
		lastSweep: time.Now(),
	}
}

func (rl *rateLimiter) allow(key string, now time.Time) (bool, time.Duration) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	rl.sweepLocked(now)

	v, ok := rl.visitors[key]
	if !ok {
		v = &visitor{tokens: rl.burst, last: now}
		rl.visitors[key] = v
	}

	elapsed := now.Sub(v.last).Seconds()
	v.tokens = math.Min(rl.burst, v.tokens+elapsed*rl.rate)
	v.last = now

	if v.tokens >= 1 {
		v.tokens--
		return true, 0
	}

	wait := time.Duration((1 - v.tokens) / rl.rate * float64(time.Second))
	return false, wait
}

func (rl *rateLimiter) sweepLocked(now time.Time) {
	if now.Sub(rl.lastSweep) < rl.ttl {
		return
	}
	for key, v := range rl.visitors {
		if now.Sub(v.last) > rl.ttl {
			delete(rl.visitors, key)
		}
	}
	rl.lastSweep = now
}

func rateLimitMiddleware(rl *rateLimiter) middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/healthz" {
				next.ServeHTTP(w, r)
				return
			}

			ok, wait := rl.allow(clientIP(r), time.Now())
			if !ok {
				secs := int(math.Ceil(wait.Seconds()))
				if secs < 1 {
					secs = 1
				}
				w.Header().Set("Retry-After", strconv.Itoa(secs))
				writeJSONError(w, http.StatusTooManyRequests, "rate limit exceeded")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
