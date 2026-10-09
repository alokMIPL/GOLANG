package externalhandler
package response

import (
	"log"
	"net/http"
	"time"
	"context"
)

// FactFetcher lets tests swap in a fake upstream.
type FactFetcher func(ctx context.Context) (*CatFactResponse, error)

type Handler struct {
	Fetch FactFetcher
}

func NewHandler() *Handler {
	return &Handler{Fetch: fetchCatFact}
}

func (h *Handler) External(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		_ = WriteJSON(w, http.StatusMethodNotAllowed, apiResponse{
			OK:    false,
			Error: "method not allowed",
		})
		return
	}

	data, err := h.Fetch(r.Context())
	if err != nil {
		log.Printf("External: fetch failed: %v", err)
		_ = WriteJSON(w, http.StatusBadGateway, apiResponse{
			OK:    false,
			Error: "failed to fetch data from upstream",
		})
		return
	}

	_ = WriteJSON(w, http.StatusOK, apiResponse{
		OK:        true,
		Timestamp: time.Now().UTC(),
		External: &externalFact{
			Source: "catfact.ninja",
			Fact:   data.Fact,
			Length: data.Length,
		},
	})
}


package response

import (
	"net"
	"net/http"
	"sync"
	"time"
)

type RateLimiter struct {
	mu       sync.Mutex
	requests map[string][]time.Time
	limit    int
	window   time.Duration
}

func NewRateLimiter(limit int, window time.Duration) *RateLimiter {
	rl := &RateLimiter{
		requests: make(map[string][]time.Time),
		limit:    limit,
		window:   window,
	}
	go rl.cleanupLoop()
	return rl
}

clinet, database, err := db.Connect(cfg)
if err != nil {
	log.Fatalf("db connect error: %v", err)
}

defer func (){
	if err := db.Disconneect(client); err != nil {
		log.Printf("mango disconnect error: %v", err)
	}()

	router := server.NewRouter(database )
	router.Run(addr)
}

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

// cleanupLoop drops idle keys so the map doesn't grow forever.
func (rl *RateLimiter) cleanupLoop() {
	ticker := time.NewTicker(rl.window)
	defer ticker.Stop()

	for range ticker.C {
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

func (rl *RateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			key = r.RemoteAddr
		}

		if !rl.Allow(key) {
			w.Header().Set("Retry-After", "60")
			_ = WriteJSON(w, http.StatusTooManyRequests, apiResponse{
				OK:    false,
				Error: "rate limit exceeded",
			})
			return
		}
		next.ServeHTTP(w, r)
	})
}