package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

const taskColumns = "id, title, done, created_at, updated_at"

// maxBatch caps how many rows one batch call may touch. It also keeps us
// well below SQLite's bound-parameter limit (999 on older builds).
const maxBatch = 500

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

// BatchStore adds bulk and maintenance operations on top of TaskStore.
// It is a separate interface so existing TaskStore mocks keep compiling.
type BatchStore interface {
	TaskStore
	// CreateMany creates all tasks or none. Titles are validated first.
	CreateMany(ctx context.Context, titles []string) ([]Task, error)
	// SetDoneMany sets the done flag on the given ids and reports how many
	// rows actually changed. Unknown ids are ignored.
	SetDoneMany(ctx context.Context, ids []int64, done bool) (int64, error)
	// DeleteMany removes the given ids and reports how many existed.
	DeleteMany(ctx context.Context, ids []int64) (int64, error)
	// DeleteOlderThan removes tasks created before cutoff. If onlyDone is
	// true, pending tasks are kept.
	DeleteOlderThan(ctx context.Context, cutoff time.Time, onlyDone bool) (int64, error)
	Exists(ctx context.Context, id int64) (bool, error)
	Stats(ctx context.Context) (Stats, error)
}

var (
	_ TaskStore  = (*sqliteStore)(nil)
	_ BatchStore = (*sqliteStore)(nil)
	_ BatchStore = (*memoryStore)(nil)
)

// errTooManyItems is returned when a batch exceeds maxBatch.
var errTooManyItems = fmt.Errorf("too many items: at most %d per call", maxBatch)

// Stats is a cheap summary of the whole table.
type Stats struct {
	Total       int       `json:"total"`
	Done        int       `json:"done"`
	Pending     int       `json:"pending"`
	OldestTask  time.Time `json:"oldest_task,omitempty"`
	NewestTask  time.Time `json:"newest_task,omitempty"`
	LastUpdated time.Time `json:"last_updated,omitempty"`
}

// CompletionRate returns done/total in the range [0, 1].
func (s Stats) CompletionRate() float64 {
	if s.Total == 0 {
		return 0
	}
	return float64(s.Done) / float64(s.Total)
}

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

// Close releases the underlying database handle.
func (s *sqliteStore) Close() error { return s.db.Close() }

// SchemaVersion reports the applied migration count (useful for health checks).
func (s *sqliteStore) SchemaVersion(ctx context.Context) (int, error) {
	var v int
	err := s.db.QueryRowContext(ctx, "PRAGMA user_version").Scan(&v)
	return v, err
}

// ---------------------------------------------------------------------
// transaction + query helpers
// ---------------------------------------------------------------------

// querier is satisfied by *sql.DB and *sql.Tx so helpers work on both.
type querier interface {
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

// withTx runs fn inside a transaction, committing on success and rolling
// back on any error or panic.
func (s *sqliteStore) withTx(ctx context.Context, fn func(tx *sql.Tx) error) (err error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p)
		}
		if err != nil {
			tx.Rollback()
		}
	}()
	if err = fn(tx); err != nil {
		return err
	}
	return tx.Commit()
}

// placeholders returns "?, ?, ?" for n bound parameters.
func placeholders(n int) string {
	if n <= 0 {
		return ""
	}
	return strings.TrimSuffix(strings.Repeat("?, ", n), ", ")
}

// idArgs converts ids to []any, removing duplicates while keeping order.
func idArgs(ids []int64) []any {
	seen := make(map[int64]struct{}, len(ids))
	out := make([]any, 0, len(ids))
	for _, id := range ids {
		if _, dup := seen[id]; dup {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
}

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

// oneOn runs a query expected to yield a single task row on any querier
// (a DB or a transaction) and maps "no rows" to errNotFound.
func oneOn(ctx context.Context, q querier, query string, args ...any) (Task, error) {
	t, err := scanTask(q.QueryRowContext(ctx, query, args...))
	if errors.Is(err, sql.ErrNoRows) {
		return Task{}, errNotFound
	}
	return t, err
}

// one runs a query expected to yield a single task row (SELECT or
// UPDATE ... RETURNING) and maps "no rows" to errNotFound.
func (s *sqliteStore) one(ctx context.Context, query string, args ...any) (Task, error) {
	return oneOn(ctx, s.db, query, args...)
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

// ---------------------------------------------------------------------
// BatchStore implementation
// ---------------------------------------------------------------------

func (s *sqliteStore) CreateMany(ctx context.Context, titles []string) ([]Task, error) {
	if len(titles) == 0 {
		return []Task{}, nil
	}
	if len(titles) > maxBatch {
		return nil, errTooManyItems
	}

	// Validate everything up front so a bad title never leaves a partial batch.
	clean := make([]string, len(titles))
	for i, raw := range titles {
		t, err := validateTitle(raw)
		if err != nil {
			return nil, fmt.Errorf("item %d: %w", i, err)
		}
		clean[i] = t
	}

	now := s.clock().Unix()
	out := make([]Task, 0, len(clean))
	err := s.withTx(ctx, func(tx *sql.Tx) error {
		for _, title := range clean {
			t, err := oneOn(ctx, tx,
				`INSERT INTO tasks (title, done, created_at, updated_at) VALUES (?, 0, ?, ?)
				 RETURNING `+taskColumns,
				title, now, now)
			if err != nil {
				return err
			}
			out = append(out, t)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (s *sqliteStore) SetDoneMany(ctx context.Context, ids []int64, done bool) (int64, error) {
	args := idArgs(ids)
	if len(args) == 0 {
		return 0, nil
	}
	if len(args) > maxBatch {
		return 0, errTooManyItems
	}
	// Only touch rows whose state differs, so updated_at stays meaningful
	// and the reported count reflects real changes.
	query := `UPDATE tasks SET done = ?, updated_at = ?
	          WHERE done != ? AND id IN (` + placeholders(len(args)) + `)`
	full := append([]any{boolToInt(done), s.clock().Unix(), boolToInt(done)}, args...)
	res, err := s.db.ExecContext(ctx, query, full...)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

func (s *sqliteStore) DeleteMany(ctx context.Context, ids []int64) (int64, error) {
	args := idArgs(ids)
	if len(args) == 0 {
		return 0, nil
	}
	if len(args) > maxBatch {
		return 0, errTooManyItems
	}
	res, err := s.db.ExecContext(ctx,
		`DELETE FROM tasks WHERE id IN (`+placeholders(len(args))+`)`, args...)
	if err != nil {
		return 0;, err
	}
	return res.RowsAffected()
}

func (s *sqliteStore) DeleteOlderThan(ctx context.Context, cutoff time.Time, onlyDone bool) (int64, error) {
	query := `DELETE FROM tasks WHERE created_at < ?`
	if onlyDone {
		query += ` AND done = 1`
	}
	res, err := s.db.ExecContext(ctx, query, cutoff.UTC().Unix())
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

func (s *sqliteStore) Exists(ctx context.Context, id int64) (bool, error) {
	var one int
	err := s.db.QueryRowContext(ctx, `SELECT 1 FROM tasks WHERE id = ? LIMIT 1`, id).Scan(&one)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	return err == nil, err
}

func (s *sqliteStore) Stats(ctx context.Context) (Stats, error) {
	var (
		st                      Stats
		oldest, newest, updated int64
	)
	// COALESCE keeps the scan working on an empty table (MIN/MAX/SUM are NULL).
	err := s.db.QueryRowContext(ctx,
		`SELECT COUNT(*),
		        COALESCE(SUM(done), 0),
		        COALESCE(MIN(created_at), 0),
		        COALESCE(MAX(created_at), 0),
		        COALESCE(MAX(updated_at), 0)
		 FROM tasks`).Scan(&st.Total, &st.Done, &oldest, &newest, &updated)
	if err != nil {
		return Stats{}, err
	}
	st.Pending = st.Total - st.Done
	if st.Total > 0 {
		st.OldestTask = time.Unix(oldest, 0).UTC()
		st.NewestTask = time.Unix(newest, 0).UTC()
		st.LastUpdated = time.Unix(updated, 0).UTC()
	}
	return st, nil
}

// ---------------------------------------------------------------------
// maintenance
// ---------------------------------------------------------------------

// Backup writes a consistent, compacted copy of the database to dest using
// VACUUM INTO. dest must not already exist. It is safe to run while the
// application is serving traffic.
func (s *sqliteStore) Backup(ctx context.Context, dest string) error {
	if strings.TrimSpace(dest) == "" {
		return errors.New("backup: empty destination path")
	}
	if _, err := s.db.ExecContext(ctx, `VACUUM INTO ?`, dest); err != nil {
		return fmt.Errorf("backup to %q: %w", dest, err)
	}
	return nil
}

// Checkpoint flushes the WAL file into the main database and truncates it.
// Call it periodically (or before a file-level backup) to keep the WAL small.
func (s *sqliteStore) Checkpoint(ctx context.Context) error {
	if _, err := s.db.ExecContext(ctx, `PRAGMA wal_checkpoint(TRUNCATE)`); err != nil {
		return fmt.Errorf("wal checkpoint: %w", err)
	}
	return nil
}

// ---------------------------------------------------------------------
// in-memory implementation (for unit tests and local development)
// ---------------------------------------------------------------------

// memoryStore is a goroutine-safe BatchStore backed by a map. It mirrors
// the validation and error behaviour of sqliteStore so HTTP-layer tests can
// run without a database.
type memoryStore struct {
	mu     sync.RWMutex
	tasks  map[int64]Task
	nextID int64
	now    func() time.Time
}

func newMemoryStore() *memoryStore {
	return &memoryStore{
		tasks:  make(map[int64]Task),
		nextID: 1,
		now:    time.Now,
	}
}

func (m *memoryStore) clock() time.Time {
	return m.now().UTC().Truncate(time.Second)
}

// insertLocked adds a task; the caller must hold the write lock.
func (m *memoryStore) insertLocked(title string) Task {
	now := m.clock()
	t := Task{ID: m.nextID, Title: title, CreatedAt: now, UpdatedAt: now}
	m.nextID++
	m.tasks[t.ID] = t
	return t
}

// sortedLocked returns all tasks ordered by id; the caller must hold a lock.
func (m *memoryStore) sortedLocked(desc bool) []Task {
	all := make([]Task, 0, len(m.tasks))
	for _, t := range m.tasks {
		all = append(all, t)
	}
	sort.Slice(all, func(i, j int) bool {
		if desc {
			return all[i].ID > all[j].ID
		}
		return all[i].ID < all[j].ID
	})
	return all
}

func (m *memoryStore) Ping(ctx context.Context) error { return ctx.Err() }

func (m *memoryStore) Create(ctx context.Context, title string) (Task, error) {
	title, err := validateTitle(title)
	if err != nil {
		return Task{}, err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.insertLocked(title), nil
}

func (m *memoryStore) List(ctx context.Context, p ListParams) ([]Task, int, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	needle := strings.ToLower(p.Query)
	matches := []Task{}
	for _, t := range m.sortedLocked(p.Desc) {
		if p.Done != nil && t.Done != *p.Done {
			continue
		}
		if needle != "" && !strings.Contains(strings.ToLower(t.Title), needle) {
			continue
		}
		matches = append(matches, t)
	}

	total := len(matches)
	start := p.Offset
	if start < 0 {
		start = 0
	}
	if start > total {
		start = total
	}
	end := total
	if p.Limit > 0 && start+p.Limit < total {
		end = start + p.Limit
	}
	return matches[start:end], total, nil
}

func (m *memoryStore) Get(ctx context.Context, id int64) (Task, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	t, ok := m.tasks[id]
	if !ok {
		return Task{}, errNotFound
	}
	return t, nil
}

func (m *memoryStore) Replace(ctx context.Context, id int64, title string, done bool) (Task, error) {
	title, err := validateTitle(title)
	if err != nil {
		return Task{}, err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	t, ok := m.tasks[id]
	if !ok {
		return Task{}, errNotFound
	}
	t.Title, t.Done, t.UpdatedAt = title, done, m.clock()
	m.tasks[id] = t
	return t, nil
}

func (m *memoryStore) Patch(ctx context.Context, id int64, p TaskPatch) (Task, error) {
	if p.isEmpty() {
		return m.Get(ctx, id)
	}
	// Validate before locking so we never half-apply a patch.
	var newTitle string
	if p.Title != nil {
		var err error
		if newTitle, err = validateTitle(*p.Title); err != nil {
			return Task{}, err
		}
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	t, ok := m.tasks[id]
	if !ok {
		return Task{}, errNotFound
	}
	if p.Title != nil {
		t.Title = newTitle
	}
	if p.Done != nil {
		t.Done = *p.Done
	}
	t.UpdatedAt = m.clock()
	m.tasks[id] = t
	return t, nil
}

func (m *memoryStore) Toggle(ctx context.Context, id int64) (Task, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	t, ok := m.tasks[id]
	if !ok {
		return Task{}, errNotFound
	}
	t.Done = !t.Done
	t.UpdatedAt = m.clock()
	m.tasks[id] = t
	return t, nil
}

func (m *memoryStore) Delete(ctx context.Context, id int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.tasks[id]; !ok {
		return errNotFound
	}
	delete(m.tasks, id)
	return nil
}

func (m *memoryStore) DeleteCompleted(ctx context.Context) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var n int64
	for id, t := range m.tasks {
		if t.Done {
			delete(m.tasks, id)
			n++
		}
	}
	return n, nil
}

func (m *memoryStore) CreateMany(ctx context.Context, titles []string) ([]Task, error) {
	if len(titles) == 0 {
		return []Task{}, nil
	}
	if len(titles) > maxBatch {
		return nil, errTooManyItems
	}
	clean := make([]string, len(titles))
	for i, raw := range titles {
		t, err := validateTitle(raw)
		if err != nil {
			return nil, fmt.Errorf("item %d: %w", i, err)
		}
		clean[i] = t
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]Task, 0, len(clean))
	for _, title := range clean {
		out = append(out, m.insertLocked(title))
	}
	return out, nil
}

func (m *memoryStore) SetDoneMany(ctx context.Context, ids []int64, done bool) (int64, error) {
	if len(ids) > maxBatch {
		return 0, errTooManyItems
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	var n int64
	now := m.clock()
	for _, a := range idArgs(ids) {
		id := a.(int64)
		t, ok := m.tasks[id]
		if !ok || t.Done == done {
			continue
		}
		t.Done, t.UpdatedAt = done, now
		m.tasks[id] = t
		n++
	}
	return n, nil
}

func (m *memoryStore) DeleteMany(ctx context.Context, ids []int64) (int64, error) {
	if len(ids) > maxBatch {
		return 0, errTooManyItems
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	var n int64
	for _, a := range idArgs(ids) {
		id := a.(int64)
		if _, ok := m.tasks[id]; ok {
			delete(m.tasks, id)
			n++
		}
	}
	return n, nil
}

func (m *memoryStore) DeleteOlderThan(ctx context.Context, cutoff time.Time, onlyDone bool) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var n int64
	for id, t := range m.tasks {
		if t.CreatedAt.Before(cutoff) && (!onlyDone || t.Done) {
			delete(m.tasks, id)
			n++
		}
	}
	return n, nil
}

func (m *memoryStore) Exists(ctx context.Context, id int64) (bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	_, ok := m.tasks[id]
	return ok, nil
}

func (m *memoryStore) Stats(ctx context.Context) (Stats, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var st Stats
	for _, t := range m.tasks {
		st.Total++
		if t.Done {
			st.Done++
		}
		if st.OldestTask.IsZero() || t.CreatedAt.Before(st.OldestTask) {
			st.OldestTask = t.CreatedAt
		}
		if t.CreatedAt.After(st.NewestTask) {
			st.NewestTask = t.CreatedAt
		}
		if t.UpdatedAt.After(st.LastUpdated) {
			st.LastUpdated = t.UpdatedAt
		}
	}
	st.Pending = st.Total - st.Done
	return st, nil
}
