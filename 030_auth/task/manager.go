package task
package tasks

import (
    "context"
    "sync"
)

type Manager struct {
    mu    sync.Mutex
    tasks []*Task
}

func NewManager() *Manager {
    return &Manager{tasks: make([]*Task, 0)}
}

func (m *Manager) Add(t *Task) {
    m.mu.Lock()
    defer m.mu.Unlock()
    m.tasks = append(m.tasks, t)
}

func (m *Manager) All() []*Task {
    m.mu.Lock()
    defer m.mu.Unlock()
    out := make([]*Task, len(m.tasks))
    copy(out, m.tasks)
    return out
}

// RunAll runs tasks concurrently, respecting context cancellation/timeout
func (m *Manager) RunAll(ctx context.Context, numWorkers int) []*Task {
    m.mu.Lock()
    taskList := make([]*Task, len(m.tasks))
    copy(taskList, m.tasks)
    m.mu.Unlock()
		

    jobs := make(chan *Task, len(taskList))
    results := make(chan *Task, len(taskList))

    var wg sync.WaitGroup
    for w := 1; w <= numWorkers; w++ {
        wg.Add(1)
        go func(id int) {
            defer wg.Done()
            Worker(ctx, id, jobs, results)
        }(w)
    }

    for _, t := range taskList {
        jobs <- t
    }
    close(jobs)

    go func() {
        wg.Wait()
        close(results)
    }()

    var completed []*Task
    for r := range results {
        completed = append(completed, r)
    }
    return completed
}

func (m *Manager) Summary() map[Status]int {
    m.mu.Lock()
    defer m.mu.Unlock()
    summary := make(map[Status]int)
    for _, t := range m.tasks {
        summary[t.Status]++
    }
    return summary
}
