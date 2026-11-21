// Command notesd serves the notes API.
//
// Configuration (environment variables):
//
//	NOTESD_ADDR       listen address            (default ":8080")
//	NOTESD_MONGO_URI  MongoDB connection string (default "mongodb://localhost:27017")
//	NOTESD_DB         database name             (default "notesapp")
//
// Authentication here is the development-only X-User-ID header. Replace
// notes.HeaderUserAuth with real credential verification before deploying.
package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"example.com/notesapp/notes"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type config struct {
	addr     string
	mongoURI string
	dbName   string
}

func loadConfig() config {
	return config{
		addr:     getenv("NOTESD_ADDR", ":8080"),
		mongoURI: getenv("NOTESD_MONGO_URI", "mongodb://localhost:27017"),
		dbName:   getenv("NOTESD_DB", "notesapp"),
	}
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func main() {
	if err := run(); err != nil {
		slog.Error("notesd exited with error", "err", err)
		os.Exit(1)
	}
}

func run() error {
	cfg := loadConfig()
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(cfg.mongoURI))
	if err != nil {
		return err
	}
	defer func() {
		dctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := client.Disconnect(dctx); err != nil {
			log.Error("mongo disconnect failed", "err", err)
		}
	}()

	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := client.Ping(pingCtx, nil); err != nil {
		return err
	}

	repo, err := notes.NewMongoRepository(ctx, client.Database(cfg.dbName))
	if err != nil {
		return err
	}
	svc := notes.NewService(repo)
	handler := notes.NewHandler(svc, log)

	api := notes.Chain(handler.Routes(),
		notes.RequestLogger(log),
		notes.Recoverer(log),
		notes.HeaderUserAuth("X-User-ID"),
	)
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		hctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()
		if err := client.Ping(hctx, nil); err != nil {
			http.Error(w, "database unavailable", http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
	mux.Handle("/", api)

	srv := &http.Server{
		Addr:              cfg.addr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		log.Info("listening", "addr", cfg.addr, "db", cfg.dbName)
		errCh <- srv.ListenAndServe()
	}()

	select {
	case err := <-errCh:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	case <-ctx.Done():
		log.Info("shutting down")
	}

	shutdownCtx, cancelShutdown := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelShutdown()
	return srv.Shutdown(shutdownCtx)
}
