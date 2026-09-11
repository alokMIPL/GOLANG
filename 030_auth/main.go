package auth
package tasks

import (
    "errors"
    "fmt"
    "time"
)

type Status int

const (
    Pending Status = iota
    Running
    Done
    Failed
    Cancelled
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
    case Cancelled:
        return "Cancelled"
    default:
        return "Unknown"
    }
}

// MarshalJSON makes Status serialize as a string instead of a number
func (s Status) MarshalJSON() ([]byte, error) {
    return []byte(`"` + s.String() + `"`), nil
}

type Task struct {
    ID        int       `json:"id"`
    Name      string    `json:"name"`
    Status    Status    `json:"status"`
    CreatedAt time.Time `json:"created_at"`
    ErrMsg    string    `json:"error,omitempty"`
}

var ErrInvalidTask = errors.New("task name cannot be empty")

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