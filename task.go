package gotask

import (
	"runtime"
	"sync"
	"sync/atomic"
)

// Task provides a simple interface for managing concurrent goroutines with a maximum concurrency limit.
// It uses a semaphore pattern to control the number of simultaneously running goroutines.
type Task interface {
	// Go executes the given function in a goroutine, respecting the concurrency limit.
	// If the maximum concurrent tasks limit is reached, this method will block until a slot becomes available.
	Go(task func())

	// Wait blocks until all submitted tasks have completed.
	Wait()

	// ActiveCount returns the current number of active/running tasks.
	ActiveCount() int64

	// TotalSubmitted returns the total number of tasks that have been submitted.
	TotalSubmitted() int64
}

// task implements the Task interface using a semaphore pattern with channels and sync.WaitGroup.
type task struct {
	wg             sync.WaitGroup // synchronizes completion of all tasks
	semaphore      chan struct{}  // controls maximum concurrent executions
	activeCount    int64          // atomic counter for active tasks
	submittedCount int64          // atomic counter for total submitted tasks
	maxConcurrency int            // maximum number of concurrent tasks allowed
}

// NewTask creates a new Task instance with the specified maximum concurrent task limit.
//
// Performance considerations for maxConcurrentTask:
//   - For I/O-bound tasks: can be 2-10x CPU cores depending on I/O wait time
//   - For CPU-bound tasks: should typically equal CPU cores or slightly less
//   - For mixed workloads: start with 2x CPU cores and tune based on monitoring
//   - For low-spec hardware with high traffic: consider using a lower value (1-2x CPU cores)
//     to prevent memory exhaustion and context switching overhead
//
// If maxConcurrentTask <= 0, defaults to runtime.NumCPU() * 2, which provides a balance
// between throughput and resource consumption for mixed workloads.
func NewTask(maxConcurrentTask int) Task {
	// Default to 2x CPU cores for better I/O handling while preventing resource exhaustion
	if maxConcurrentTask <= 0 {
		maxConcurrentTask = runtime.NumCPU() * 2
	}

	return &task{
		semaphore:      make(chan struct{}, maxConcurrentTask),
		maxConcurrency: maxConcurrentTask,
	}
}

// Go executes the given function in a goroutine, respecting the concurrency limit.
// It uses a semaphore pattern to ensure that no more than maxConcurrentTask goroutines
// are running simultaneously. If the limit is reached, this call will block until
// a slot becomes available.
func (t *task) Go(taskFunc func()) {
	// Pre-increment counters to reduce atomic operations inside goroutine
	atomic.AddInt64(&t.submittedCount, 1)

	// Acquire semaphore slot (blocks if limit reached)
	t.semaphore <- struct{}{}

	atomic.AddInt64(&t.activeCount, 1)
	t.wg.Add(1)

	go func() {
		defer func() {
			// Release semaphore slot and decrement active counter
			<-t.semaphore
			atomic.AddInt64(&t.activeCount, -1)
			t.wg.Done()
		}()

		// Execute the actual task
		taskFunc()
	}()
}

// Wait blocks until all submitted tasks have completed execution.
// This method should be called after all Go() calls to ensure
// all goroutines finish before the program continues.
func (t *task) Wait() {
	t.wg.Wait()
}

// ActiveCount returns the current number of actively running tasks.
// This can be useful for monitoring and debugging purposes.
func (t *task) ActiveCount() int64 {
	return atomic.LoadInt64(&t.activeCount)
}

// TotalSubmitted returns the total number of tasks that have been submitted
// since the Task instance was created. This includes completed, active, and queued tasks.
func (t *task) TotalSubmitted() int64 {
	return atomic.LoadInt64(&t.submittedCount)
}
