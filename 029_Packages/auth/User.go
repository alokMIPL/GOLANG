package tasks

import (
	"errors"
	"fmt"
	"time"
)

// Status represents the state of a task
type Status int

const (
	Pending Status = iota
	Running
	Done
	Failed
)

func (s Status) String() string {
	switch s {
	case Pending:
		return "Pending"
	case Running:
		return "Running"
	case Done:
		return "Done"
	case Failed:
		return "Failed"
	default:
		return "Unknown"
	}
}

// Task represents a single unit of work
type Task struct {
	ID        int
	Name      string
	Status    Status
	CreatedAt time.Time
	Err       error
}

// ErrInvalidTask is returned when a task fails validation
var ErrInvalidTask = errors.New("task name cannot be empty")

// NewTask creates a validated Task
func NewTask(id int, name string) (*Task, error) {
	if name == "" {
		return nil, ErrInvalidTask
	}
	return &Task{
		ID:        id,
		Name:      name,
		Status:    Pending,
		CreatedAt: time.Now(),
	}, nil
}

func (t *Task) String() string {
	return fmt.Sprintf("[Task #%d] %s - %s", t.ID, t.Name, t.Status)
}
