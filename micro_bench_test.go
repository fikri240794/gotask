package gotask

import (
	"context"
	"runtime"
	"sync/atomic"
	"testing"
)

// BenchmarkTaskSubmissionOverhead benchmarks the overhead of task submission
func BenchmarkTaskSubmissionOverhead(b *testing.B) {
	task := NewTask(runtime.NumCPU() * 2)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		task.Go(func() {
			// Minimal work to test submission overhead
		})
	}
	task.Wait()
}

// BenchmarkErrorTaskSubmissionOverhead benchmarks the overhead of error task submission
func BenchmarkErrorTaskSubmissionOverhead(b *testing.B) {
	errTask, _ := NewErrorTask(context.Background(), runtime.NumCPU()*2)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		errTask.Go(func() error {
			// Minimal work to test submission overhead
			return nil
		})
	}
	errTask.Wait()
}

// BenchmarkAtomicOperations benchmarks atomic operations used in counters
func BenchmarkAtomicOperations(b *testing.B) {
	var counter int64

	b.Run("AtomicAdd", func(b *testing.B) {
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				atomic.AddInt64(&counter, 1)
			}
		})
	})

	b.Run("AtomicLoad", func(b *testing.B) {
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				_ = atomic.LoadInt64(&counter)
			}
		})
	})
}

// BenchmarkChannelOperations benchmarks semaphore channel operations
func BenchmarkChannelOperations(b *testing.B) {
	semaphore := make(chan struct{}, runtime.NumCPU())

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			semaphore <- struct{}{}
			<-semaphore
		}
	})
}

// BenchmarkErrorTaskCancellation benchmarks cancellation checking performance
func BenchmarkErrorTaskCancellation(b *testing.B) {
	ctx, cancel := context.WithCancel(context.Background())
	errTask, _ := NewErrorTask(ctx, runtime.NumCPU())

	// Cancel immediately to test cancellation path
	cancel()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		errTask.Go(func() error {
			return nil
		})
	}
}
