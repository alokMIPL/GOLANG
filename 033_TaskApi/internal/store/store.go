package store

import (
	"context"
	"errors"
	"slices"
	"sync"
	"testing"
	"time"
)

// ---------------------------------------------------------------------------
// Helpers
//
// Assumptions about the implementation (adjust if they differ):
//   - NewMemory() returns *Memory
//   - Create/Get/Toggle return a Task value (not a pointer)
//   - IDs are ints
//   - Memory has an unexported `now func() time.Time` field
//   - Go 1.21+ (for the standard "slices" package)
// ---------------------------------------------------------------------------

func mustCreate(t *testing.T, s *Memory, title string) Task {
	t.Helper()
	task, err := s.Create(context.Background(), title)
	if err != nil {
		t.Fatalf("Create(%q): %v", title, err)
	}
	return task
}

func mustGet(t *testing.T, s *Memory, id int) Task {
	t.Helper()
	task, err := s.Get(context.Background(), id)
	if err != nil {
		t.Fatalf("Get(%d): %v", id, err)
	}
	return task
}

func mustToggle(t *testing.T, s *Memory, id int) Task {
	t.Helper()
	task, err := s.Toggle(context.Background(), id)
	if err != nil {
		t.Fatalf("Toggle(%d): %v", id, err)
	}
	return task
}

func mustDelete(t *testing.T, s *Memory, id int) {
	t.Helper()
	if err := s.Delete(context.Background(), id); err != nil {
		t.Fatalf("Delete(%d): %v", id, err)
	}
}

func mustList(t *testing.T, s *Memory) []Task {
	t.Helper()
	list, err := s.List(context.Background())
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	return list
}

func listIDs(list []Task) []int {
	ids := make([]int, 0, len(list))
	for _, task := range list {
		ids = append(ids, task.ID)
	}
	return ids
}

// idOps are the operations that take an ID, so the same table drives both the
// "missing ID" and "cancelled context" tests.
var idOps = []struct {
	name string
	call func(ctx context.Context, s *Memory, id int) error
}{
	{"Get", func(ctx context.Context, s *Memory, id int) error {
		_, err := s.Get(ctx, id)
		return err
	}},
	{"Toggle", func(ctx context.Context, s *Memory, id int) error {
		_, err := s.Toggle(ctx, id)
		return err
	}},
	{"Delete", func(ctx context.Context, s *Memory, id int) error {
		return s.Delete(ctx, id)
	}},
}

// ---------------------------------------------------------------------------
// Create
// ---------------------------------------------------------------------------

func TestCreate_AssignsSequentialIDsAndTrimsTitle(t *testing.T) {
	t.Parallel()
	s := NewMemory()

	a := mustCreate(t, s, "  first  ")
	b := mustCreate(t, s, "second")

	if a.ID != 1 || b.ID != 2 {
		t.Fatalf("got IDs %d, %d; want 1, 2", a.ID, b.ID)
	}
	if a.Title != "first" {
		t.Fatalf("title = %q; want %q", a.Title, "first")
	}
}

func TestCreate_InitialState(t *testing.T) {
	t.Parallel()
	task := mustCreate(t, NewMemory(), "x")

	if task.Done {
		t.Error("new task Done = true; want false")
	}
	if task.CreatedAt.IsZero() {
		t.Error("new task CreatedAt is zero; want it set")
	}
}

func TestCreate_PreservesInnerWhitespace(t *testing.T) {
	t.Parallel()
	task := mustCreate(t, NewMemory(), "  a  b  ")

	if task.Title != "a  b" {
		t.Fatalf("title = %q; want %q", task.Title, "a  b")
	}
}

func TestCreate_InvalidTitle(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		title string
	}{
		{"empty", ""},
		{"spaces", "   "},
		{"tabs and newlines", "\t\n"},
		{"non-breaking space", "\u00a0"}, // assumes the impl uses strings.TrimSpace
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewMemory()

			_, err := s.Create(context.Background(), tt.title)
			if !errors.Is(err, ErrInvalidTitle) {
				t.Fatalf("Create(%q) error = %v; want ErrInvalidTitle", tt.title, err)
			}
			if n := len(mustList(t, s)); n != 0 {
				t.Fatalf("len(list) = %d after failed Create; want 0", n)
			}
		})
	}
}

func TestCreate_InvalidTitleDoesNotConsumeID(t *testing.T) {
	t.Parallel()
	s := NewMemory()

	if _, err := s.Create(context.Background(), "  "); !errors.Is(err, ErrInvalidTitle) {
		t.Fatalf("error = %v; want ErrInvalidTitle", err)
	}
	if task := mustCreate(t, s, "ok"); task.ID != 1 {
		t.Fatalf("ID = %d; want 1 (failed Create must not consume an ID)", task.ID)
	}
}

func TestCreate_UsesInjectedClockInUTC(t *testing.T) {
	t.Parallel()
	s := NewMemory()
	fixed := time.Date(2026, 1, 2, 3, 4, 5, 0, time.FixedZone("IST", 5*3600+1800))
	s.now = func() time.Time { return fixed }

	task := mustCreate(t, s, "x")

	if !task.CreatedAt.Equal(fixed) {
		t.Fatalf("CreatedAt = %v; want instant %v", task.CreatedAt, fixed)
	}
	if task.CreatedAt.Location() != time.UTC {
		t.Fatalf("CreatedAt location = %v; want UTC", task.CreatedAt.Location())
	}
}

// ---------------------------------------------------------------------------
// Missing IDs
// ---------------------------------------------------------------------------

func TestMissingID_ReturnsErrNotFound(t *testing.T) {
	t.Parallel()

	for _, op := range idOps {
		for _, id := range []int{42, 0, -1} {
			t.Run(op.name, func(t *testing.T) {
				s := NewMemory()
				mustCreate(t, s, "x") // so the store is non-empty

				if err := op.call(context.Background(), s, id); !errors.Is(err, ErrNotFound) {
					t.Fatalf("%s(%d) error = %v; want ErrNotFound", op.name, id, err)
				}
			})
		}
	}
}

// ---------------------------------------------------------------------------
// Toggle
// ---------------------------------------------------------------------------

func TestToggle_TwiceRestoresState(t *testing.T) {
	t.Parallel()
	s := NewMemory()
	task := mustCreate(t, s, "x")

	if got := mustToggle(t, s, task.ID); !got.Done {
		t.Fatal("first toggle: Done = false; want true")
	}
	if got := mustToggle(t, s, task.ID); got.Done {
		t.Fatal("second toggle: Done = true; want false")
	}
	if stored := mustGet(t, s, task.ID); stored.Done {
		t.Fatal("stored task Done = true; want false")
	}
}

func TestToggle_AffectsOnlyTargetAndKeepsCreatedAt(t *testing.T) {
	t.Parallel()
	s := NewMemory()
	a := mustCreate(t, s, "a")
	b := mustCreate(t, s, "b")

	mustToggle(t, s, a.ID)

	gotA := mustGet(t, s, a.ID)
	gotB := mustGet(t, s, b.ID)
	if !gotA.Done {
		t.Error("task a: Done = false; want true")
	}
	if gotB.Done {
		t.Error("task b: Done = true; want false (toggle must not leak)")
	}
	if !gotA.CreatedAt.Equal(a.CreatedAt) {
		t.Errorf("task a CreatedAt changed: %v -> %v", a.CreatedAt, gotA.CreatedAt)
	}
}

// ---------------------------------------------------------------------------
// List
// ---------------------------------------------------------------------------

func TestList_OrderedByIDAfterInterleavedOps(t *testing.T) {
	t.Parallel()
	s := NewMemory()

	for _, title := range []string{"a", "b", "c", "d"} {
		mustCreate(t, s, title)
	}
	mustDelete(t, s, 2)
	mustCreate(t, s, "e")

	got := listIDs(mustList(t, s))
	want := []int{1, 3, 4, 5}
	if !slices.Equal(got, want) {
		t.Fatalf("list IDs = %v; want %v", got, want)
	}
}

func TestList_EmptyReturnsEmptySlice(t *testing.T) {
	t.Parallel()
	list := mustList(t, NewMemory())

	if list == nil || len(list) != 0 {
		t.Fatalf("List = %#v; want empty non-nil slice", list)
	}
}

// ---------------------------------------------------------------------------
// Delete
// ---------------------------------------------------------------------------

func TestDelete_RemovesTask(t *testing.T) {
	t.Parallel()
	s := NewMemory()
	a := mustCreate(t, s, "a")

	mustDelete(t, s, a.ID)

	if _, err := s.Get(context.Background(), a.ID); !errors.Is(err, ErrNotFound) {
		t.Errorf("Get after Delete error = %v; want ErrNotFound", err)
	}
	if err := s.Delete(context.Background(), a.ID); !errors.Is(err, ErrNotFound) {
		t.Errorf("second Delete error = %v; want ErrNotFound", err)
	}
	if n := len(mustList(t, s)); n != 0 {
		t.Errorf("len(list) = %d; want 0", n)
	}
}

func TestDelete_DoesNotReuseIDs(t *testing.T) {
	t.Parallel()
	s := NewMemory()

	a := mustCreate(t, s, "a")
	mustDelete(t, s, a.ID)
	b := mustCreate(t, s, "b")

	if b.ID == a.ID {
		t.Fatalf("ID %d was reused after delete", a.ID)
	}
}

// ---------------------------------------------------------------------------
// Isolation: callers must not be able to mutate stored state
// ---------------------------------------------------------------------------

func TestGet_ReturnsCopy(t *testing.T) {
	t.Parallel()
	s := NewMemory()
	task := mustCreate(t, s, "x")

	got := mustGet(t, s, task.ID)
	got.Done = true
	got.Title = "mutated"

	again := mustGet(t, s, task.ID)
	if again.Done || again.Title != "x" {
		t.Fatalf("stored task changed via returned value: %+v", again)
	}
}

func TestList_ReturnsCopy(t *testing.T) {
	t.Parallel()
	s := NewMemory()
	mustCreate(t, s, "x")

	list := mustList(t, s)
	list[0].Done = true
	list[0].Title = "mutated"

	again := mustList(t, s)
	if again[0].Done || again[0].Title != "x" {
		t.Fatalf("stored task changed via returned slice: %+v", again[0])
	}
}

// ---------------------------------------------------------------------------
// Context handling
// ---------------------------------------------------------------------------

func TestContextErrors(t *testing.T) {
	t.Parallel()

	cancelled, cancel := context.WithCancel(context.Background())
	cancel()

	expired, cancelExpired := context.WithDeadline(context.Background(), time.Now().Add(-time.Hour))
	defer cancelExpired()

	tests := []struct {
		name string
		ctx  context.Context
		want error
	}{
		{"cancelled", cancelled, context.Canceled},
		{"deadline exceeded", expired, context.DeadlineExceeded},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewMemory()
			task := mustCreate(t, s, "x")

			if _, err := s.Create(tt.ctx, "y"); !errors.Is(err, tt.want) {
				t.Errorf("Create error = %v; want %v", err, tt.want)
			}
			if _, err := s.List(tt.ctx); !errors.Is(err, tt.want) {
				t.Errorf("List error = %v; want %v", err, tt.want)
			}
			for _, op := range idOps {
				if err := op.call(tt.ctx, s, task.ID); !errors.Is(err, tt.want) {
					t.Errorf("%s error = %v; want %v", op.name, err, tt.want)
				}
			}

			// The failed calls must not have changed anything.
			list := mustList(t, s)
			if len(list) != 1 {
				t.Fatalf("len(list) = %d; want 1", len(list))
			}
			if list[0].ID != task.ID || list[0].Done {
				t.Fatalf("task changed by failed calls: %+v", list[0])
			}
			if next := mustCreate(t, s, "z"); next.ID != task.ID+1 {
				t.Fatalf("next ID = %d; want %d (failed Create must not consume an ID)",
					next.ID, task.ID+1)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Concurrency. Run with: go test -race ./...
// ---------------------------------------------------------------------------

func TestConcurrentCreate_UniqueSequentialIDs(t *testing.T) {
	t.Parallel()

	const n = 200
	s := NewMemory()
	ctx := context.Background()

	ids := make(chan int, n)
	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			task, err := s.Create(ctx, "concurrent")
			if err != nil {
				t.Error(err)
				return
			}
			ids <- task.ID
		}()
	}
	wg.Wait()
	close(ids)

	got := make([]int, 0, n)
	for id := range ids {
		got = append(got, id)
	}
	slices.Sort(got)

	if len(got) != n {
		t.Fatalf("created %d tasks; want %d", len(got), n)
	}
	for i, id := range got {
		if id != i+1 {
			t.Fatalf("ids[%d] = %d; want %d (IDs must be unique and sequential)", i, id, i+1)
		}
	}
}

func TestConcurrentMixedOps(t *testing.T) {
	t.Parallel()

	const half = 50
	s := NewMemory()
	ctx := context.Background()

	for i := 0; i < 2*half; i++ {
		mustCreate(t, s, "seed")
	}

	// IDs 1..half are toggled (and read); IDs half+1..2*half are deleted.
	// Every ID is written by exactly one goroutine, so the end state is
	// deterministic even though the interleaving is not.
	var wg sync.WaitGroup
	for id := 1; id <= half; id++ {
		wg.Add(3)
		go func(id int) {
			defer wg.Done()
			if _, err := s.Toggle(ctx, id); err != nil {
				t.Errorf("Toggle(%d): %v", id, err)
			}
		}(id)
		go func(id int) {
			defer wg.Done()
			if _, err := s.Get(ctx, id); err != nil {
				t.Errorf("Get(%d): %v", id, err)
			}
		}(id)
		go func() {
			defer wg.Done()
			if _, err := s.List(ctx); err != nil {
				t.Errorf("List: %v", err)
			}
		}()
	}
	for id := half + 1; id <= 2*half; id++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			if err := s.Delete(ctx, id); err != nil {
				t.Errorf("Delete(%d): %v", id, err)
			}
		}(id)
	}
	wg.Wait()

	list := mustList(t, s)
	if len(list) != half {
		t.Fatalf("len(list) = %d; want %d", len(list), half)
	}
	for i, task := range list {
		if task.ID != i+1 {
			t.Errorf("list[%d].ID = %d; want %d", i, task.ID, i+1)
		}
		if !task.Done {
			t.Errorf("task %d: Done = false; want true", task.ID)
		}
	}
}
