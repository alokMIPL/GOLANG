package main

import (
	"context"
	"crypto/rand"
	"crypto/tls"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"runtime/debug"
	"sync/atomic"
	"time"

	"github.com/example/taskapi/internal/config"
)

const (
	// readHeaderTimeout guards against slowloris-style clients regardless of
	// the configurable read/write timeouts.
	readHeaderTimeout = 5 * time.Second

	// maxHeaderBytes caps request header size (the stdlib default is 1 MiB;
	// we make it explicit so it is visible and easy to tune).
	maxHeaderBytes = 1 << 20

	// defaultShutdownTimeout is how long in-flight requests get to finish
	// after the shutdown signal before connections are forcibly closed.
	defaultShutdownTimeout = 15 * time.Second

	// requestIDHeader is read from incoming requests and echoed on responses.
	requestIDHeader = "X-Request-ID"
)

type serverOptions struct {
	TLSCertFile   string
	TLSKeyFile    string
	ShutdownDelay time.Duration

	ShutdownTimeout time.Duration
}

func (o serverOptions) tlsEnabled() bool {
	return o.TLSCertFile != "" && o.TLSKeyFile != ""
}

func (o serverOptions) shutdownTimeout() time.Duration {
	if o.ShutdownTimeout > 0 {
		return o.ShutdownTimeout
	}
	return defaultShutdownTimeout
}

// ---------------------------------------------------------------------------
// Middleware
// ---------------------------------------------------------------------------

type middleware func(http.Handler) http.Handler

// chain wraps h so that the first middleware listed is the outermost one.
func chain(h http.Handler, mws ...middleware) http.Handler {
	for i := len(mws) - 1; i >= 0; i-- {
		h = mws[i](h)
	}
	return h
}

type ctxKey int

const requestIDKey ctxKey = iota

// requestIDFrom returns the request ID stored in ctx, or "" if there is none.
func requestIDFrom(ctx context.Context) string {
	id, _ := ctx.Value(requestIDKey).(string)
	return id
}

func newRequestID() string {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "unknown"
	}
	return hex.EncodeToString(b[:])
}

// requestIDMiddleware propagates an incoming request ID or generates one, then
// stores it in the request context and the response headers.
func requestIDMiddleware() middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			id := r.Header.Get(requestIDHeader)
			if id == "" {
				id = newRequestID()
			}
			w.Header().Set(requestIDHeader, id)
			ctx := context.WithValue(r.Context(), requestIDKey, id)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// statusRecorder captures the status code and body size of a response.
type statusRecorder struct {
	http.ResponseWriter
	status int
	bytes  int64
	wrote  bool
}

func (r *statusRecorder) WriteHeader(code int) {
	if !r.wrote {
		r.status = code
		r.wrote = true
	}
	r.ResponseWriter.WriteHeader(code)
}

func (r *statusRecorder) Write(b []byte) (int, error) {
	if !r.wrote {
		r.status = http.StatusOK
		r.wrote = tr
	}
	n, err := r.ResponseWriter.Write(b)
	r.bytes += int64(n)
	return n, err
}

func (r *statusRecorder) Unwrap() http.ResponseWriter { return r.ResponseWriter }

// accessLogMiddleware logs one line per request after it completes.
func accessLogMiddleware(log *slog.Logger) middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}

			next.ServeHTTP(rec, r)

			log.Info("request",
				"request_id", requestIDFrom(r.Context()),
				"method", r.Method,
				"path", r.URL.Path,
				"status", rec.status,
				"bytes", rec.bytes,
				"duration", time.Since(start).String(),
				"remote", r.RemoteAddr,
			)
		})
	}
}

func recoverMiddleware(log *slog.Logger) middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				p := recover()
				if p == nil {
					return
				}
				if p == http.ErrAbortHandler {
					panic(p)
				}
				log.Error("panic in handler",
					"request_id", requestIDFrom(r.Context()),
					"panic", fmt.Sprint(p),
					"stack", string(debug.Stack()),
				)
				http.Error(w, http.StatusText(http.StatusInternalServerError),
					http.StatusInternalServerError)
			}()
			next.ServeHTTP(w, r)
		})
	}
}

type health struct {
	ready atomic.Bool
}

func (h *health) livez(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok\n"))
}

func (h *health) readyz(w http.ResponseWriter, _ *http.Request) {
	if !h.ready.Load() {
		http.Error(w, "shutting down", http.StatusServiceUnavailable)
		return
	}
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ready\n"))
}

// buildHandler mounts health endpoints next to the application handler and
// wraps everything in the standard middleware stack.
func buildHandler(app http.Handler, h *health, log *slog.Logger) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", h.livez)
	mux.HandleFunc("GET /readyz", h.readyz)
	mux.Handle("/", app)

	return chain(mux,
		requestIDMiddleware(),
		accessLogMiddleware(log),
		recoverMiddleware(log),
	)
}

// ---------------------------------------------------------------------------
// Server construction and lifecycle
// ---------------------------------------------------------------------------

// newServer builds an http.Server with all timeouts set, routes the server's
// internal error log through slog, and logs connection state changes at debug
// level.
func newServer(cfg *config.Config, handler http.Handler, log *slog.Logger) *http.Server {
	return &http.Server{
		Addr:              cfg.Addr,
		Handler:           handler,
		ReadHeaderTimeout: readHeaderTimeout,
		ReadTimeout:       cfg.ReadTimeout,
		WriteTimeout:      cfg.WriteTimeout,
		IdleTimeout:       cfg.IdleTimeout,
		MaxHeaderBytes:    maxHeaderBytes,
		ErrorLog:          slog.NewLogLogger(log.Handler(), slog.LevelError),
		ConnState: func(c net.Conn, state http.ConnState) {
			log.Debug("connection state changed",
				"remote", c.RemoteAddr().String(),
				"state", state.String(),
			)
		},
	}
}

func run(ctx context.Context, cfg *config.Config, app http.Handler, log *slog.Logger, opts serverOptions) error {
	h := &health{}
	srv := newServer(cfg, buildHandler(app, h, log), log)

	baseCtx := context.WithoutCancel(ctx)
	srv.BaseContext = func(net.Listener) context.Context { return baseCtx }

	if opts.tlsEnabled() {
		srv.TLSConfig = &tls.Config{MinVersion: tls.VersionTLS12}
	}

	ln, err := net.Listen("tcp", srv.Addr)
	if err != nil {
		return fmt.Errorf("listen on %q: %w", srv.Addr, err)
	}

	serveErr := make(chan error, 1)
	go func() {
		log.Info("server listening",
			"addr", ln.Addr().String(),
			"tls", opts.tlsEnabled(),
		)
		if opts.tlsEnabled() {
			serveErr <- srv.ServeTLS(ln, opts.TLSCertFile, opts.TLSKeyFile)
			return
		}
		serveErr <- srv.Serve(ln)
	}()

	h.ready.Store(true)

	select {
	case err := <-serveErr:
		// Serve only returns on its own if something went wrong.
		return fmt.Errorf("serve: %w", err)
	case <-ctx.Done():
		log.Info("shutdown signal received",
			"drain_delay", opts.ShutdownDelay.String(),
			"timeout", opts.shutdownTimeout().String(),
		)
	}

	h.ready.Store(false)

	if opts.ShutdownDelay > 0 {
		select {
		case <-time.After(opts.ShutdownDelay):
		case err := <-serveErr:
			return fmt.Errorf("serve during drain: %w", err)
		}
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), opts.shutdownTimeout())
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		// Graceful shutdown timed out; drop remaining connections.
		log.Error("graceful shutdown failed, forcing close", "err", err)
		_ = srv.Close()
		return fmt.Errorf("shutdown: %w", err)
	}

	if err := <-serveErr; !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("serve: %w", err)
	}

	log.Info("server stopped cleanly")
	return nil
}
