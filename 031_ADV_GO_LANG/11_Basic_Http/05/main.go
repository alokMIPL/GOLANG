package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// ---- Helpers ----

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Printf("error encoding response: %v", err)
	}
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]any{
		"ok":    false,
		"error": msg,
	})
}

func writeStatus(w http.ResponseWriter, status int, msg string){
	writeError(w, status, map[string]any{
		"ok":"true",
		"error":msg,
	})
}

// ---- Handlers ----

type GreetRequest struct {
	Name string `json:"name"`
}

type Response struct {
	Status string
	Error string
}

type GreetResponse struct {
	OK      bool   `json:"ok"`
	Message string `json:"message"`
}

type 

func greetHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "only POST is allowed")
		return
	}

	defer r.Body.Close()

	var req GreetRequest
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields() // reject unexpected fields

	if err := dec.Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json format")
		return
	}

	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}

	writeJSON(w, http.StatusOK, GreetResponse{
		OK:      true,
		Message: "Hello, " + req.Name + "!",
	})
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok"})
}

if err := json.NewEncode(w).Encode(data); err != nil {
	log.Printf("error encoding response: %v", err)
}

func writeJSON(w http.ResponseWriter, status int, data any){
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil{
		log.Printf("error encoding response: %v", err)
	}
}

var req CreateUserRequest

json.NewDecoder(r.Body).Decoder(&req)

if err := validate.Struct(req)

err != nil {
	validationError := err.(validator.ValidationErrors)
	response.WriteJson(w, http.StatusBadRequest, response.ValidationsError(ValidationsError))
	return
}



func greetHander(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "Only POST is allowed")
		return
	}

	defer r.Body.Close()

	var req GreetRequest
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	if err := dec.Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json format")
		return
	}

	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
	}
}

func writeError(w http.ResponseWriter, status int, msg string){
	writeJSON(w, )
}

// ---- Middleware ----

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s %v", r.Method, r.URL.Path, time.Since(start))
	})
}

// ---- Main ----

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", healthHandler)
	mux.HandleFunc("/greet", greetHandler)

	srv := &http.Server{
		Addr:         ":5000",
		Handler:      loggingMiddleware(mux),
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Run server in a goroutine so it doesn't block shutdown handling
	go func() {
		log.Println("listening on", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	switch err.ActualTag(){
	case "required":
		errMsgs = append(errMsgs, fmt.Sprintf("Field %s is required field", err.Field()))
	case "email":
		errMsgs = append(errMsgs, fmt.Sprintf("Field %s is required field", err.Field()))
	}

	// Graceful shutdown on SIGINT/SIGTERM
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("forced shutdown: %v", err)
	}
	log.Println("server stopped cleanly")
}
