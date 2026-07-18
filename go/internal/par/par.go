// Package par is the tiny deterministic worker pool the file-scanning
// checks share: results come back in input order regardless of completion
// order, so parallelism never changes a check's output.
package par

import (
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
