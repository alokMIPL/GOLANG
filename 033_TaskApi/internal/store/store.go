package store

import (
	"context"
	"errors"
	"sort"
	"sync"
	"testing"
	"time"
)

func TestCreate_AssignsSequentialIDsAndTrimsTitle(t *testing.T) {
	s := NewMemory()
	ctx := context.Background()

	a, err := s.Create(ctx, "  first  ")
	if err != nil {
		t.Fatal(err)
	}
	b, _ := s.Create(ctx, "second")

	if a.ID != 1 || b.ID != 2 {
		t.Fatalf("got IDs %d, %d; want 1, 2", a.ID, b.ID)
	}
	if a.Title != "first" {
		t.Fatalf("title = %q; want %q", a.Title, "first")
	}
}

func TestCreate_InvalidTitle(t *testing.T) {
	s := NewMemory()
	for _, title := range []string{"", "   ", "\t\n"} {
		if _, err := s.Create(context.Background(), title); !errors.Is(err, ErrInvalidTitle) {
			t.Errorf("Create(%q) error = %v; want ErrInvalidTitle", title, err)
		}
	}
}

func TestCreate_UsesInjectedClockInUTC(t *testing.T) {
	s := NewMemory()
	fixed := time.Date(2026, 1, 2, 3, 4, 5, 0, time.FixedZone("IST", 5*3600+1800))
	s.now = func() time.Time { return fixed }

	task, _ := s.Create(context.Background(), "x")
	if !task.CreatedAt.Equal(fixed) {
		t.Fatalf("CreatedAt = %v; want instant %v", task.CreatedAt, fixed)
	}
	if task.CreatedAt.Location() != time.UTC {
		t.Fatalf("CreatedAt location = %v; want UTC", task.CreatedAt.Location())
	}
}

func TestMissingID_ReturnsErrNotFound(t *testing.T) {
	s := NewMemory()
	ctx := context.Background()

	if _, err := s.Get(ctx, 42); !errors.Is(err, ErrNotFound) {
		t.Errorf("Get error = %v; want ErrNotFound", err)
	}
	if _, err := s.Toggle(ctx, 42); !errors.Is(err, ErrNotFound) {
		t.Errorf("Toggle error = %v; want ErrNotFound", err)
	}
	if err := s.Delete(ctx, 42); !errors.Is(err, ErrNotFound) {
		t.Errorf("Delete error = %v; want ErrNotFound", err)
	}
}

func TestToggle_TwiceRestoresState(t *testing.T) {
	s := NewMemory()
	ctx := context.Background()
	task, _ := s.Create(ctx, "x")

	got, _ := s.Toggle(ctx, task.ID)
	if !got.Done {
		t.Fatal("first toggle: Done = false; want true")
	}
	got, _ = s.Toggle(ctx, task.ID)
	if got.Done {
		t.Fatal("second toggle: Done = true; want false")
	}

	stored, _ := s.Get(ctx, task.ID)
	if stored.Done {
		t.Fatal("stored task Done = true; want false")
	}
}

func TestList_OrderedByIDAfterInterleavedOps(t *testing.T) {
	s := NewMemory()
	ctx := context.Background()

	for _, title := range []string{"a", "b", "c", "d"} {
		if _, err := s.Create(ctx, title); err != nil {
			t.Fatal(err)
		}
	}
	if err := s.Delete(ctx, 2); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Create(ctx, "e"); err != nil {
		t.Fatal(err)
	}

	list, err := s.List(ctx)
	if err != nil {
		t.Fatal(err)
	}
	want := []int{1, 3, 4, 5}
	if len(list) != len(want) {
		t.Fatalf("len = %d; want %d", len(list), len(want))
	}
	for i, task := range list {
		if task.ID != want[i] {
			t.Errorf("list[%d].ID = %d; want %d", i, task.ID, want[i])
		}
	}
}

func TestList_EmptyReturnsEmptySlice(t *testing.T) {
	list, err := NewMemory().List(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if list == nil || len(list) != 0 {
		t.Fatalf("List = %#v; want empty non-nil slice", list)
	}
}

func TestDelete_DoesNotReuseIDs(t *testing.T) {
	s := NewMemory()
	ctx := context.Background()

	a, _ := s.Create(ctx, "a")
	if err := s.Delete(ctx, a.ID); err != nil {
		t.Fatal(err)
	}
	b, _ := s.Create(ctx, "b")

	if b.ID == a.ID {
		t.Fatalf("ID %d was reused after delete", a.ID)
	}
}

func TestCancelledContext(t *testing.T) {
	s := NewMemory()
	task, _ := s.Create(context.Background(), "x")

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if _, err := s.Create(ctx, "y"); !errors.Is(err, context.Canceled) {
		t.Errorf("Create error = %v; want context.Canceled", err)
	}
	if _, err := s.List(ctx); !errors.Is(err, context.Canceled) {
		t.Errorf("List error = %v; want context.Canceled", err)
	}
	if _, err := s.Get(ctx, task.ID); !errors.Is(err, context.Canceled) {
		t.Errorf("Get error = %v; want context.Canceled", err)
	}
	if _, err := s.Toggle(ctx, task.ID); !errors.Is(err, context.Canceled) {
		t.Errorf("Toggle error = %v; want context.Canceled", err)
	}
	if err := s.Delete(ctx, task.ID); !errors.Is(err, context.Canceled) {
		t.Errorf("Delete error = %v; want context.Canceled", err)
	}

	// The cancelled calls must not have changed anything.
	list, _ := s.List(context.Background())
	if len(list) != 1 {
		t.Fatalf("len(list) = %d; want 1", len(list))
	}
}

// Run with: go test -race ./...
func TestConcurrentCreate_UniqueSequentialIDs(t *testing.T) {
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
	sort.Ints(got)

	if len(got) != n {
		t.Fatalf("created %d tasks; want %d", len(got), n)
	}
	for i, id := range got {
		if id != i+1 {
			t.Fatalf("ids[%d] = %d; want %d (IDs must be unique and sequential)", i, id, i+1)
		}
	}
}

func TestConcurrentMixedOps_NoRaces(t *testing.T) {
	s := NewMemory()
	ctx := context.Background()
	for i := 0; i < 50; i++ {
		if _, err := s.Create(ctx, "seed"); err != nil {
			t.Fatal(err)
		}
	}

	var wg sync.WaitGroup
	for i := 1; i <= 50; i++ {
		wg.Add(3)
		go func(id int) { defer wg.Done(); _, _ = s.Toggle(ctx, id) }(i)
		go func(id int) { defer wg.Done(); _, _ = s.Get(ctx, id) }(i)
		go func() { defer wg.Done(); _, _ = s.List(ctx) }()
	}
	wg.Wait()
}
