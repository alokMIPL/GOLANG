package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
)

type cleanupFunc struct {
	name string
	fn   func(context.Context) error
}

type cleanupStack struct {
	mu    sync.Mutex
	funcs []cleanupFunc
	log   *slog.Logger
}

func (c *cleanupStack) Add(name string, fn func(context.Context) error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.funcs = append(c.funcs, cleanupFunc{name: name, fn: fn})
}

func (c *cleanupStack) Run(ctx context.Context) error {
	c.mu.Lock()
	funcs := c.funcs
	c.funcs = nil
	c.mu.Unlock()

	var errs []error
	for i := len(funcs) - 1; i >= 0; i-- {
		f := funcs[i]
		if err := f.fn(ctx); err != nil {
			errs = append(errs, fmt.Errorf("cleanup %s: %w", f.name, err))
			if c.log != nil {
				c.log.Error("cleanup failed", "name", f.name, "err", err)
			}
			continue
		}
		if c.log != nil {
			c.log.Debug("cleanup done", "name", f.name)
		}
	}
	return errors.Join(errs...)
}
