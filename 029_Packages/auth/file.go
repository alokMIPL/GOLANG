package tasks

import "sync"

// Manager coordinates concurrent task execution using a worker pool
type Manager struct {
	mu    sync.Mutex
	tasks []*Task
}

// NewManager creates an empty Manager
func NewManager() *Manager {
	return &Manager{tasks: make([]*Task, 0)}
}

// Add appends a task to the manager
func (m *Manager) Add(t *Task) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.tasks = append(m.tasks, t)
}

// RunAll processes all tasks concurrently using a worker pool
func (m *Manager) RunAll(numWorkers int) []*Task {
	jobs := make(chan *Task, len(m.tasks))
	results := make(chan *Task, len(m.tasks))

	var wg sync.WaitGroup
	for w := 1; w <= numWorkers; w++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			Worker(id, jobs, results)
		}(w)
	}

	for _, t := range m.tasks {
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

// Summary returns counts of tasks by status
func (m *Manager) Summary() map[Status]int {
	m.mu.Lock()
	defer m.mu.Unlock()

	summary := make(map[Status]int)
	for _, t := range m.tasks {
		summary[t.Status]++
	}
	return summary
}
