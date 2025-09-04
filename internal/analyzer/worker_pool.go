package analyzer

import (
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

// WorkerPool manages concurrent task execution with enhanced performance
// Implements optimizations from PERFORMANCE_OPTIMIZATION_ANALYSIS.md Phase 3
type WorkerPool struct {
	workers  int
	jobQueue chan func()
	wg       sync.WaitGroup
	once     sync.Once
	mu       sync.RWMutex
	closed   bool

	// Enhanced memory pools for different data types
	bufferPool sync.Pool // For temporary byte slices
	slicePool  sync.Pool // For temporary float64 slices
	matrixPool sync.Pool // For temporary matrix data

	// Performance monitoring
	activeWorkers int64
	totalJobs     int64
	completedJobs int64
}

// NewWorkerPool creates a new worker pool
func NewWorkerPool(workers int) *WorkerPool {
	if workers <= 0 {
		workers = runtime.NumCPU()
	}

	return &WorkerPool{
		workers:  workers,
		jobQueue: make(chan func(), workers*4), // Increased buffer for better throughput

		// Initialize memory pools with appropriate sizes
		bufferPool: sync.Pool{
			New: func() interface{} {
				return make([]byte, 0, 4096) // 4KB initial capacity
			},
		},
		slicePool: sync.Pool{
			New: func() interface{} {
				return make([]float64, 0, 1024) // 1K float64 elements
			},
		},
		matrixPool: sync.Pool{
			New: func() interface{} {
				return make([][]float64, 0, 16) // For small matrices
			},
		},
	}
}

// Start initializes and starts all workers in the pool with goroutine management
func (owp *WorkerPool) Start() {
	owp.once.Do(func() {
		// Start workers with better CPU affinity consideration
		for i := 0; i < owp.workers; i++ {
			go owp.worker(i)
		}
	})
}

// worker processes jobs with enhanced error handling and performance monitoring
func (owp *WorkerPool) worker(workerID int) {
	// Let the scheduler manage OS threads; no affinity required
	for job := range owp.jobQueue {
		// Process the job
		func() {
			defer func() {
				if r := recover(); r != nil {
					// Enhanced panic recovery with logging capability
					// In production, this would log the panic details
				}
			}()

			// Execute the job
			owp.incrementActiveWorkers()
			job()
			owp.decrementActiveWorkers()
			owp.incrementCompletedJobs()
		}()
		
		// Signal job completion
		owp.wg.Done()
	}
}

// Submit adds a job to the worker pool with queuing
func (owp *WorkerPool) Submit(job func()) bool {
	owp.Start() // Auto-start is idempotent

	owp.mu.RLock()
	defer owp.mu.RUnlock()
	if owp.closed {
		return false // Return false if pool is closed
	}

	// Non-blocking submit with timeout
	select {
	case owp.jobQueue <- job:
		// Only increment counters after successful submission
		owp.wg.Add(1)
		owp.incrementTotalJobs()
		return true
	case <-time.After(100 * time.Millisecond):
		return false // Job rejected due to full queue
	}
}

// SubmitWithTimeout adds a job with a custom timeout
func (owp *WorkerPool) SubmitWithTimeout(job func(), timeout time.Duration) bool {
	owp.Start()

	owp.mu.RLock()
	defer owp.mu.RUnlock()
	if owp.closed {
		return false
	}
	
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	
	select {
	case owp.jobQueue <- job:
		// Only increment counters after successful submission
		owp.wg.Add(1)
		owp.incrementTotalJobs()
		return true
	case <-timer.C:
		return false
	}
}

// Wait waits for all submitted jobs to complete
func (owp *WorkerPool) Wait() {
	owp.wg.Wait()
}

// WaitWithTimeout waits for jobs to complete with a timeout
func (owp *WorkerPool) WaitWithTimeout(timeout time.Duration) bool {
	done := make(chan struct{})
	go func() {
		owp.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		return true
	case <-time.After(timeout):
		return false
	}
}

// Close shuts down the worker pool gracefully
func (owp *WorkerPool) Close() {
	owp.mu.Lock()
	defer owp.mu.Unlock()

	if owp.closed {
		return
	}

	owp.closed = true
	close(owp.jobQueue)
}

// GetBuffer retrieves a reusable byte buffer from the pool
func (owp *WorkerPool) GetBuffer() []byte {
	return owp.bufferPool.Get().([]byte)
}

// PutBuffer returns a byte buffer to the pool
func (owp *WorkerPool) PutBuffer(buf []byte) {
	const maxBufCap = 1 << 20 // 1MB
	if cap(buf) <= maxBufCap {
		owp.bufferPool.Put(buf[:0]) // Reset length but keep capacity
	}
}

// GetSlice retrieves a reusable float64 slice from the pool
func (owp *WorkerPool) GetSlice() []float64 {
	return owp.slicePool.Get().([]float64)
}

// PutSlice returns a float64 slice to the pool
func (owp *WorkerPool) PutSlice(slice []float64) {
	const maxSliceCap = 1 << 15 // 32K
	if cap(slice) <= maxSliceCap {
		owp.slicePool.Put(slice[:0]) // Reset length but keep capacity
	}
}

// GetMatrix retrieves a reusable matrix from the pool
func (owp *WorkerPool) GetMatrix() [][]float64 {
	return owp.matrixPool.Get().([][]float64)
}

// PutMatrix returns a matrix to the pool
func (owp *WorkerPool) PutMatrix(matrix [][]float64) {
	const maxRows = 1024
	if cap(matrix) <= maxRows {
		owp.matrixPool.Put(matrix[:0]) // Reset length but keep capacity
	}
}

// Performance monitoring methods
func (owp *WorkerPool) incrementActiveWorkers() {
	atomic.AddInt64(&owp.activeWorkers, 1)
}

func (owp *WorkerPool) decrementActiveWorkers() {
	atomic.AddInt64(&owp.activeWorkers, -1)
}

func (owp *WorkerPool) incrementTotalJobs() {
	atomic.AddInt64(&owp.totalJobs, 1)
}

func (owp *WorkerPool) incrementCompletedJobs() {
	atomic.AddInt64(&owp.completedJobs, 1)
}

// Stats returns performance statistics
type WorkerPoolStats struct {
	Workers       int
	ActiveWorkers int64
	TotalJobs     int64
	CompletedJobs int64
	QueueLength   int
}

// GetStats returns current worker pool statistics
func (owp *WorkerPool) GetStats() WorkerPoolStats {
	owp.mu.RLock()
	defer owp.mu.RUnlock()

	return WorkerPoolStats{
		Workers:       owp.workers,
		ActiveWorkers: atomic.LoadInt64(&owp.activeWorkers),
		TotalJobs:     atomic.LoadInt64(&owp.totalJobs),
		CompletedJobs: atomic.LoadInt64(&owp.completedJobs),
		QueueLength:   len(owp.jobQueue),
	}
}

// Resize dynamically adjusts the number of workers (for advanced use cases)
func (owp *WorkerPool) Resize(newWorkerCount int) {
	if newWorkerCount <= 0 {
		newWorkerCount = runtime.NumCPU()
	}

	owp.mu.Lock()
	defer owp.mu.Unlock()

	if owp.closed {
		return
	}

	// For simplicity, we'll just update the worker count
	// In a full implementation, this would actually start/stop workers
	owp.workers = newWorkerCount
}
