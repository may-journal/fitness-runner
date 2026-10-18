// Package par is the tiny deterministic worker pool the file-scanning
// checks share: results come back in input order regardless of completion
// order, so parallelism never changes a check's output.
package par

import (
	"context"
	"runtime"
	"sync"
)

// Map runs fn over the indices 0..n-1 on a bounded worker pool and returns
// the results in index order. Zero or negative workers means NumCPU.
func Map[T any](n, workers int, fn func(i int) T) []T {
	workers = clampWorkers(workers, n)
	out := make([]T, n)
	if n == 0 {
		return out
	}
	next := make(chan int)
	var wg sync.WaitGroup
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go worker(&wg, next, out, fn)
	}
	for i := 0; i < n; i++ {
		next <- i
	}
	close(next)
	wg.Wait()
	return out
}

// MapErr runs fn over 0..n-1 like Map, each call under its own context from
// ctx. When call i fails, later calls are cancelled or skipped while earlier
// ones finish, so the returned error is always the lowest-index one.
func MapErr[T any](ctx context.Context, n, workers int, fn func(ctx context.Context, i int) (T, error)) ([]T, error) {
	s := &stopper{parent: ctx, failed: n, cancels: make([]context.CancelFunc, n)}
	errs := make([]error, n)
	out := Map(n, workers, func(i int) T {
		var zero T
		c, ok := s.start(i)
		if !ok {
			return zero
		}
		v, err := fn(c, i)
		s.finish(i, err)
		errs[i] = err
		return v
	})
	for _, err := range errs {
		if err != nil {
			return nil, err
		}
	}
	return out, nil
}

// stopper hands each MapErr call a context and cancels the calls after the
// lowest-index failure.
type stopper struct {
	mu      sync.Mutex
	parent  context.Context
	failed  int
	cancels []context.CancelFunc
}

// start returns call i's context, or false when an earlier call failed.
func (s *stopper) start(i int) (context.Context, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if i > s.failed {
		return nil, false
	}
	ctx, cancel := context.WithCancel(s.parent)
	s.cancels[i] = cancel
	return ctx, true
}

// finish releases call i's context and, on its failure, cancels later calls.
func (s *stopper) finish(i int, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cancels[i]()
	if err == nil || i >= s.failed {
		return
	}
	s.failed = i
	for _, cancel := range s.cancels[i+1:] {
		if cancel != nil {
			cancel()
		}
	}
}

// Processes is the pool size for work that starts child processes: the
// runner's handshakes and check pool and the coverage audit's test runs all
// use it. Each child is its own process, so every core takes one.
func Processes() int { return runtime.NumCPU() }

// clampWorkers bounds the pool: NumCPU by default, never more than items.
func clampWorkers(workers, n int) int {
	if workers <= 0 {
		workers = runtime.NumCPU()
	}
	if workers > n {
		return n
	}
	return workers
}

// worker drains indices from next, storing each result at its own index.
func worker[T any](wg *sync.WaitGroup, next <-chan int, out []T, fn func(i int) T) {
	defer wg.Done()
	for i := range next {
		out[i] = fn(i)
	}
}
