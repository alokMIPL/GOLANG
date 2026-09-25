package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/gorilla/websocket"
)

// ============ JWT AUTH ============

var jwtSecret = []byte("change-this-to-a-real-secret-from-env")

type Claims struct {
	UserID int    `json:"user_id"`
	Name   string `json:"name"`
	jwt.RegisteredClaims
}

func generateToken(userID int, name string) (string, error) {
	claims := Claims{
		UserID: userID,
		Name:   name,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtSecret)
}

func parseToken(tokenStr string) (*Claims, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (any, error) {
		return jwtSecret, nil
	})
	if err != nil || !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}
	return claims, nil
}

// context key for storing claims
type ctxKey string

const claimsKey ctxKey = "claims"

func authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if !strings.HasPrefix(authHeader, "Bearer ") {
			writeError(w, http.StatusUnauthorized, "missing or invalid authorization header")
			return
		}

		tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
		claims, err := parseToken(tokenStr)
		if err != nil {
			writeError(w, http.StatusUnauthorized, "invalid or expired token")
			return
		}

		ctx := context.WithValue(r.Context(), claimsKey, claims)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func getClaims(r *http.Request) *Claims {
	claims, _ := r.Context().Value(claimsKey).(*Claims)
	return claims
}

// ---- Login handler (issues a token) ----

func loginHandler(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}

	// In real code: verify credentials against a database here
	token, err := generateToken(1, body.Name)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not generate token")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "token": token})
}

// ---- Protected handler (requires valid JWT) ----

func profileHandler(w http.ResponseWriter, r *http.Request) {
	claims := getClaims(r)
	writeJSON(w, http.StatusOK, map[string]any{
		"user_id": claims.UserID,
		"name":    claims.Name,
	})
}

// ============ RATE LIMITING ============

// Simple token-bucket rate limiter per client IP
type RateLimiter struct {
	mu       sync.Mutex
	visitors map[string]*bucket
	rate     int           // tokens added per interval
	interval time.Duration // refill interval
	burst    int           // max tokens
}

type bucket struct {
	tokens   int
	lastSeen time.Time
}

func NewRateLimiter(rate int, interval time.Duration, burst int) *RateLimiter {
	rl := &RateLimiter{
		visitors: make(map[string]*bucket),
		rate:     rate,
		interval: interval,
		burst:    burst,
	}
	go rl.cleanup()
	return rl
}

func (rl *RateLimiter) Allow(ip string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	b, exists := rl.visitors[ip]
	if !exists {
		rl.visitors[ip] = &bucket{tokens: rl.burst - 1, lastSeen: time.Now()}
		return true
	}

	elapsed := time.Since(b.lastSeen)
	refill := int(elapsed/rl.interval) * rl.rate
	if refill > 0 {
		b.tokens += refill
		if b.tokens > rl.burst {
			b.tokens = rl.burst
		}
		b.lastSeen = time.Now()
	}

	if b.tokens <= 0 {
		return false
	}
	b.tokens--
	return true
}

// periodically evict stale entries so the map doesn't grow forever
func (rl *RateLimiter) cleanup() {
	for {
		time.Sleep(10 * time.Minute)
		rl.mu.Lock()
		for ip, b := range rl.visitors {
			if time.Since(b.lastSeen) > 30*time.Minute {
				delete(rl.visitors, ip)
			}
		}
		rl.mu.Unlock()
	}
}

func rateLimitMiddleware(rl *RateLimiter) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := r.RemoteAddr // in production, parse X-Forwarded-For behind a proxy
			if !rl.Allow(ip) {
				writeError(w, http.StatusTooManyRequests, "rate limit exceeded")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// ============ WEBSOCKETS ============

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     func(r *http.Request) bool { return true }, // tighten in production
}

func wsHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("upgrade error:", err)
		return
	}
	defer conn.Close()

	log.Println("client connected:", conn.RemoteAddr())

	for {
		msgType, msg, err := conn.ReadMessage()
		if err != nil {
			log.Println("client disconnected:", err)
			break
		}
		log.Printf("received: %s", msg)

		// echo it back
		if err := conn.WriteMessage(msgType, msg); err != nil {
			log.Println("write error:", err)
			break
		}
	}
}

// ============ Shared helpers (from before) ============

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]any{"ok": false, "error": msg})
}

type Middleware func(http.Handler) http.Handler

func Chain(h http.Handler, mws ...Middleware) http.Handler {
	for i := len(mws) - 1; i >= 0; i-- {
		h = mws[i](h)
	}
	return h
}

// ============ MAIN ============

func main() {
	rl := NewRateLimiter(5, time.Second, 10) // 5 tokens/sec, burst of 10

	mux := http.NewServeMux()
	mux.HandleFunc("/login", loginHandler)
	mux.HandleFunc("/ws", wsHandler)

	// protected routes need the auth middleware
	protected := http.NewServeMux()
	protected.HandleFunc("/profile", profileHandler)
	mux.Handle("/profile", authMiddleware(protected))

	handler := Chain(mux, rateLimitMiddleware(rl))

	log.Println("listening on :5000")
	log.Fatal(http.ListenAndServe(":5000", handler))
}
