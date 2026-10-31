package store

import (
	"context"
	"errors"
	"sort"
	"sync"
	"time"

	"github.com/yourname/taskapi/internal/model"
)

var ErrNotFound = errors.New("task not found")

type TaskStore interface {
	Create(ctx context.Context, title string) (model.Task, error)
	List(ctx context.Context) ([]model.Task, error)
	Get(ctx context.Context, id int) (model.Task, error)
	Toggle(ctx context.Context, id int) (model.Task, error)
	Delete(ctx context.Context, id int) error
}

type MemoryStore struct {
	mu     sync.RWMutex
	tasks  map[int]model.Task
	nextID int
}

func NewMemory() *MemoryStore {
	return &MemoryStore{tasks: make(map[int]model.Task), nextID: 1}
}

func (s *MemoryStore) Create(_ context.Context, title string) (model.Task, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	t := model.Task{ID: s.nextID, Title: title, CreatedAt: time.Now()}
	s.tasks[t.ID] = t
	s.nextID++
	return t, nil
}

func (s *MemoryStore) List(_ context.Context) ([]model.Task, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]model.Task, 0, len(s.tasks))
	for _, t := range s.tasks {
		out = append(out, t)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}

func (s *MemoryStore) Get(_ context.Context, id int) (model.Task, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	t, ok := s.tasks[id]
	if !ok {
		return model.Task{}, ErrNotFound
	}
	return t, nil
}

func (s *MemoryStore) Toggle(_ context.Context, id int) (model.Task, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	t, ok := s.tasks[id]
	if !ok {
		return model.Task{}, ErrNotFound
	}
	t.Done = !t.Done
	s.tasks[id] = t
	return t, nil
}

func (s *MemoryStore) Delete(_ context.Context, id int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.tasks[id]; !ok {
		return ErrNotFound
	}
	delete(s.tasks, id)
	return nil
}
