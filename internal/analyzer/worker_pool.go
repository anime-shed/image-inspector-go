package analyzer

import (
	"runtime"
	"sync"
)

// WorkerPool manages concurrent image processing tasks
type WorkerPool struct {
	workers    int
	jobQueue   chan func()
	wg         sync.WaitGroup
	once       sync.Once
	workerPool sync.Pool
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
		job()
	}
}

// Submit adds a job to the worker pool queue
func (wp *WorkerPool) Submit(job func()) {
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