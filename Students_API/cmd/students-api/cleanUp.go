package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"sync"
	"time"
)

type cleanupFunc struct {
	name    string
	fn      func(context.Context) error
	timeout time.Duration // 0 means no per-function timeout
}

// cleanupStack collects teardown functions and runs them in reverse order
// (last added, first run), like a chain of defers that can be built up
// dynamically during startup.
type cleanupStack struct {
	mu    sync.Mutex
	funcs []cleanupFunc
	log   *slog.Logger
}

// newCleanupStack returns an empty stack. log may be nil.
func newCleanupStack(log *slog.Logger) *cleanupStack {
	return &cleanupStack{log: log}
}

// logger returns a usable logger even when none was configured, so the rest
// of the code does not need nil checks.
func (c *cleanupStack) logger() *slog.Logger {
	if c.log != nil {
		return c.log
	}
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

// Add registers fn to run during Run with no per-function timeout.
func (c *cleanupStack) Add(name string, fn func(context.Context) error) {
	c.AddWithTimeout(name, 0, fn)
}

func (c *cleanupStack) AddWithTimeout(name string, timeout time.Duration, fn func(context.Context) error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.funcs = append(c.funcs, cleanupFunc{name: name, fn: fn, timeout: timeout})
}

func (c *cleanupStack) AddCloser(name string, closer io.Closer) {
	c.Add(name, func(context.Context) error { return closer.Close() })
}

func (c *cleanupStack) Len() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.funcs)
}

func (c *cleanupStack) Run(ctx context.Context) error {
	c.mu.Lock()
	funcs := c.funcs
	c.funcs = nil
	c.mu.Unlock()

	log := c.logger()
	log.Debug("running cleanups", "count", len(funcs))

	var errs []error
	for i := len(funcs) - 1; i >= 0; i-- {
		f := funcs[i]
		start := time.Now()

		err := c.runOne(ctx, f)
		elapsed := time.Since(start).String()

		if err != nil {
			errs = append(errs, fmt.Errorf("cleanup %s: %w", f.name, err))
			log.Error("cleanup failed", "name", f.name, "elapsed", elapsed, "err", err)
			continue
		}
		log.Debug("cleanup done", "name", f.name, "elapsed", elapsed)
	}
	return errors.Join(errs...)
}

// runOne runs a single cleanup, applying its timeout and converting a panic
// into an error so one bad cleanup cannot prevent the rest from running.
func (c *cleanupStack) runOne(ctx context.Context, f cleanupFunc) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic: %v", r)
		}
	}()

	if f.timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, f.timeout)
		defer cancel()
	}
	return f.fn(ctx)
}
