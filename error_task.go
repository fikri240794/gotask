package gotask

import (
	"context"
	"runtime"
	"sync"
	"sync/atomic"
)

// ErrorTask provides a Task interface with error handling and context cancellation capabilities.
// When any task returns an error, all remaining tasks are cancelled and the error is returned.
type ErrorTask interface {
	// Go executes the given function in a goroutine, respecting the concurrency limit.
	// If any previously submitted task has failed, this method returns immediately without
	// executing the task.
	Go(task func() error)

	// Wait blocks until all submitted tasks have completed or any task returns an error.
	// Returns the first error encountered, if any.
	Wait() error

	// ActiveCount returns the current number of active/running tasks.
	ActiveCount() int64

	// TotalSubmitted returns the total number of tasks that have been submitted.
	TotalSubmitted() int64

	// Context returns the internal context that gets cancelled when an error occurs.
	Context() context.Context
}

// errorTask implements ErrorTask with enhanced error handling and performance monitoring.
type errorTask struct {
	wg             sync.WaitGroup     // synchronizes completion of all tasks
	semaphore      chan struct{}      // controls maximum concurrent executions
	err            error              // stores the first error encountered
	errMutex       sync.RWMutex       // protects err field (RWMutex for better read performance)
	ctx            context.Context    // context for cancellation propagation
	cancelCtx      context.CancelFunc // function to cancel the context
	activeCount    int64              // atomic counter for active tasks
	submittedCount int64              // atomic counter for total submitted tasks
	maxConcurrency int                // maximum number of concurrent tasks allowed
	cancelled      int32              // atomic flag for quick cancellation check (0 = not cancelled, 1 = cancelled)
}

// NewErrorTask creates a new ErrorTask instance with the specified parameters.
//
// Performance considerations:
// - Uses the same maxConcurrentTask logic as NewTask for consistency
// - Creates a cancellable context derived from the provided parent context
// - Returns both the ErrorTask instance and the internal context for task functions to use
//
// The returned context should be used by all task functions to respect cancellation
// when an error occurs in any other task.
func NewErrorTask(ctx context.Context, maxConcurrentTask int) (ErrorTask, context.Context) {
	// Apply same default logic as regular Task for consistency
	if maxConcurrentTask <= 0 {
		maxConcurrentTask = runtime.NumCPU() * 2
	}

	errTask := &errorTask{
		semaphore:      make(chan struct{}, maxConcurrentTask),
		maxConcurrency: maxConcurrentTask,
	}

	// Create cancellable context for error propagation
	errTask.ctx, errTask.cancelCtx = context.WithCancel(ctx)

	return errTask, errTask.ctx
}

// Go executes the given function in a goroutine with error handling.
// If the context has already been cancelled (due to a previous error or parent context),
// this method returns immediately without executing the task.
// Otherwise, it respects the concurrency limit using a semaphore pattern.
func (et *errorTask) Go(taskFunc func() error) {
	// Fast-path: check atomic cancellation flag first (cheaper than context check)
	if atomic.LoadInt32(&et.cancelled) != 0 {
		return
	}

	// Check if parent context is already cancelled
	select {
	case <-et.ctx.Done():
		return
	default:
	}

	// Pre-increment submitted counter
	atomic.AddInt64(&et.submittedCount, 1)

	// Acquire semaphore slot (blocks if limit reached)
	et.semaphore <- struct{}{}

	atomic.AddInt64(&et.activeCount, 1)
	et.wg.Add(1)

	go func() {
		defer func() {
			// Release semaphore slot and decrement active counter
			<-et.semaphore
			atomic.AddInt64(&et.activeCount, -1)
			et.wg.Done()
		}()

		// Check cancellation again inside goroutine (double-check pattern)
		if atomic.LoadInt32(&et.cancelled) != 0 {
			return
		}

		// Execute task and handle any error
		if err := taskFunc(); err != nil {
			et.setError(err)
		}
	}()
}

// setError safely sets the first error and cancels the context.
// Uses a fast-path read check before acquiring write lock for better performance.
func (et *errorTask) setError(err error) {
	// Fast-path: check if error is already set (read-only, no lock needed for atomic check)
	et.errMutex.RLock()
	if et.err != nil {
		et.errMutex.RUnlock()
		return // Error already set, no need to do anything
	}
	et.errMutex.RUnlock()

	// Slow-path: acquire write lock and set error
	et.errMutex.Lock()
	defer et.errMutex.Unlock()

	// Double-check after acquiring write lock (another goroutine might have set it)
	if et.err == nil {
		et.err = err
		atomic.StoreInt32(&et.cancelled, 1) // Set atomic cancellation flag
		et.cancelCtx()                      // Cancel context to signal other goroutines to stop
	}
}

// Wait blocks until all submitted tasks complete or any task returns an error.
// Returns the first error encountered, if any.
func (et *errorTask) Wait() error {
	et.wg.Wait()

	// Set cancellation flag and cancel context to ensure cleanup
	atomic.StoreInt32(&et.cancelled, 1)
	et.cancelCtx() // Ensure context is cancelled even if no errors occurred

	// Read error safely
	et.errMutex.RLock()
	defer et.errMutex.RUnlock()
	return et.err
}

// ActiveCount returns the current number of actively running tasks.
func (et *errorTask) ActiveCount() int64 {
	return atomic.LoadInt64(&et.activeCount)
}

// TotalSubmitted returns the total number of tasks that have been submitted.
func (et *errorTask) TotalSubmitted() int64 {
	return atomic.LoadInt64(&et.submittedCount)
}

// Context returns the internal context that gets cancelled when an error occurs.
// Task functions should use this context to respect cancellation.
func (et *errorTask) Context() context.Context {
	return et.ctx
}
