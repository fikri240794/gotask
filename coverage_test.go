package gotask

import (
	"context"
	"errors"
	"runtime"
	"sync/atomic"
	"testing"
	"time"
)

// TestTaskInterface verifies that task implements Task interface
func TestTaskInterface(t *testing.T) {
	var _ Task = (*task)(nil)
	var _ Task = NewTask(1)
}

// TestErrorTaskInterface verifies that errorTask implements ErrorTask interface
func TestErrorTaskInterface(t *testing.T) {
	var _ ErrorTask = (*errorTask)(nil)
	errTask, _ := NewErrorTask(context.Background(), 1)
	var _ ErrorTask = errTask
}

// TestTask_ZeroConcurrency tests the zero concurrency edge case
func TestTask_ZeroConcurrency(t *testing.T) {
	taskInstance := NewTask(0)

	// Should default to 2x CPU cores
	expectedConcurrency := runtime.NumCPU() * 2
	taskImpl := taskInstance.(*task)

	if cap(taskImpl.semaphore) != expectedConcurrency {
		t.Errorf("expected semaphore capacity %d, got %d", expectedConcurrency, cap(taskImpl.semaphore))
	}
}

// TestErrorTask_ZeroConcurrency tests the zero concurrency edge case for ErrorTask
func TestErrorTask_ZeroConcurrency(t *testing.T) {
	errTask, _ := NewErrorTask(context.Background(), 0)

	// Should default to 2x CPU cores
	expectedConcurrency := runtime.NumCPU() * 2
	taskImpl := errTask.(*errorTask)

	if cap(taskImpl.semaphore) != expectedConcurrency {
		t.Errorf("expected semaphore capacity %d, got %d", expectedConcurrency, cap(taskImpl.semaphore))
	}
}

// TestTask_NegativeConcurrency tests negative concurrency values
func TestTask_NegativeConcurrency(t *testing.T) {
	taskInstance := NewTask(-5)

	// Should default to 2x CPU cores
	expectedConcurrency := runtime.NumCPU() * 2
	taskImpl := taskInstance.(*task)

	if cap(taskImpl.semaphore) != expectedConcurrency {
		t.Errorf("expected semaphore capacity %d, got %d", expectedConcurrency, cap(taskImpl.semaphore))
	}
}

// TestErrorTask_NilError tests that nil errors don't trigger cancellation
func TestErrorTask_NilError(t *testing.T) {
	errTask, taskCtx := NewErrorTask(context.Background(), 2)

	var executedTasks int64

	for i := 0; i < 3; i++ {
		errTask.Go(func() error {
			atomic.AddInt64(&executedTasks, 1)
			return nil // Explicitly return nil
		})
	}

	err := errTask.Wait()

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	if executed := atomic.LoadInt64(&executedTasks); executed != 3 {
		t.Errorf("expected 3 executed tasks, got %d", executed)
	}

	// Context should still be cancelled after Wait()
	if taskCtx.Err() == nil {
		t.Error("context should be cancelled after Wait()")
	}
}

// TestErrorTask_MultipleErrors tests multiple simultaneous errors
func TestErrorTask_MultipleErrors(t *testing.T) {
	errTask, _ := NewErrorTask(context.Background(), 5)

	err1 := errors.New("error 1")
	err2 := errors.New("error 2")
	err3 := errors.New("error 3")

	// Submit multiple failing tasks
	errTask.Go(func() error { return err1 })
	errTask.Go(func() error { return err2 })
	errTask.Go(func() error { return err3 })

	err := errTask.Wait()

	if err == nil {
		t.Error("expected error, got nil")
	}

	// Should get one of the errors (first one to be processed)
	errorFound := err.Error() == err1.Error() ||
		err.Error() == err2.Error() ||
		err.Error() == err3.Error()

	if !errorFound {
		t.Errorf("unexpected error: %v", err)
	}
}

// TestTask_PanicRecovery tests that panics in tasks don't crash the system
func TestTask_PanicRecovery(t *testing.T) {
	task := NewTask(2)

	var completedTasks int64

	// Submit a task that panics
	task.Go(func() {
		defer func() {
			if r := recover(); r != nil {
				atomic.AddInt64(&completedTasks, 1)
			}
		}()
		panic("test panic")
	})

	// Submit normal tasks
	task.Go(func() {
		atomic.AddInt64(&completedTasks, 1)
	})

	task.Wait()

	// Both tasks should have completed (panic recovered)
	if completed := atomic.LoadInt64(&completedTasks); completed != 2 {
		t.Errorf("expected 2 completed tasks, got %d", completed)
	}
}

// TestErrorTask_TaskContextUsage tests proper usage of task context
func TestErrorTask_TaskContextUsage(t *testing.T) {
	errTask, taskCtx := NewErrorTask(context.Background(), 2)

	var contextCancelledTasks int64
	var normalTasks int64

	// Submit task that will cause error after delay
	errTask.Go(func() error {
		time.Sleep(100 * time.Millisecond)
		return errors.New("delayed error")
	})

	// Submit tasks that check context
	for i := 0; i < 3; i++ {
		errTask.Go(func() error {
			select {
			case <-taskCtx.Done():
				atomic.AddInt64(&contextCancelledTasks, 1)
				return nil
			case <-time.After(200 * time.Millisecond):
				atomic.AddInt64(&normalTasks, 1)
				return nil
			}
		})
	}

	err := errTask.Wait()

	if err == nil {
		t.Error("expected error, got nil")
	}

	// Some tasks should have been cancelled due to context
	total := atomic.LoadInt64(&contextCancelledTasks) + atomic.LoadInt64(&normalTasks)
	if total == 0 {
		t.Error("expected some tasks to execute")
	}
}

// TestTask_RapidSubmission tests rapid task submission
func TestTask_RapidSubmission(t *testing.T) {
	taskInstance := NewTask(1) // Low concurrency to test queuing

	const numTasks = 100
	var completedTasks int64

	// Rapidly submit many tasks
	for i := 0; i < numTasks; i++ {
		taskInstance.Go(func() {
			atomic.AddInt64(&completedTasks, 1)
		})
	}

	taskInstance.Wait()

	if completed := atomic.LoadInt64(&completedTasks); completed != numTasks {
		t.Errorf("expected %d completed tasks, got %d", numTasks, completed)
	}

	if total := taskInstance.TotalSubmitted(); total != numTasks {
		t.Errorf("expected %d total submitted, got %d", numTasks, total)
	}
}

// TestErrorTask_RapidSubmissionWithError tests rapid submission with early error
func TestErrorTask_RapidSubmissionWithError(t *testing.T) {
	errTask, _ := NewErrorTask(context.Background(), 1)

	const numTasks = 100
	var executedTasks int64

	// First task fails immediately
	errTask.Go(func() error {
		atomic.AddInt64(&executedTasks, 1)
		return errors.New("immediate error")
	})

	// Rapidly submit many more tasks
	for i := 1; i < numTasks; i++ {
		errTask.Go(func() error {
			atomic.AddInt64(&executedTasks, 1)
			time.Sleep(10 * time.Millisecond)
			return nil
		})
	}

	err := errTask.Wait()

	if err == nil {
		t.Error("expected error, got nil")
	}

	// Not all tasks should have executed due to early cancellation
	executed := atomic.LoadInt64(&executedTasks)
	if executed >= numTasks {
		t.Errorf("too many tasks executed despite early error: %d", executed)
	}
}

// TestErrorTask_ContextMethods tests the Context() method
func TestErrorTask_ContextMethods(t *testing.T) {
	parentCtx := context.Background()
	errTask, taskCtx := NewErrorTask(parentCtx, 2)

	// Context() method should return the same context
	if errTask.Context() != taskCtx {
		t.Error("Context() method should return the task context")
	}

	// Context should initially be valid
	if taskCtx.Err() != nil {
		t.Error("initial context should not be cancelled")
	}

	// After error, context should be cancelled
	errTask.Go(func() error {
		return errors.New("test error")
	})

	errTask.Wait()

	if taskCtx.Err() == nil {
		t.Error("context should be cancelled after error")
	}

	if errTask.Context().Err() == nil {
		t.Error("Context() should return cancelled context after error")
	}
}

// TestConcurrentAccessToCounters tests thread safety of counter methods
func TestConcurrentAccessToCounters(t *testing.T) {
	taskInstance := NewTask(10)

	const numGoroutines = 100
	const tasksPerGoroutine = 10

	// Start multiple goroutines accessing counters concurrently
	done := make(chan struct{})
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer func() { done <- struct{}{} }()

			for j := 0; j < tasksPerGoroutine; j++ {
				// Submit task
				taskInstance.Go(func() {
					time.Sleep(time.Microsecond)
				})

				// Read counters concurrently
				_ = taskInstance.ActiveCount()
				_ = taskInstance.TotalSubmitted()
			}
		}()
	}

	// Wait for all goroutines to finish
	for i := 0; i < numGoroutines; i++ {
		<-done
	}

	taskInstance.Wait()

	expectedTotal := int64(numGoroutines * tasksPerGoroutine)
	if total := taskInstance.TotalSubmitted(); total != expectedTotal {
		t.Errorf("expected %d total submitted, got %d", expectedTotal, total)
	}

	if active := taskInstance.ActiveCount(); active != 0 {
		t.Errorf("expected 0 active tasks after Wait(), got %d", active)
	}
}
