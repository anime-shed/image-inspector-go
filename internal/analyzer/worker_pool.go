package analyzer

import (
	"runtime"
	"sync"
)

// WorkerPool manages concurrent task execution
type WorkerPool struct {
	workers    int
	jobQueue   chan func()
	wg         sync.WaitGroup
	once       sync.Once
	workerPool sync.Pool
	mu         sync.RWMutex
	closed     bool
}

// NewWorkerPool creates a new worker pool with the specified number of workers
func NewWorkerPool(workers int) *WorkerPool {
	if workers <= 0 {
		workers = runtime.NumCPU()
	}

	return &WorkerPool{
		workers:  workers,
		jobQueue: make(chan func(), workers*2),
		workerPool: sync.Pool{
			New: func() interface{} {
				return make([]byte, 0, 1024)
			},
		},
	}
}

// Start initializes and starts all workers in the pool
func (wp *WorkerPool) Start() {
	wp.once.Do(func() {
		for i := 0; i < wp.workers; i++ {
			go wp.worker()
		}
	})
}

// worker processes jobs from the job queue
func (wp *WorkerPool) worker() {
	for job := range wp.jobQueue {
		func() {
			defer wp.wg.Done()
			defer func() {
				if recover() != nil {
					// optionally log/report the panic here
				}
			}()
			job()
		}()
	}
}

// Submit adds a job to the worker pool
func (wp *WorkerPool) Submit(job func()) {
	// Auto-start is idempotent due to once.Do inside Start()
	wp.Start()
	
	wp.mu.RLock()
	if wp.closed {
		wp.mu.RUnlock()
		return // No-op if pool is closed
	}
	wp.wg.Add(1)
	wp.mu.RUnlock()
	
	wp.jobQueue <- job
}

// Wait waits for all submitted jobs to complete
func (wp *WorkerPool) Wait() {
	wp.wg.Wait()
}

// Close shuts down the worker pool
func (wp *WorkerPool) Close() {
	wp.mu.Lock()
	defer wp.mu.Unlock()
	
	if wp.closed {
		return // Already closed, idempotent
	}
	
	wp.closed = true
	close(wp.jobQueue)
}
