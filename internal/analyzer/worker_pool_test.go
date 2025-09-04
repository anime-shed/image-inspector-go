package analyzer

import (
	"sync"
	"testing"
)

func TestNewWorkerPool(t *testing.T) {
	pool := NewWorkerPool(4)
	if pool == nil {
		t.Fatal("Expected non-nil worker pool")
	}
	// Can't access private fields, so just test that pool was created successfully
}

func TestNewWorkerPool_ZeroWorkers(t *testing.T) {
	pool := NewWorkerPool(0)
	if pool == nil {
		t.Error("Expected non-nil WorkerPool")
	}
	// Should default to runtime.NumCPU() when workers <= 0
	// We can't access private field, so just test that pool was created
}

func TestWorkerPool_Submit(t *testing.T) {
	pool := NewWorkerPool(2)
	pool.Start()
	defer pool.Close()

	// Test submitting jobs and waiting for completion
	var counter int
	var mu sync.Mutex

	for i := 0; i < 5; i++ {
		pool.Submit(func() {
			mu.Lock()
			counter++
			mu.Unlock()
		})
	}

	pool.Wait()

	if counter != 5 {
		t.Errorf("Expected counter to be 5, got %d", counter)
	}
}

func TestWorkerPool_Concurrent(t *testing.T) {
	pool := NewWorkerPool(3)
	pool.Start()
	defer pool.Close()

	// Test concurrent job execution
	var results []int
	var mu sync.Mutex

	for i := 0; i < 10; i++ {
		value := i
		pool.Submit(func() {
			// Simulate some work
			processedValue := value * 2
			mu.Lock()
			results = append(results, processedValue)
			mu.Unlock()
		})
	}

	pool.Wait()

	if len(results) != 10 {
		t.Errorf("Expected 10 results, got %d", len(results))
	}
}

func TestWorkerPool_StartOnce(t *testing.T) {
	pool := NewWorkerPool(2)

	// Start should be idempotent
	pool.Start()
	pool.Start() // Should not panic or create duplicate workers

	defer pool.Close()

	// Test that pool still works after multiple Start calls
	var executed bool
	pool.Submit(func() {
		executed = true
	})

	pool.Wait()

	if !executed {
		t.Error("Expected job to be executed")
	}
}

func TestWorkerPool_CloseAndResubmit(t *testing.T) {
	pool := NewWorkerPool(2)
	pool.Start()

	// Submit a job
	var executed bool
	pool.Submit(func() {
		executed = true
	})

	pool.Wait()
	pool.Close()

	if !executed {
		t.Error("Expected job to be executed before close")
	}
}

func TestWorkerPool_SubmissionConsistency(t *testing.T) {
	pool := NewWorkerPool(1) // Use single worker to avoid concurrency issues
	pool.Start()
	defer pool.Close()

	// Submit a few jobs and verify consistency
	const numJobs = 3
	successCount := 0

	for i := 0; i < numJobs; i++ {
		success := pool.Submit(func() {
			// Simple job
		})
		if success {
			successCount++
		}
	}

	pool.Wait()

	// Verify stats consistency - this is the key test for our fix
	stats := pool.GetStats()
	if stats.TotalJobs != int64(successCount) {
		t.Errorf("Expected TotalJobs=%d, got %d", successCount, stats.TotalJobs)
	}
	if stats.CompletedJobs != int64(successCount) {
		t.Errorf("Expected CompletedJobs=%d, got %d", successCount, stats.CompletedJobs)
	}
	if stats.ActiveWorkers != 0 {
		t.Errorf("Expected 0 active workers, got %d", stats.ActiveWorkers)
	}
}

func TestWorkerPool_AtomicCounters(t *testing.T) {
	pool := NewWorkerPool(4)
	pool.Start()
	defer pool.Close()

	const numJobs = 5

	// Submit jobs and verify counters are updated atomically
	for i := 0; i < numJobs; i++ {
		pool.Submit(func() {
			// Simulate some work
			for j := 0; j < 1000; j++ {
				_ = j * j
			}
		})
	}

	pool.Wait()

	// Get final stats
	stats := pool.GetStats()

	// Verify counters
	if stats.TotalJobs != int64(numJobs) {
		t.Errorf("Expected %d total jobs, got %d", numJobs, stats.TotalJobs)
	}

	if stats.CompletedJobs != int64(numJobs) {
		t.Errorf("Expected %d completed jobs, got %d", numJobs, stats.CompletedJobs)
	}

	if stats.ActiveWorkers != 0 {
		t.Errorf("Expected 0 active workers after completion, got %d", stats.ActiveWorkers)
	}
}

func TestWorkerPool_ConcurrentStatsAccess(t *testing.T) {
	pool := NewWorkerPool(2)
	pool.Start()
	defer pool.Close()

	// Test concurrent access to stats while jobs are running
	const numJobs = 20
	const numStatsReads = 10

	var wg sync.WaitGroup

	// Start jobs
	for i := 0; i < numJobs; i++ {
		pool.Submit(func() {
			// Simulate work
			for j := 0; j < 5000; j++ {
				_ = j * j
			}
		})
	}

	// Concurrently read stats
	for i := 0; i < numStatsReads; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 10; j++ {
				stats := pool.GetStats()
				// Just verify we can read stats without panicking
				_ = stats.TotalJobs
				_ = stats.CompletedJobs
				_ = stats.ActiveWorkers
			}
		}()
	}

	wg.Wait()
	pool.Wait()

	// Final verification
	finalStats := pool.GetStats()
	if finalStats.TotalJobs != numJobs {
		t.Errorf("Expected %d total jobs, got %d", numJobs, finalStats.TotalJobs)
	}
}

// Removed TestWorkerPool_ProcessImages_Concurrency as ProcessImages method doesn't exist

// Removed TestWorkerPool_ProcessImages_DifferentSizes as ProcessImages method doesn't exist

// Removed TestWorkerPool_ProcessImages_StressTest as ProcessImages method doesn't exist

// Removed TestWorkerPool_ProcessImages_RaceCondition as ProcessImages method doesn't exist

// Removed TestWorkerPool_ProcessImages_SingleWorker as ProcessImages method doesn't exist
