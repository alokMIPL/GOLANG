package response

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-playground/validator/v10"
)

const defaultTimeout = 5 * time.Second

var validate = validator.New()

type config struct {
	fetch   Fetcher
	timeout time.Duration
	logger  *slog.Logger
}

// Option customises NewHandler.
type Option func(*config)

// WithFetcher replaces the upstream source. A nil Fetcher is ignored.
func WithFetcher(f Fetcher) Option {
	return func(c *config) {
		if f != nil {
			c.fetch = f
		}
	}
}

// WithTimeout bounds how long one request may wait for upstream.
func WithTimeout(d time.Duration) Option {
	return func(c *config) {
		if d > 0 {
			c.timeout = d
		}
	}
}

// WithLogger sets the logger used for upstream problems. A nil logger is ignored.
func WithLogger(l *slog.Logger) Option {
	return func(c *config) {
		if l != nil {
			c.logger = l
		}
	}
}

type handler struct{ cfg config }

// NewHandler returns an http.Handler that serves one cat fact per GET.
//
// Status codes:
//
//	200 fact returned
//	405 method other than GET (with an Allow header)
//	502 upstream failed or returned data that failed validation
//	504 upstream did not answer within the timeout
//
// Internal error details are logged, never sent to the client.
func NewHandler(opts ...Option) http.Handler {
	cfg := config{
		fetch:   NewHTTPFetcher(nil, DefaultCatFactURL),
		timeout: defaultTimeout,
		logger:  slog.Default(),
	}
	for _, opt := range opts {
		opt(&cfg)
	}
	return handler{cfg: cfg}
}

func (h handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), h.cfg.timeout)
	defer cancel()

	fact, err := h.cfg.fetch(ctx)
	if err != nil {
		h.fetchFailed(w, r, err)
		return
	}
	if fact == nil {
		h.cfg.logger.Error("fetcher returned nil result without an error")
		WriteError(w, http.StatusBadGateway, "upstream returned invalid data")
		return
	}

	if err := validate.Struct(fact); err != nil {
		var ve validator.ValidationErrors
		if errors.As(err, &ve) {
			// The client did nothing wrong, so details go to the log only.
			h.cfg.logger.Warn("upstream payload failed validation", "problems", ValidationError(ve).Error)
		} else {
			h.cfg.logger.Error("validate upstream payload", "error", err)
		}
		WriteError(w, http.StatusBadGateway, "upstream returned invalid data")
		return
	}

	writeJSON(w, http.StatusOK, apiResponse{OK: true, External: fact})
}
func (h handler) fetchFailed(w http.ResponseWriter, r *http.Request, err error) {
	// The caller hung up (or its own deadline passed): nobody is listening.
	if r.Context().Err() != nil {
		h.cfg.logger.Debug("client went away during upstream fetch", "error", err)
		return
	}

	status, msg := http.StatusBadGateway, "upstream service unavailable"
	if errors.Is(err, context.DeadlineExceeded) {
		status, msg = http.StatusGatewayTimeout, "upstream service timed out"
	}
	h.cfg.logger.Warn("upstream fetch failed", "error", err, "status", status)
	WriteError(w, status, msg)
}
