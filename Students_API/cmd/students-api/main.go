package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/alokMIPL/students-api/internal/config"
	"github.com/alokMIPL/students-api/internal/http/handlers/student"
)

func main() {

	fmt.Println("Welcome to students api")

	// Load config
	cfg := config.MustLoad()
	log.Println("Environment:", cfg.Env)

	// database setup

	// setup router
	router := http.NewServeMux()

	router.HandleFunc("POST /api/students", student.New())

	// setup server
	server := http.Server{
		Addr:    cfg.Addr,
		Handler: router,
	}

	slog.Info("Storage path", slog.String("path", cfg.StoragePath))

	slog.Info("Server started", slog.String("address", cfg.Addr))

	done := make(chan os.Signal, 1)

	signal.Notify(done, os.Interrupt, syscall.SIGTERM)

	go func() {
		err := server.ListenAndServe()
		// ErrServerClosed is returned on normal Shutdown, so it is not a failure
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatal("Failed to start server: ", err)
		}
	}()

	<-done

	// Now Gracefully ShutDown.
	slog.Info("Shutting down the server")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		slog.Error("Failed to shutdown server", slog.String("error", err.Error()))
	}

	slog.Info("Server shutdown successfully")

}
