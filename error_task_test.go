package gotask

import (
	"context"
	"errors"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// TestNewErrorTask tests the ErrorTask constructor with various input values
func TestNewErrorTask(t *testing.T) {
	tests := []struct {
		name                   string
		ctx                    context.Context
		maxConcurrentTask      int
		expectedMaxConcurrency int
	}{
		{
			name:                   "zero concurrent tasks defaults to 2x CPU cores",
			ctx:                    context.Background(),
			maxConcurrentTask:      0,
			expectedMaxConcurrency: runtime.NumCPU() * 2,
		},
		{
			name:                   "negative concurrent tasks defaults to 2x CPU cores",
			ctx:                    context.Background(),
			maxConcurrentTask:      -1,
			expectedMaxConcurrency: runtime.NumCPU() * 2,
		},
		{
			name:                   "positive concurrent tasks uses provided value",
			ctx:                    context.Background(),
			maxConcurrentTask:      4,
			expectedMaxConcurrency: 4,
		},
		{
			name:                   "single concurrent task",
			ctx:                    context.Background(),
			maxConcurrentTask:      1,
			expectedMaxConcurrency: 1,
		},
		{
			name:                   "with cancelled context",
			ctx:                    func() context.Context { ctx, cancel := context.WithCancel(context.Background()); cancel(); return ctx }(),
			maxConcurrentTask:      2,
			expectedMaxConcurrency: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errTask, taskCtx := NewErrorTask(tt.ctx, tt.maxConcurrentTask)
			task := errTask.(*errorTask)

			if cap(task.semaphore) != tt.expectedMaxConcurrency {
				t.Errorf("expected semaphore capacity %d, got %d", tt.expectedMaxConcurrency, cap(task.semaphore))
			}

			if task.maxConcurrency != tt.expectedMaxConcurrency {
				t.Errorf("expected maxConcurrency %d, got %d", tt.expectedMaxConcurrency, task.maxConcurrency)
			}

			if taskCtx != task.ctx {
				t.Error("returned context should match internal context")
			}

			if errTask.Context() != taskCtx {
				t.Error("Context() method should return the same context")
			}
		})
	}
}

// TestErrorTask_SuccessfulExecution tests successful task execution without errors
func TestErrorTask_SuccessfulExecution(t *testing.T) {
	const numTasks = 5

	errTask, _ := NewErrorTask(context.Background(), 2)

	var completedTasks int64

	for i := 0; i < numTasks; i++ {
		errTask.Go(func() error {
			atomic.AddInt64(&completedTasks, 1)
			time.Sleep(10 * time.Millisecond)
			return nil
		})
	}

	err := errTask.Wait()

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	if completed := atomic.LoadInt64(&completedTasks); completed != numTasks {
		t.Errorf("expected %d completed tasks, got %d", numTasks, completed)
	}

	if total := errTask.TotalSubmitted(); total != numTasks {
		t.Errorf("expected %d total submitted, got %d", numTasks, total)
	}
}

// TestErrorTask_ErrorHandling tests error propagation and cancellation
func TestErrorTask_ErrorHandling(t *testing.T) {
	errTask, taskCtx := NewErrorTask(context.Background(), 2)

	expectedErr := errors.New("test error")
	var completedTasks int64
	var cancelledTasks int64

	// Submit multiple tasks, one will fail
	for i := 0; i < 5; i++ {
		taskNum := i
		errTask.Go(func() error {
			// First task fails immediately
			if taskNum == 0 {
				atomic.AddInt64(&completedTasks, 1)
				return expectedErr
			}

			// Other tasks check cancellation
			select {
			case <-taskCtx.Done():
				atomic.AddInt64(&cancelledTasks, 1)
				return nil
			case <-time.After(100 * time.Millisecond):
				atomic.AddInt64(&completedTasks, 1)
				return nil
			}
		})
	}

	err := errTask.Wait()

	if err == nil {
		t.Error("expected error, got nil")
	}

	if err.Error() != expectedErr.Error() {
		t.Errorf("expected error %q, got %q", expectedErr.Error(), err.Error())
	}

	// Context should be cancelled after error
	if taskCtx.Err() == nil {
		t.Error("context should be cancelled after error")
	}
}

// TestErrorTask_FirstErrorWins tests that only the first error is returned
func TestErrorTask_FirstErrorWins(t *testing.T) {
	errTask, _ := NewErrorTask(context.Background(), 3)

	err1 := errors.New("first error")
	err2 := errors.New("second error")

	var errorBarrier sync.WaitGroup
	errorBarrier.Add(2)

	// Two tasks that both return errors simultaneously
	errTask.Go(func() error {
		errorBarrier.Wait() // Wait for both to be ready
		return err1
	})

	errTask.Go(func() error {
		errorBarrier.Wait() // Wait for both to be ready
		return err2
	})

	// Release both error tasks simultaneously
	errorBarrier.Done()
	errorBarrier.Done()

	err := errTask.Wait()

	if err == nil {
		t.Error("expected error, got nil")
	}

	// Should get one of the errors (first one wins)
	if err.Error() != err1.Error() && err.Error() != err2.Error() {
		t.Errorf("expected either %q or %q, got %q", err1.Error(), err2.Error(), err.Error())
	}
}

// TestErrorTask_ConcurrencyLimit tests that the error task respects concurrency limits
func TestErrorTask_ConcurrencyLimit(t *testing.T) {
	const maxConcurrency = 2
	const numTasks = 10

	errTask, _ := NewErrorTask(context.Background(), maxConcurrency)

	var (
		currentConcurrency     int64
		maxObservedConcurrency int64
		completedTasks         int64
	)

	for i := 0; i < numTasks; i++ {
		errTask.Go(func() error {
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
			return nil
		})
	}

	err := errTask.Wait()

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// Verify all tasks completed
	if completed := atomic.LoadInt64(&completedTasks); completed != numTasks {
		t.Errorf("expected %d completed tasks, got %d", numTasks, completed)
	}

	// Verify concurrency limit was respected
	if maxObserved := atomic.LoadInt64(&maxObservedConcurrency); maxObserved > maxConcurrency {
		t.Errorf("concurrency limit exceeded: max observed %d, limit %d", maxObserved, maxConcurrency)
	}
}

// TestErrorTask_CancelledContext tests behavior with pre-cancelled context
func TestErrorTask_CancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	errTask, taskCtx := NewErrorTask(ctx, 2)

	// Tasks submitted to cancelled context should not execute
	var executedTasks int64

	for i := 0; i < 3; i++ {
		errTask.Go(func() error {
			atomic.AddInt64(&executedTasks, 1)
			return nil
		})
	}

	err := errTask.Wait()

	// Should not return an error for cancelled context (no tasks executed)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// No tasks should have executed
	if executed := atomic.LoadInt64(&executedTasks); executed != 0 {
		t.Errorf("expected 0 executed tasks, got %d", executed)
	}

	// Context should be cancelled
	if taskCtx.Err() == nil {
		t.Error("context should be cancelled")
	}
}

// TestErrorTask_ActiveCountAndTotalSubmitted tests monitoring methods
func TestErrorTask_ActiveCountAndTotalSubmitted(t *testing.T) {
	const maxConcurrency = 2
	const numTasks = 5

	errTask, _ := NewErrorTask(context.Background(), maxConcurrency)

	// Initially, counters should be zero
	if active := errTask.ActiveCount(); active != 0 {
		t.Errorf("expected 0 active tasks initially, got %d", active)
	}
	if total := errTask.TotalSubmitted(); total != 0 {
		t.Errorf("expected 0 total submitted initially, got %d", total)
	}

	// Use a simpler approach: submit tasks without blocking
	var tasksStarted int64
	var tasksFinished int64

	// Submit tasks that increment counters
	for i := 0; i < numTasks; i++ {
		errTask.Go(func() error {
			atomic.AddInt64(&tasksStarted, 1)

			// Simulate some work
			time.Sleep(50 * time.Millisecond)

			atomic.AddInt64(&tasksFinished, 1)
			return nil
		})
	}

	// Give tasks a moment to start
	time.Sleep(100 * time.Millisecond)

	// Check that total submitted is correct (all tasks should be submitted immediately)
	if total := errTask.TotalSubmitted(); total != numTasks {
		t.Errorf("expected %d total submitted, got %d", numTasks, total)
	}

	// Check active count (should be limited by maxConcurrency)
	active := errTask.ActiveCount()
	if active > maxConcurrency {
		t.Errorf("active tasks exceeded concurrency limit: got %d, max %d", active, maxConcurrency)
	}

	// Wait for all tasks to complete
	err := errTask.Wait()

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// After completion, all tasks should have finished
	if finished := atomic.LoadInt64(&tasksFinished); finished != numTasks {
		t.Errorf("expected %d finished tasks, got %d", numTasks, finished)
	}

	// After completion, active should be 0, total should remain
	if active := errTask.ActiveCount(); active != 0 {
		t.Errorf("expected 0 active tasks after Wait(), got %d", active)
	}
	if total := errTask.TotalSubmitted(); total != numTasks {
		t.Errorf("expected %d total submitted after Wait(), got %d", numTasks, total)
	}
}

// TestErrorTask_EmptyWorkload tests behavior with no tasks submitted
func TestErrorTask_EmptyWorkload(t *testing.T) {
	errTask, _ := NewErrorTask(context.Background(), 2)

	// Wait should return immediately with no tasks
	err := errTask.Wait()

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// Counters should remain zero
	if active := errTask.ActiveCount(); active != 0 {
		t.Errorf("expected 0 active tasks, got %d", active)
	}
	if total := errTask.TotalSubmitted(); total != 0 {
		t.Errorf("expected 0 total submitted, got %d", total)
	}
}

// TestErrorTask_ContextTimeout tests context timeout handling
func TestErrorTask_ContextTimeout(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	errTask, taskCtx := NewErrorTask(ctx, 2)

	var completedTasks int64

	// Submit tasks that take longer than context timeout
	for i := 0; i < 3; i++ {
		errTask.Go(func() error {
			select {
			case <-taskCtx.Done():
				return nil // Cancelled due to timeout
			case <-time.After(200 * time.Millisecond):
				atomic.AddInt64(&completedTasks, 1)
				return nil
			}
		})
	}

	err := errTask.Wait()

	// Should not return an error (timeout is not an error from task perspective)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// Context should be done due to timeout
	if taskCtx.Err() == nil {
		t.Error("context should be cancelled due to timeout")
	}

	// Some tasks might complete before timeout, but not all
	completed := atomic.LoadInt64(&completedTasks)
	if completed == 3 {
		t.Error("all tasks completed despite context timeout")
	}
}

// BenchmarkErrorTask_Throughput benchmarks error task submission and execution throughput
func BenchmarkErrorTask_Throughput(b *testing.B) {
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
			errTask, _ := NewErrorTask(context.Background(), bm.concurrency)

			b.ResetTimer()
			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					errTask.Go(func() error {
						// Simulate minimal work
						runtime.Gosched()
						return nil
					})
				}
			})
			errTask.Wait()
		})
	}
}

// BenchmarkErrorTask_WithErrorHandling benchmarks performance with error conditions
func BenchmarkErrorTask_WithErrorHandling(b *testing.B) {
	errTask, _ := NewErrorTask(context.Background(), runtime.NumCPU())

	testErr := errors.New("benchmark error")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		errTask.Go(func() error {
			if i%100 == 0 { // Introduce errors occasionally
				return testErr
			}
			return nil
		})
	}
	errTask.Wait()
}

// TestErrorTask_SetErrorRaceCondition tests race conditions in setError method
// to ensure 100% coverage of the double-check pattern
func TestErrorTask_SetErrorRaceCondition(t *testing.T) {
	const numGoroutines = 50
	const numAttempts = 10

	for attempt := 0; attempt < numAttempts; attempt++ {
		errTask, _ := NewErrorTask(context.Background(), numGoroutines)

		var startBarrier sync.WaitGroup
		startBarrier.Add(numGoroutines)

		// Submit many tasks that will all try to set error simultaneously
		for i := 0; i < numGoroutines; i++ {
			taskNum := i
			errTask.Go(func() error {
				startBarrier.Done()
				startBarrier.Wait() // All goroutines start simultaneously

				// Return unique errors to force race condition in setError
				return errors.New("error from task " + string(rune(taskNum)))
			})
		}

		err := errTask.Wait()

		// Should get exactly one error (first one wins)
		if err == nil {
			t.Error("expected error, got nil")
		}

		// Verify error string contains expected format
		if err.Error() == "" {
			t.Error("error should not be empty")
		}
	}
}
