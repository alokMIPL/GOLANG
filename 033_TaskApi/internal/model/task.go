package model

import (
	"errors"
	"fmt"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

// ---------------------------------------------------------------------
// limits and sentinel errors
// ---------------------------------------------------------------------

const (
	// MaxTitleLength is the maximum title length in characters (runes).
	MaxTitleLength = 200

	// DefaultLimit and MaxLimit control list pagination.
	DefaultLimit = 50
	MaxLimit     = 200
)

var (
	ErrEmptyTitle        = errors.New("title must not be empty")
	ErrTitleTooLong      = fmt.Errorf("title must be at most %d characters", MaxTitleLength)
	ErrNotFound          = errors.New("task not found")
	ErrInvalidID         = errors.New("id must be a positive integer")
	ErrInvalidPagination = errors.New("invalid pagination parameters")
	ErrInvalidFilter     = errors.New("invalid filter parameters")
	ErrEmptyPatch        = errors.New("patch must set at least one field")
)

// ---------------------------------------------------------------------
// Task
// ---------------------------------------------------------------------

// Task is a single to-do item.
type Task struct {
	ID        int       `json:"id"`
	Title     string    `json:"title"`
	Done      bool      `json:"done"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Validate reports whether the task is internally consistent.
func (t Task) Validate() error {
	if t.ID < 0 {
		return ErrInvalidID
	}
	if _, err := NormalizeTitle(t.Title); err != nil {
		return err
	}
	if !t.UpdatedAt.IsZero() && t.UpdatedAt.Before(t.CreatedAt) {
		return errors.New("updated_at must not be before created_at")
	}
	return nil
}

// Clone returns a copy of the task. Task has only value fields, so a plain
// copy is already deep; the method exists to make intent explicit.
func (t Task) Clone() Task { return t }

// Age returns how long ago the task was created, relative to now.
func (t Task) Age(now time.Time) time.Duration {
	if now.Before(t.CreatedAt) {
		return 0
	}
	return now.Sub(t.CreatedAt)
}

// IsModified reports whether the task changed after it was created.
func (t Task) IsModified() bool { return t.UpdatedAt.After(t.CreatedAt) }

// Status returns "done" or "pending" for display and logging.
func (t Task) Status() string {
	if t.Done {
		return "done"
	}
	return "pending"
}

// String implements fmt.Stringer, e.g. `#7 [done] "Buy milk"`.
func (t Task) String() string {
	return fmt.Sprintf("#%d [%s] %q", t.ID, t.Status(), t.Title)
}

// Toggle flips the done flag and stamps UpdatedAt.
func (t *Task) Toggle(now time.Time) {
	t.Done = !t.Done
	t.UpdatedAt = now
}

// SetDone sets the done flag. It reports whether anything changed.
func (t *Task) SetDone(done bool, now time.Time) bool {
	if t.Done == done {
		return false
	}
	t.Done = done
	t.UpdatedAt = now
	return true
}

// Rename validates and sets a new title. It reports whether anything changed.
func (t *Task) Rename(title string, now time.Time) (bool, error) {
	clean, err := NormalizeTitle(title)
	if err != nil {
		return false, err
	}
	if clean == t.Title {
		return false, nil
	}
	t.Title = clean
	t.UpdatedAt = now
	return true, nil
}

// NormalizeTitle trims the title, collapses runs of whitespace into single
// spaces and enforces the length limits.
func NormalizeTitle(title string) (string, error) {
	clean := strings.Join(strings.Fields(title), " ")
	if clean == "" {
		return "", ErrEmptyTitle
	}
	if utf8.RuneCountInString(clean) > MaxTitleLength {
		return "", ErrTitleTooLong
	}
	return clean, nil
}

// ---------------------------------------------------------------------
// request payloads
// ---------------------------------------------------------------------

// CreateTaskRequest is the body of POST /tasks.
type CreateTaskRequest struct {
	Title string `json:"title"`
}

// Validate normalises the request in place.
func (r *CreateTaskRequest) Validate() error {
	title, err := NormalizeTitle(r.Title)
	if err != nil {
		return err
	}
	r.Title = title
	return nil
}

// ReplaceTaskRequest is the body of PUT /tasks/{id} (full overwrite).
type ReplaceTaskRequest struct {
	Title string `json:"title"`
	Done  bool   `json:"done"`
}

// Validate normalises the request in place.
func (r *ReplaceTaskRequest) Validate() error {
	title, err := NormalizeTitle(r.Title)
	if err != nil {
		return err
	}
	r.Title = title
	return nil
}

// BulkCreateRequest is the body of POST /tasks/bulk.
type BulkCreateRequest struct {
	Titles []string `json:"titles"`
}

// Validate normalises every title and reports which item was invalid.
func (r *BulkCreateRequest) Validate() error {
	if len(r.Titles) == 0 {
		return errors.New("titles must contain at least one item")
	}
	for i, raw := range r.Titles {
		title, err := NormalizeTitle(raw)
		if err != nil {
			return fmt.Errorf("titles[%d]: %w", i, err)
		}
		r.Titles[i] = title
	}
	return nil
}

// BulkIDsRequest identifies several tasks, e.g. for bulk delete or bulk
// done/undone updates.
type BulkIDsRequest struct {
	IDs  []int `json:"ids"`
	Done *bool `json:"done,omitempty"`
}

// Validate rejects empty lists and non-positive ids, and removes duplicates.
func (r *BulkIDsRequest) Validate() error {
	if len(r.IDs) == 0 {
		return errors.New("ids must contain at least one item")
	}
	seen := make(map[int]struct{}, len(r.IDs))
	unique := r.IDs[:0:0]
	for i, id := range r.IDs {
		if id <= 0 {
			return fmt.Errorf("ids[%d]: %w", i, ErrInvalidID)
		}
		if _, dup := seen[id]; dup {
			continue
		}
		seen[id] = struct{}{}
		unique = append(unique, id)
	}
	r.IDs = unique
	return nil
}

// TaskPatch holds the optional fields of a PATCH request. A nil pointer
// means "leave unchanged", which is what lets false and "" be sent explicitly.
type TaskPatch struct {
	Title *string `json:"title,omitempty"`
	Done  *bool   `json:"done,omitempty"`
}

// IsEmpty reports whether the patch changes nothing.
func (p TaskPatch) IsEmpty() bool { return p.Title == nil && p.Done == nil }

// Validate normalises the title (if set) and rejects empty patches.
func (p *TaskPatch) Validate() error {
	if p.IsEmpty() {
		return ErrEmptyPatch
	}
	if p.Title != nil {
		title, err := NormalizeTitle(*p.Title)
		if err != nil {
			return err
		}
		p.Title = &title
	}
	return nil
}

// Apply copies the patch onto t and reports whether anything changed.
// It validates first, so t is untouched when an error is returned.
func (p TaskPatch) Apply(t *Task, now time.Time) (bool, error) {
	if p.IsEmpty() {
		return false, nil
	}
	if err := (&p).Validate(); err != nil {
		return false, err
	}
	changed := false
	if p.Title != nil {
		c, _ := t.Rename(*p.Title, now) // already validated
		changed = changed || c
	}
	if p.Done != nil {
		changed = t.SetDone(*p.Done, now) || changed
	}
	return changed, nil
}

// ---------------------------------------------------------------------
// listing, filtering and pagination
// ---------------------------------------------------------------------

// ListParams describes one page of a filtered task listing.
type ListParams struct {
	Done   *bool  // nil = both done and pending
	Query  string // case-insensitive title substring
	Desc   bool   // newest first when true
	Limit  int
	Offset int
}

// Normalize fills defaults and clamps values into their valid ranges.
func (p *ListParams) Normalize() {
	p.Query = strings.TrimSpace(p.Query)
	if p.Limit <= 0 {
		p.Limit = DefaultLimit
	}
	if p.Limit > MaxLimit {
		p.Limit = MaxLimit
	}
	if p.Offset < 0 {
		p.Offset = 0
	}
}

// Validate checks values without silently correcting them.
func (p ListParams) Validate() error {
	if p.Limit < 0 || p.Limit > MaxLimit {
		return fmt.Errorf("%w: limit must be between 0 and %d", ErrInvalidPagination, MaxLimit)
	}
	if p.Offset < 0 {
		return fmt.Errorf("%w: offset must not be negative", ErrInvalidPagination)
	}
	return nil
}

// ParseListParams builds ListParams from URL query values:
//
//	?done=true|false  ?q=text  ?order=asc|desc  ?limit=N  ?offset=N
//
// The result is already normalised.
func ParseListParams(v url.Values) (ListParams, error) {
	var p ListParams

	if raw := v.Get("done"); raw != "" {
		b, err := strconv.ParseBool(raw)
		if err != nil {
			return p, fmt.Errorf("%w: done must be true or false", ErrInvalidFilter)
		}
		p.Done = &b
	}
	p.Query = v.Get("q")

	switch strings.ToLower(v.Get("order")) {
	case "", "asc":
	case "desc":
		p.Desc = true
	default:
		return p, fmt.Errorf("%w: order must be asc or desc", ErrInvalidFilter)
	}

	if raw := v.Get("limit"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil {
			return p, fmt.Errorf("%w: limit must be a number", ErrInvalidPagination)
		}
		p.Limit = n
	}
	if raw := v.Get("offset"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil {
			return p, fmt.Errorf("%w: offset must be a number", ErrInvalidPagination)
		}
		p.Offset = n
	}

	if err := p.Validate(); err != nil {
		return p, err
	}
	p.Normalize()
	return p, nil
}

// Page is a generic paginated response envelope.
type Page[T any] struct {
	Items   []T  `json:"items"`
	Total   int  `json:"total"`
	Limit   int  `json:"limit"`
	Offset  int  `json:"offset"`
	HasMore bool `json:"has_more"`
}

// NewPage wraps items with pagination metadata. A nil slice is replaced with
// an empty one so JSON encodes [] rather than null.
func NewPage[T any](items []T, total int, p ListParams) Page[T] {
	if items == nil {
		items = []T{}
	}
	return Page[T]{
		Items:   items,
		Total:   total,
		Limit:   p.Limit,
		Offset:  p.Offset,
		HasMore: p.Offset+len(items) < total,
	}
}

// ---------------------------------------------------------------------
// statistics
// ---------------------------------------------------------------------

// Stats summarises the task list.
type Stats struct {
	Total   int `json:"total"`
	Done    int `json:"done"`
	Pending int `json:"pending"`
}

// CompletionRate returns done/total in [0, 1], or 0 for an empty list.
func (s Stats) CompletionRate() float64 {
	if s.Total == 0 {
		return 0
	}
	return float64(s.Done) / float64(s.Total)
}

// ComputeStats counts tasks by status.
func ComputeStats(tasks []Task) Stats {
	s := Stats{Total: len(tasks)}
	for _, t := range tasks {
		if t.Done {
			s.Done++
		}
	}
	s.Pending = s.Total - s.Done
	return s
}

// ---------------------------------------------------------------------
// slice helpers
// ---------------------------------------------------------------------

// Filter returns the tasks that match p (done flag and title substring).
// Pagination fields are ignored; use Paginate for that.
func Filter(tasks []Task, p ListParams) []Task {
	needle := strings.ToLower(p.Query)
	out := make([]Task, 0, len(tasks))
	for _, t := range tasks {
		if p.Done != nil && t.Done != *p.Done {
			continue
		}
		if needle != "" && !strings.Contains(strings.ToLower(t.Title), needle) {
			continue
		}
		out = append(out, t)
	}
	return out
}

// SortByID orders tasks by id, ascending or descending, in place.
func SortByID(tasks []Task, desc bool) {
	sort.SliceStable(tasks, func(i, j int) bool {
		if desc {
			return tasks[i].ID > tasks[j].ID
		}
		return tasks[i].ID < tasks[j].ID
	})
}

// SortByCreated orders tasks by creation time (ties broken by id), in place.
func SortByCreated(tasks []Task, desc bool) {
	sort.SliceStable(tasks, func(i, j int) bool {
		a, b := tasks[i], tasks[j]
		if a.CreatedAt.Equal(b.CreatedAt) {
			if desc {
				return a.ID > b.ID
			}
			return a.ID < b.ID
		}
		if desc {
			return a.CreatedAt.After(b.CreatedAt)
		}
		return a.CreatedAt.Before(b.CreatedAt)
	})
}

// Paginate returns the window of tasks selected by p.Offset and p.Limit,
// plus the total before slicing. It never panics on out-of-range values.
func Paginate(tasks []Task, p ListParams) ([]Task, int) {
	total := len(tasks)
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
	return tasks[start:end], total
}

// IndexByID maps task ids to tasks for quick lookups.
func IndexByID(tasks []Task) map[int]Task {
	m := make(map[int]Task, len(tasks))
	for _, t := range tasks {
		m[t.ID] = t
	}
	return m
}

// IDs extracts the ids of the given tasks, preserving order.
func IDs(tasks []Task) []int {
	out := make([]int, len(tasks))
	for i, t := range tasks {
		out[i] = t.ID
	}
	return out
}

// Titles extracts the titles of the given tasks, preserving order.
func Titles(tasks []Task) []string {
	out := make([]string, len(tasks))
	for i, t := range tasks {
		out[i] = t.Title
	}
	return out
}
