package store

import "testing"

func TestCreateAndToggle(t *testing.T) {
	s := NewMemory()
	task := s.Create("learn Go")
	if task.ID != 1 || task.Done {
		t.Fatalf("unexpected task: %+v", task)
	}

	got, err := s.Toggle(task.ID)
	if err != nil || !got.Done {
		t.Fatalf("toggle failed: %+v, %v", got, err)
	}
}

func TestDeleteNotFound(t *testing.T) {
	s := NewMemory()
	if err := s.Delete(99); err != ErrNotFound {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}
