package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

const taskColumns = "id, title, done, created_at, updated_at"

// TaskStore is the persistence contract used by the HTTP layer.
// Implementations validate titles and return errEmptyTitle, errTitleTooLong
// or errNotFound where appropriate.
type TaskStore interface {
	Create(ctx context.Context, title string) (Task, error)
	// List returns one page of tasks plus the total number of matches.
	List(ctx context.Context, p ListParams) ([]Task, int, error)
	Get(ctx context.Context, id int64) (Task, error)
	// Replace overwrites title and done (PUT semantics).
	Replace(ctx context.Context, id int64, title string, done bool) (Task, error)
	// Patch updates only the fields that are set (PATCH semantics).
	Patch(ctx context.Context, id int64, p TaskPatch) (Task, error)
	Toggle(ctx context.Context, id int64) (Task, error)
	Delete(ctx context.Context, id int64) error
	// DeleteCompleted removes all done tasks and reports how many were removed.
	DeleteCompleted(ctx context.Context) (int64, error)
	Ping(ctx context.Context) error
}

var _ TaskStore = (*sqliteStore)(nil)

// migrations are applied in order; the index+1 is stored in PRAGMA user_version.
// Never edit a released migration: append a new one instead.
var migrations = []string{
	// v1: initial schema (idempotent so databases created by older versions
	// of this program, which have user_version 0, upgrade cleanly).
	`CREATE TABLE IF NOT EXISTS tasks (
		id         INTEGER PRIMARY KEY AUTOINCREMENT,
		title      TEXT    NOT NULL,
		done       INTEGER NOT NULL DEFAULT 0,
		created_at INTEGER NOT NULL
	)`,
	// v2: track modification time.
	`ALTER TABLE tasks ADD COLUMN updated_at INTEGER NOT NULL DEFAULT 0;
	 UPDATE tasks SET updated_at = created_at;`,
	// v3: speed up done filtering and clear-completed.
	`CREATE INDEX IF NOT EXISTS idx_tasks_done ON tasks (done)`,
}

type sqliteStore struct {
	db  *sql.DB
	now func() time.Time // injectable for tests
}

// openSQLite opens (and pings) a SQLite database with sane pragmas.
func openSQLite(ctx context.Context, path string, maxConns int) (*sql.DB, error) {
	dsn := fmt.Sprintf("file:%s?_journal_mode=WAL&_busy_timeout=5000&_foreign_keys=on&_synchronous=NORMAL", path)
	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(maxConns)
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, err
	}
	return db, nil
}

func newSQLiteStore(db *sql.DB) *sqliteStore {
	return &sqliteStore{db: db, now: time.Now}
}

func (s *sqliteStore) clock() time.Time {
	return s.now().UTC().Truncate(time.Second)
}

// migrate brings the schema up to date. Each migration runs in its own
// transaction together with the version bump, so a failure leaves the
// database at the last good version. It assumes a single migrating process.
func (s *sqliteStore) migrate(ctx context.Context) error {
	var current int
	if err := s.db.QueryRowContext(ctx, "PRAGMA user_version").Scan(&current); err != nil {
		return fmt.Errorf("read schema version: %w", err)
	}
	if current > len(migrations) {
		return fmt.Errorf("database schema version %d is newer than this binary supports (%d)",
			current, len(migrations))
	}

	for i := current; i < len(migrations); i++ {
		tx, err := s.db.BeginTx(ctx, nil)
		if err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, migrations[i]); err != nil {
			tx.Rollback()
			return fmt.Errorf("migration %d: %w", i+1, err)
		}
		// PRAGMA does not accept bound parameters; i is an int, so this is safe.
		if _, err := tx.ExecContext(ctx, fmt.Sprintf("PRAGMA user_version = %d", i+1)); err != nil {
			tx.Rollback()
			return fmt.Errorf("migration %d: set version: %w", i+1, err)
		}
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("migration %d: commit: %w", i+1, err)
		}
	}
	return nil
}

func (s *sqliteStore) Ping(ctx context.Context) error { return s.db.PingContext(ctx) }

// ---------------------------------------------------------------------
// scanning helpers
// ---------------------------------------------------------------------

// scanner is satisfied by both *sql.Row and *sql.Rows.
type scanner interface {
	Scan(dest ...any) error
}

func scanTask(sc scanner) (Task, error) {
	var (
		t                Task
		done             int
		created, updated int64
	)
	if err := sc.Scan(&t.ID, &t.Title, &done, &created, &updated); err != nil {
		return Task{}, err
	}
	t.Done = done != 0
	t.CreatedAt = time.Unix(created, 0).UTC()
	t.UpdatedAt = time.Unix(updated, 0).UTC()
	return t, nil
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

// one runs a query expected to yield a single task row (SELECT or
// UPDATE ... RETURNING) and maps "no rows" to errNotFound.
func (s *sqliteStore) one(ctx context.Context, query string, args ...any) (Task, error) {
	t, err := scanTask(s.db.QueryRowContext(ctx, query, args...))
	if errors.Is(err, sql.ErrNoRows) {
		return Task{}, errNotFound
	}
	return t, err
}

// escapeLike escapes LIKE wildcards so user input is matched literally.
func escapeLike(s string) string {
	r := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)
	return r.Replace(s)
}

// ---------------------------------------------------------------------
// TaskStore implementation
// ---------------------------------------------------------------------

func (s *sqliteStore) Create(ctx context.Context, title string) (Task, error) {
	title, err := validateTitle(title)
	if err != nil {
		return Task{}, err
	}
	now := s.clock()
	return s.one(ctx,
		`INSERT INTO tasks (title, done, created_at, updated_at) VALUES (?, 0, ?, ?)
		 RETURNING `+taskColumns,
		title, now.Unix(), now.Unix())
}

func (s *sqliteStore) List(ctx context.Context, p ListParams) ([]Task, int, error) {
	var (
		where []string
		args  []any
	)
	if p.Done != nil {
		where = append(where, "done = ?")
		args = append(args, boolToInt(*p.Done))
	}
	if p.Query != "" {
		// SQLite's LIKE is case-insensitive for ASCII only.
		where = append(where, `title LIKE ? ESCAPE '\'`)
		args = append(args, "%"+escapeLike(p.Query)+"%")
	}
	clause := ""
	if len(where) > 0 {
		clause = " WHERE " + strings.Join(where, " AND ")
	}

	// Count and page are separate statements, so under concurrent writes the
	// total can be momentarily out of sync with the page. Acceptable here.
	var total int
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM tasks`+clause, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	order := "ASC"
	if p.Desc {
		order = "DESC"
	}
	pageArgs := append(append([]any{}, args...), p.Limit, p.Offset)
	rows, err := s.db.QueryContext(ctx,
		`SELECT `+taskColumns+` FROM tasks`+clause+` ORDER BY id `+order+` LIMIT ? OFFSET ?`,
		pageArgs...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	tasks := []Task{} // non-nil so JSON encodes as [] instead of null
	for rows.Next() {
		t, err := scanTask(rows)
		if err != nil {
			return nil, 0, err
		}
		tasks = append(tasks, t)
	}
	return tasks, total, rows.Err()
}

func (s *sqliteStore) Get(ctx context.Context, id int64) (Task, error) {
	return s.one(ctx, `SELECT `+taskColumns+` FROM tasks WHERE id = ?`, id)
}

func (s *sqliteStore) Replace(ctx context.Context, id int64, title string, done bool) (Task, error) {
	title, err := validateTitle(title)
	if err != nil {
		return Task{}, err
	}
	return s.one(ctx,
		`UPDATE tasks SET title = ?, done = ?, updated_at = ? WHERE id = ?
		 RETURNING `+taskColumns,
		title, boolToInt(done), s.clock().Unix(), id)
}

func (s *sqliteStore) Patch(ctx context.Context, id int64, p TaskPatch) (Task, error) {
	if p.isEmpty() {
		return s.Get(ctx, id)
	}

	// Column names below are fixed literals; only values are bound.
	sets := []string{"updated_at = ?"}
	args := []any{s.clock().Unix()}
	if p.Title != nil {
		title, err := validateTitle(*p.Title)
		if err != nil {
			return Task{}, err
		}
		sets = append(sets, "title = ?")
		args = append(args, title)
	}
	if p.Done != nil {
		sets = append(sets, "done = ?")
		args = append(args, boolToInt(*p.Done))
	}
	args = append(args, id)

	return s.one(ctx,
		`UPDATE tasks SET `+strings.Join(sets, ", ")+` WHERE id = ? RETURNING `+taskColumns,
		args...)
}

func (s *sqliteStore) Toggle(ctx context.Context, id int64) (Task, error) {
	return s.one(ctx,
		`UPDATE tasks SET done = 1 - done, updated_at = ? WHERE id = ?
		 RETURNING `+taskColumns,
		s.clock().Unix(), id)
}

func (s *sqliteStore) Delete(ctx context.Context, id int64) error {
	res, err := s.db.ExecContext(ctx, `DELETE FROM tasks WHERE id = ?`, id)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return errNotFound
	}
	return nil
}

func (s *sqliteStore) DeleteCompleted(ctx context.Context) (int64, error) {
	res, err := s.db.ExecContext(ctx, `DELETE FROM tasks WHERE done = 1`)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}
