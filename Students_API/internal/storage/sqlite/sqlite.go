package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	// Registers the "sqlite3" driver with database/sql.
	_ "github.com/mattn/go-sqlite3"

	"github.com/alokMIPL/students-api/internal/config"
)

const (
	driverName  = "sqlite3"
	pingTimeout = 5 * time.Second
	initTimeout = 10 * time.Second
	closeWait   = 5 * time.Second
)

// ErrEmptyStoragePath is returned when the config has no database path.
var ErrEmptyStoragePath = errors.New("sqlite: storage path is empty")

// schema is applied on startup. IF NOT EXISTS makes it safe to run every time.
const schema = `
CREATE TABLE IF NOT EXISTS students (
	id    INTEGER PRIMARY KEY AUTOINCREMENT,
	name  TEXT    NOT NULL,
	email TEXT    NOT NULL UNIQUE,
	age   INTEGER NOT NULL CHECK (age >= 0)
);`

// Sqlite wraps a *sql.DB for the students API.
type Sqlite struct {
	Db *sql.DB
}

// New opens the SQLite database at cfg.StoragePath, verifies the connection,
// and makes sure the schema exists. The caller must call Close when done.
func New(cfg config.Config) (*Sqlite, error) {
	if cfg.StoragePath == "" {
		return nil, ErrEmptyStoragePath
	}

	db, err := sql.Open(driverName, cfg.StoragePath)
	if err != nil {
		return nil, fmt.Errorf("sqlite: open %q: %w", cfg.StoragePath, err)
	}

	// SQLite allows a single writer at a time; one connection avoids
	// "database is locked" errors for simple workloads.
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	db.SetConnMaxLifetime(0)

	// sql.Open does not actually connect, so ping to fail fast on a bad path.
	pingCtx, cancelPing := context.WithTimeout(context.Background(), pingTimeout)
	defer cancelPing()
	if err := db.PingContext(pingCtx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("sqlite: ping: %w", err)
	}

	s := &Sqlite{Db: db}

	initCtx, cancelInit := context.WithTimeout(context.Background(), initTimeout)
	defer cancelInit()
	if err := s.migrate(initCtx); err != nil {
		_ = db.Close()
		return nil, err
	}

	return s, nil
}

// migrate creates the tables this package needs.
func (s *Sqlite) migrate(ctx context.Context) error {
	if _, err := s.Db.ExecContext(ctx, schema); err != nil {
		return fmt.Errorf("sqlite: apply schema: %w", err)
	}
	return nil
}

// Health reports whether the database is reachable.
func (s *Sqlite) Health(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, pingTimeout)
	defer cancel()
	if err := s.Db.PingContext(ctx); err != nil {
		return fmt.Errorf("sqlite: health check: %w", err)
	}
	return nil
}

// Close releases the underlying database handle. It is safe to call on a nil
// receiver or nil Db.
func (s *Sqlite) Close() error {
	if s == nil || s.Db == nil {
		return nil
	}
	if err := s.Db.Close(); err != nil {
		return fmt.Errorf("sqlite: close: %w", err)
	}
	return nil
}
