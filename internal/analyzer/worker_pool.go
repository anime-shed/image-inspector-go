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
	mu         sync.Mutex
	started    bool
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
		wp.mu.Lock()
		wp.started = true
		wp.mu.Unlock()
		for i := 0; i < wp.workers; i++ {
			go wp.worker()
		}
	})
}

// worker processes jobs from the job queue
func (wp *WorkerPool) worker() {
	for job := range wp.jobQueue {
		job()
		wp.wg.Done()
	}
}

// Submit adds a job to the worker pool queue
// Auto-starts the pool if not already started to prevent blocking
func (wp *WorkerPool) Submit(job func()) {
	wp.mu.Lock()
	if !wp.started {
		wp.started = true
		wp.mu.Unlock()
		wp.Start()
	} else {
		wp.mu.Unlock()
	}
	wp.wg.Add(1)
	wp.jobQueue <- job
}

// Wait waits for all submitted jobs to complete
func (wp *WorkerPool) Wait() {
	wp.wg.Wait()
}

// Close shuts down the worker pool
func (wp *WorkerPool) Close() {
	close(wp.jobQueue)
}
