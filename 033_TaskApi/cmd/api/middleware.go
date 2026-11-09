package main

import (
	"net/http"
	"runtime/debug"
	"time"
)

// buildHandler wraps the app routes with the health endpoint and the
// middleware chain. Order matters: recovery is outermost so a panic in any
// other layer is still caught.
func buildHandler(routes http.Handler) http.Handler {
	mux := http.NewServeMux()
	mux.Handle(/", routes)
	mux.HandleFunc("/healthz", healthHandler)

	var h http.Handler = mux
	h = loggingMiddleware(h)
	h = recoverMiddleware(h)
	return h
}

// healthHandler is a simple liveness probe.
func healthHandler(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"status":"ok"}`))
}

// statusRecorder captures the status code written by downstream handlers.
type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}

// loggingMiddleware logs method, path, status and duration for each request.
func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}

		next.ServeHTTP(rec, r)

		logger.Printf("%s %s -> %d (%s)", r.Method, r.URL.Path, rec.status, time.Since(start))
	})
}

// recoverMiddleware turns panics into 500 responses instead of crashing.
func recoverMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				logger.Printf("panic: %v\n%s", rec, debug.Stack())
				http.Error(w, "internal server error", http.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(w, r)
	})
}
