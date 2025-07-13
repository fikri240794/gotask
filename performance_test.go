package gotask

import (
	"context"
	"fmt"
	"runtime"
	"sync/atomic"
	"testing"
	"time"
)

// BenchmarkComparison_TaskVsErrorTask compares performance between Task and ErrorTask
func BenchmarkComparison_TaskVsErrorTask(b *testing.B) {
	concurrency := runtime.NumCPU()

	b.Run("Task", func(b *testing.B) {
		task := NewTask(concurrency)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			task.Go(func() {
				runtime.Gosched()
			})
		}
		task.Wait()
	})

	b.Run("ErrorTask", func(b *testing.B) {
		errTask, _ := NewErrorTask(context.Background(), concurrency)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			errTask.Go(func() error {
				runtime.Gosched()
				return nil
			})
		}
		errTask.Wait()
	})
}

// BenchmarkMemoryAllocation tests memory allocation patterns
func BenchmarkMemoryAllocation(b *testing.B) {
	concurrencies := []int{1, 2, 4, 8, 16, 32}

	for _, concurrency := range concurrencies {
		b.Run(fmt.Sprintf("Task_Concurrency%d", concurrency), func(b *testing.B) {
			b.ReportAllocs()

			task := NewTask(concurrency)

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				task.Go(func() {})
			}
			task.Wait()
		})

		b.Run(fmt.Sprintf("ErrorTask_Concurrency%d", concurrency), func(b *testing.B) {
			b.ReportAllocs()

			errTask, _ := NewErrorTask(context.Background(), concurrency)

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				errTask.Go(func() error {
					return nil
				})
			}
			errTask.Wait()
		})
	}
}

// BenchmarkHighThroughput simulates high-throughput scenarios
func BenchmarkHighThroughput(b *testing.B) {
	scenarios := []struct {
		name        string
		concurrency int
		workload    func()
	}{
		{
			name:        "LowLatency_HighConcurrency",
			concurrency: runtime.NumCPU() * 4,
			workload: func() {
				// Minimal work, test scheduling overhead
				runtime.Gosched()
			},
		},
		{
			name:        "CPUIntensive_OptimalConcurrency",
			concurrency: runtime.NumCPU(),
			workload: func() {
				// CPU-bound work
				sum := 0
				for i := 0; i < 100; i++ {
					sum += i
				}
				_ = sum
			},
		},
		{
			name:        "IOSimulation_HighConcurrency",
			concurrency: runtime.NumCPU() * 8,
			workload: func() {
				// Simulate I/O wait
				time.Sleep(time.Microsecond)
			},
		},
	}

	for _, scenario := range scenarios {
		b.Run(scenario.name+"_Task", func(b *testing.B) {
			task := NewTask(scenario.concurrency)

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				task.Go(scenario.workload)
			}
			task.Wait()
		})

		b.Run(scenario.name+"_ErrorTask", func(b *testing.B) {
			errTask, _ := NewErrorTask(context.Background(), scenario.concurrency)

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				errTask.Go(func() error {
					scenario.workload()
					return nil
				})
			}
			errTask.Wait()
		})
	}
}

// BenchmarkResourceContention tests performance under resource contention
func BenchmarkResourceContention(b *testing.B) {
	var sharedCounter int64

	b.Run("Task_SharedResource", func(b *testing.B) {
		task := NewTask(runtime.NumCPU())

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			task.Go(func() {
				atomic.AddInt64(&sharedCounter, 1)
			})
		}
		task.Wait()
	})

	atomic.StoreInt64(&sharedCounter, 0) // Reset counter

	b.Run("ErrorTask_SharedResource", func(b *testing.B) {
		errTask, _ := NewErrorTask(context.Background(), runtime.NumCPU())

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			errTask.Go(func() error {
				atomic.AddInt64(&sharedCounter, 1)
				return nil
			})
		}
		errTask.Wait()
	})
}

// TestPerformanceRegression ensures performance doesn't degrade over time
func TestPerformanceRegression(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping performance regression test in short mode")
	}

	// Performance baseline: adjust for race detector overhead
	// Race detector typically adds 2-10x overhead, so we use a more conservative baseline
	var minTasksPerMs float64 = 100 // Conservative baseline for race detector

	// Check if we're running with race detector
	// Race detector significantly impacts performance, so we use a lower baseline
	if isRaceDetectorEnabled() {
		minTasksPerMs = 50 // Even more conservative with race detector
	}

	const numTasks = 10000

	t.Run("Task_Performance", func(t *testing.T) {
		task := NewTask(runtime.NumCPU())

		start := time.Now()
		for i := 0; i < numTasks; i++ {
			task.Go(func() {
				runtime.Gosched()
			})
		}
		task.Wait()
		duration := time.Since(start)

		tasksPerMs := float64(numTasks) / float64(duration.Nanoseconds()/1e6)

		if tasksPerMs < minTasksPerMs {
			t.Errorf("Task performance below baseline: %.2f tasks/ms (expected >= %.0f)", tasksPerMs, minTasksPerMs)
		}

		t.Logf("Task performance: %.2f tasks/ms", tasksPerMs)
	})

	t.Run("ErrorTask_Performance", func(t *testing.T) {
		errTask, _ := NewErrorTask(context.Background(), runtime.NumCPU())

		start := time.Now()
		for i := 0; i < numTasks; i++ {
			errTask.Go(func() error {
				runtime.Gosched()
				return nil
			})
		}
		errTask.Wait()
		duration := time.Since(start)

		tasksPerMs := float64(numTasks) / float64(duration.Nanoseconds()/1e6)

		if tasksPerMs < minTasksPerMs {
			t.Errorf("ErrorTask performance below baseline: %.2f tasks/ms (expected >= %.0f)", tasksPerMs, minTasksPerMs)
		}

		t.Logf("ErrorTask performance: %.2f tasks/ms", tasksPerMs)
	})
}

// isRaceDetectorEnabled checks if the race detector is enabled
// This is a simple heuristic based on runtime behavior
func isRaceDetectorEnabled() bool {
	// A simple way to detect if race detector is enabled
	// Race detector adds significant overhead to atomic operations
	start := time.Now()
	var counter int64
	for i := 0; i < 1000; i++ {
		atomic.AddInt64(&counter, 1)
	}
	duration := time.Since(start)

	// If 1000 atomic operations take more than 1ms, likely race detector is enabled
	return duration > time.Millisecond
}
