package concurrency

import (
	"context"
	"fmt"
	"log/slog"
	"runtime"
	"sync"
)

// WorkerFunc is the unit of work processed by the pool.
type WorkerFunc func(ctx context.Context) error

// Pool is a bounded worker pool with graceful shutdown.
type Pool struct {
	workers   int
	semaphore chan struct{}
	wg        sync.WaitGroup
	errOnce   sync.Once
	firstErr  error
}

// NewPool creates a worker pool. If workers <= 0 it defaults to min(cpu*2, 100).
func NewPool(workers int) *Pool {
	if workers <= 0 {
		workers = runtime.NumCPU() * 2
		if workers > 100 {
			workers = 100
		}
	}
	return &Pool{
		workers:   workers,
		semaphore: make(chan struct{}, workers),
	}
}

// Submit enqueues a task. It blocks until a worker is available or ctx is done.
func (p *Pool) Submit(ctx context.Context, fn WorkerFunc) error {
	select {
	case p.semaphore <- struct{}{}:
		p.wg.Add(1)
		go func() {
			defer func() {
				<-p.semaphore
				p.wg.Done()
			}()
			if err := fn(ctx); err != nil {
				p.errOnce.Do(func() { p.firstErr = err })
				slog.Error("worker error", "error", err)
			}
		}()
		return nil
	case <-ctx.Done():
		return fmt.Errorf("pool submit: %w", ctx.Err())
	}
}

// Wait blocks until all submitted tasks complete.
func (p *Pool) Wait() {
	p.wg.Wait()
}

// Err returns the first error encountered by any worker.
func (p *Pool) Err() error {
	return p.firstErr
}

// Size returns the configured number of workers.
func (p *Pool) Size() int {
	return p.workers
}
