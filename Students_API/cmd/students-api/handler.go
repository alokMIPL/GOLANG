package db

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"notes_api/internal/config"

	"go.mongodb.org/mongo-driver/mongo"
)

// ---------------------------------------------------------------------------
// Pure unit tests (no MongoDB needed)
// ---------------------------------------------------------------------------

func TestValidate(t *testing.T) {
	tests := []struct {
		name string
		cfg  config.Config
		want error
	}{
		{"empty uri", config.Config{MongoDB: "notes"}, ErrEmptyURI},
		{"empty database", config.Config{MongoURI: "mongodb://localhost:27017"}, ErrEmptyDatabase},
		{"both set", config.Config{MongoURI: "mongodb://localhost:27017", MongoDB: "notes"}, nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := validate(tt.cfg); !errors.Is(got, tt.want) {
				t.Fatalf("validate() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBackoffFor(t *testing.T) {
	base := 500 * time.Millisecond

	tests := []struct {
		attempt int
		want    time.Duration
	}{
		{1, 500 * time.Millisecond},
		{2, 1 * time.Second},
		{3, 2 * time.Second},
		{4, 4 * time.Second},
		{5, maxRetryBackoff}, // 8s would exceed the cap
		{10, maxRetryBackoff},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("attempt_%d", tt.attempt), func(t *testing.T) {
			if got := backoffFor(base, tt.attempt); got != tt.want {
				t.Fatalf("backoffFor(%v, %d) = %v, want %v", base, tt.attempt, got, tt.want)
			}
		})
	}
}

func TestOptionsApplied(t *testing.T) {
	s := defaultSettings()
	for _, opt := range []Option{
		WithConnectTimeout(2 * time.Second),
		WithPoolSize(2, 20),
		WithRetry(5, 100*time.Millisecond),
		WithAppName("tests"),
	} {
		opt(&s)
	}

	if s.connectTimeout != 2*time.Second {
		t.Errorf("connectTimeout = %v", s.connectTimeout)
	}
	if s.minPoolSize != 2 || s.maxPoolSize != 20 {
		t.Errorf("pool = %d..%d", s.minPoolSize, s.maxPoolSize)
	}
	if s.maxAttempts != 5 || s.backoff != 100*time.Millisecond {
		t.Errorf("retry = %d / %v", s.maxAttempts, s.backoff)
	}
	if s.appName != "tests" {
		t.Errorf("appName = %q", s.appName)
	}
}

func TestOptionsIgnoreInvalidValues(t *testing.T) {
	s := defaultSettings()
	want := s

	WithConnectTimeout(0)(&s)
	WithConnectTimeout(-time.Second)(&s)
	WithRetry(0, 0)(&s)
	WithAppName("")(&s)
	WithLogger(nil)(&s)

	if s.connectTimeout != want.connectTimeout ||
		s.maxAttempts != want.maxAttempts ||
		s.backoff != want.backoff ||
		s.appName != want.appName {
		t.Fatalf("invalid option values changed the settings: %+v", s)
	}
}

func TestErrorHelpers(t *testing.T) {
	if !IsNotFound(fmt.Errorf("wrapped: %w", mongo.ErrNoDocuments)) {
		t.Error("IsNotFound should see through wrapping")
	}
	if IsNotFound(errors.New("other")) {
		t.Error("IsNotFound matched an unrelated error")
	}
	if !IsTimeout(fmt.Errorf("wrapped: %w", context.DeadlineExceeded)) {
		t.Error("IsTimeout should match context.DeadlineExceeded")
	}
	if IsDuplicateKey(errors.New("other")) {
		t.Error("IsDuplicateKey matched an unrelated error")
	}
}

func TestNewRejectsBadConfig(t *testing.T) {
	ctx := context.Background()

	t.Run("empty uri", func(t *testing.T) {
		_, err := New(ctx, config.Config{MongoDB: "x"})
		if !errors.Is(err, ErrEmptyURI) {
			t.Fatalf("err = %v, want ErrEmptyURI", err)
		}
	})
	t.Run("empty database", func(t *testing.T) {
		_, err := New(ctx, config.Config{MongoURI: "mongodb://localhost"})
		if !errors.Is(err, ErrEmptyDatabase) {
			t.Fatalf("err = %v, want ErrEmptyDatabase", err)
		}
	})

	t.Run("malformed uri", func(t *testing.T) {
		_, err := New(ctx, config.Config{MongoURI: "not-a-uri", MongoDB: "x"})
		if !errors.Is(err, ErrInvalidURI) {
			t.Fatalf("err = %v, want ErrInvalidURI", err)
		}
	})
}

func TestCloseNilStoreIsSafe(t *testing.T) {
	var st *Store
	if err := st.Close(); err != nil {
		t.Fatalf("Close on nil store: %v", err)
	}
	if err := Disconnect(nil); err != nil {
		t.Fatalf("Disconnect(nil): %v", err)
	}
}

// TestNewUnreachableServer checks that retries run out and the final error
// still wraps one of the package sentinels.
func TestNewUnreachableServer(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping slow network test in -short mode")
	}

	cfg := config.Config{
		MongoURI: "mongodb://127.0.0.1:1", // nothing listens on port 1
		MongoDB:  "unreachable",
	}

	start := time.Now()
	_, err := New(context.Background(), cfg,
		WithServerSelectionTimeout(200*time.Millisecond),
		WithConnectTimeout(500*time.Millisecond),
		WithRetry(2, 10*time.Millisecond),
	)
	if err == nil {
		t.Fatal("expected an error, got nil")
	}
	if !errors.Is(err, ErrPing) && !errors.Is(err, ErrConnect) {
		t.Fatalf("err = %v, want ErrPing or ErrConnect", err)
	}
	if elapsed := time.Since(start); elapsed > 5*time.Second {
		t.Fatalf("took too long (%v); timeouts are not being honoured", elapsed)
	}
}

func TestNewHonoursCancelledContext(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping slow network test in -short mode")
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := New(ctx, config.Config{MongoURI: "mongodb://127.0.0.1:1", MongoDB: "x"},
		WithServerSelectionTimeout(200*time.Millisecond),
		WithRetry(3, time.Second),
	)
	if err == nil {
		t.Fatal("expected an error with a cancelled context")
	}
}

// ---------------------------------------------------------------------------
// Integration tests (need a real MongoDB)
//
//	MONGO_TEST_URI=mongodb://localhost:27017 go test ./internal/db -run Integration
// ---------------------------------------------------------------------------

func integrationConfig(t *testing.T) config.Config {
	t.Helper()
	uri := os.Getenv("MONGO_TEST_URI")
	if uri == "" {
		t.Skip("MONGO_TEST_URI not set; skipping integration test")
	}
	return config.Config{
		MongoURI: uri,
		// Unique name so parallel runs never collide.
		MongoDB: fmt.Sprintf("notes_test_%d", time.Now().UnixNano()),
	}
}

func TestIntegrationConnectPingClose(t *testing.T) {
	cfg := integrationConfig(t)
	ctx := context.Background()

	st, err := New(ctx, cfg)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	t.Cleanup(func() {
		_ = st.Database().Drop(context.Background())
		_ = st.Close()
	})

	if err := st.Ping(ctx); err != nil {
		t.Fatalf("Ping: %v", err)
	}

	h := st.Health(ctx)
	if !h.OK || h.Latency <= 0 {
		t.Fatalf("Health = %+v", h)
	}

	// Close is idempotent and later calls report ErrClosed.
	if err := st.Close(); err != nil {
		t.Fatalf("first Close: %v", err)
	}
	if err := st.Close(); err != nil {
		t.Fatalf("second Close: %v", err)
	}
	if err := st.Ping(ctx); !errors.Is(err, ErrClosed) {
		t.Fatalf("Ping after Close = %v, want ErrClosed", err)
	}
}

func TestIntegrationEnsureIndexes(t *testing.T) {
	cfg := integrationConfig(t)
	ctx := context.Background()

	st, err := New(ctx, cfg)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	t.Cleanup(func() {
		_ = st.Database().Drop(context.Background())
		_ = st.Close()
	})

	// Running it twice proves it is safe at every startup.
	for i := 0; i < 2; i++ {
		if err := st.EnsureIndexes(ctx, NotesIndexes()); err != nil {
			t.Fatalf("EnsureIndexes (run %d): %v", i+1, err)
		}
	}

	cur, err := st.Collection("notes").Indexes().List(ctx)
	if err != nil {
		t.Fatalf("list indexes: %v", err)
	}
	var got []map[string]any
	if err := cur.All(ctx, &got); err != nil {
		t.Fatalf("read indexes: %v", err)
	}
	// _id_ + the two from NotesIndexes.
	if len(got) != 3 {
		t.Fatalf("got %d indexes, want 3: %v", len(got), got)
	}
}
