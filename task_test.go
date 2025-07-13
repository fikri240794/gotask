package gotask

import (
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// TestNewTask tests the Task constructor with various input values
func TestNewTask(t *testing.T) {
	tests := []struct {
		name                   string
		maxConcurrentTask      int
		expectedMaxConcurrency int
	}{
		{
			name:                   "zero concurrent tasks defaults to 2x CPU cores",
			maxConcurrentTask:      0,
			expectedMaxConcurrency: runtime.NumCPU() * 2,
		},
		{
			name:                   "negative concurrent tasks defaults to 2x CPU cores",
			maxConcurrentTask:      -1,
			expectedMaxConcurrency: runtime.NumCPU() * 2,
		},
		{
			name:                   "positive concurrent tasks uses provided value",
			maxConcurrentTask:      4,
			expectedMaxConcurrency: 4,
		},
		{
			name:                   "single concurrent task",
			maxConcurrentTask:      1,
			expectedMaxConcurrency: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			task := NewTask(tt.maxConcurrentTask).(*task)

			if cap(task.semaphore) != tt.expectedMaxConcurrency {
				t.Errorf("expected semaphore capacity %d, got %d", tt.expectedMaxConcurrency, cap(task.semaphore))
			}

			if task.maxConcurrency != tt.expectedMaxConcurrency {
				t.Errorf("expected maxConcurrency %d, got %d", tt.expectedMaxConcurrency, task.maxConcurrency)
			}
		})
	}
}

// TestTask_ConcurrencyLimit tests that the task respects the maximum concurrency limit
func TestTask_ConcurrencyLimit(t *testing.T) {
	const maxConcurrency = 2
	const numTasks = 10

	task := NewTask(maxConcurrency)

	var (
		currentConcurrency     int64
		maxObservedConcurrency int64
		completedTasks         int64
	)

	// Start tasks that will run concurrently
	for i := 0; i < numTasks; i++ {
		task.Go(func() {
			// Track current concurrency
			current := atomic.AddInt64(&currentConcurrency, 1)

			// Update max observed concurrency
			for {
				max := atomic.LoadInt64(&maxObservedConcurrency)
				if current <= max || atomic.CompareAndSwapInt64(&maxObservedConcurrency, max, current) {
					break
				}
			}

			// Simulate some work
			time.Sleep(50 * time.Millisecond)

			// Decrement current concurrency
			atomic.AddInt64(&currentConcurrency, -1)
			atomic.AddInt64(&completedTasks, 1)
		})
	}

	task.Wait()

	// Verify all tasks completed
	if completed := atomic.LoadInt64(&completedTasks); completed != numTasks {
		t.Errorf("expected %d completed tasks, got %d", numTasks, completed)
	}

	// Verify concurrency limit was respected
	if maxObserved := atomic.LoadInt64(&maxObservedConcurrency); maxObserved > maxConcurrency {
		t.Errorf("concurrency limit exceeded: max observed %d, limit %d", maxObserved, maxConcurrency)
	}
}

// TestTask_ActiveCountAndTotalSubmitted tests the monitoring methods
func TestTask_ActiveCountAndTotalSubmitted(t *testing.T) {
	const maxConcurrency = 2
	const numTasks = 5

	task := NewTask(maxConcurrency)

	// Initially, counters should be zero
	if active := task.ActiveCount(); active != 0 {
		t.Errorf("expected 0 active tasks initially, got %d", active)
	}
	if total := task.TotalSubmitted(); total != 0 {
		t.Errorf("expected 0 total submitted initially, got %d", total)
	}

	// Use a simpler approach: submit tasks without blocking
	var tasksStarted int64
	var tasksFinished int64

	// Submit tasks that increment counters
	for i := 0; i < numTasks; i++ {
		task.Go(func() {
			atomic.AddInt64(&tasksStarted, 1)

			// Simulate some work
			time.Sleep(50 * time.Millisecond)

			atomic.AddInt64(&tasksFinished, 1)
		})
	}

	// Give tasks a moment to start
	time.Sleep(100 * time.Millisecond)

	// Check that total submitted is correct (all tasks should be submitted immediately)
	if total := task.TotalSubmitted(); total != numTasks {
		t.Errorf("expected %d total submitted, got %d", numTasks, total)
	}

	// Check active count (should be limited by maxConcurrency)
	active := task.ActiveCount()
	if active > maxConcurrency {
		t.Errorf("active tasks exceeded concurrency limit: got %d, max %d", active, maxConcurrency)
	}

	// Wait for all tasks to complete
	task.Wait()

	// After completion, all tasks should have finished
	if finished := atomic.LoadInt64(&tasksFinished); finished != numTasks {
		t.Errorf("expected %d finished tasks, got %d", numTasks, finished)
	}

	// After completion, active should be 0, total should remain
	if active := task.ActiveCount(); active != 0 {
		t.Errorf("expected 0 active tasks after Wait(), got %d", active)
	}
	if total := task.TotalSubmitted(); total != numTasks {
		t.Errorf("expected %d total submitted after Wait(), got %d", numTasks, total)
	}
}

// TestTask_EmptyWorkload tests behavior with no tasks submitted
func TestTask_EmptyWorkload(t *testing.T) {
	task := NewTask(2)

	// Wait should return immediately with no tasks
	task.Wait()

	// Counters should remain zero
	if active := task.ActiveCount(); active != 0 {
		t.Errorf("expected 0 active tasks, got %d", active)
	}
	if total := task.TotalSubmitted(); total != 0 {
		t.Errorf("expected 0 total submitted, got %d", total)
	}
}

// TestTask_SequentialExecution tests single-threaded execution
func TestTask_SequentialExecution(t *testing.T) {
	task := NewTask(1) // Only 1 concurrent task allowed

	var executionOrder []int
	var mu sync.Mutex

	// Submit tasks that should execute sequentially
	for i := 0; i < 3; i++ {
		taskNum := i
		task.Go(func() {
			mu.Lock()
			executionOrder = append(executionOrder, taskNum)
			mu.Unlock()

			// Small delay to ensure ordering
			time.Sleep(10 * time.Millisecond)
		})
	}

	task.Wait()

	// With concurrency=1, tasks should execute in order
	if len(executionOrder) != 3 {
		t.Errorf("expected 3 tasks executed, got %d", len(executionOrder))
	}

	// Verify all task numbers are present (exact order may vary due to scheduling)
	taskMap := make(map[int]bool)
	for _, taskNum := range executionOrder {
		taskMap[taskNum] = true
	}

	for i := 0; i < 3; i++ {
		if !taskMap[i] {
			t.Errorf("task %d was not executed", i)
		}
	}
}

// BenchmarkTask_Throughput benchmarks task submission and execution throughput
func BenchmarkTask_Throughput(b *testing.B) {
	benchmarks := []struct {
		name        string
		concurrency int
	}{
		{"Concurrency1", 1},
		{"Concurrency2", 2},
		{"Concurrency4", 4},
		{"Concurrency8", 8},
		{"ConcurrencyDefault", 0}, // Will use 2x CPU cores
	}

	for _, bm := range benchmarks {
		b.Run(bm.name, func(b *testing.B) {
			task := NewTask(bm.concurrency)

			b.ResetTimer()
			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					task.Go(func() {
						// Simulate minimal work
						runtime.Gosched()
					})
				}
			})
			task.Wait()
		})
	}
}

// BenchmarkTask_CPUBound benchmarks CPU-intensive tasks
func BenchmarkTask_CPUBound(b *testing.B) {
	task := NewTask(runtime.NumCPU())

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		task.Go(func() {
			// Simulate CPU-bound work
			sum := 0
			for j := 0; j < 1000; j++ {
				sum += j * j
			}
			_ = sum
		})
	}
	task.Wait()
}

// BenchmarkTask_IOBound benchmarks I/O-intensive tasks
func BenchmarkTask_IOBound(b *testing.B) {
	task := NewTask(runtime.NumCPU() * 4) // Higher concurrency for I/O

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		task.Go(func() {
			// Simulate I/O-bound work
			time.Sleep(time.Microsecond)
		})
	}
	task.Wait()
}
