package main

import (
	"context"
	"fmt"
	"runtime"
	"time"

	"github.com/fikri240794/gotask"
)

func main() {
	fmt.Println("GoTask Examples")
	fmt.Printf("Running on %d CPU cores\n\n", runtime.NumCPU())

	// Example 1: Basic Task usage
	basicTaskExample()

	// Example 2: Task with monitoring
	taskWithMonitoringExample()

	// Example 3: ErrorTask with success
	errorTaskSuccessExample()

	// Example 4: ErrorTask with error handling
	errorTaskErrorExample()

	// Example 5: Performance comparison
	performanceComparisonExample()
}

// basicTaskExample demonstrates basic Task usage
func basicTaskExample() {
	fmt.Println("=== Basic Task Example ===")

	// Create task with concurrency limit of 2
	task := gotask.NewTask(2)

	fmt.Println("Submitting 5 tasks with max concurrency of 2...")

	for i := 0; i < 5; i++ {
		taskNum := i + 1
		task.Go(func() {
			fmt.Printf("Task %d started\n", taskNum)
			time.Sleep(500 * time.Millisecond) // Simulate work
			fmt.Printf("Task %d completed\n", taskNum)
		})
	}

	fmt.Println("Waiting for all tasks to complete...")
	task.Wait()
	fmt.Println("All tasks completed!")
	fmt.Println()
}

// taskWithMonitoringExample demonstrates Task monitoring capabilities
func taskWithMonitoringExample() {
	fmt.Println("=== Task with Monitoring Example ===")

	task := gotask.NewTask(3)

	// Submit tasks
	for i := 0; i < 8; i++ {
		task.Go(func() {
			time.Sleep(200 * time.Millisecond)
		})
	}

	// Monitor progress
	fmt.Printf("Total submitted: %d\n", task.TotalSubmitted())

	// Check active tasks periodically
	done := make(chan struct{})
	go func() {
		for {
			select {
			case <-done:
				return
			case <-time.After(100 * time.Millisecond):
				fmt.Printf("Active tasks: %d\n", task.ActiveCount())
			}
		}
	}()

	task.Wait()
	close(done)

	fmt.Printf("Final - Active: %d, Total: %d\n\n", task.ActiveCount(), task.TotalSubmitted())
}

// errorTaskSuccessExample demonstrates ErrorTask with successful execution
func errorTaskSuccessExample() {
	fmt.Println("=== ErrorTask Success Example ===")

	errTask, taskCtx := gotask.NewErrorTask(context.Background(), 2)

	fmt.Println("Submitting 4 successful tasks...")

	for i := 0; i < 4; i++ {
		taskNum := i + 1
		errTask.Go(func() error {
			// Use the task context to respect cancellation
			select {
			case <-taskCtx.Done():
				fmt.Printf("Task %d cancelled\n", taskNum)
				return nil
			case <-time.After(300 * time.Millisecond):
				fmt.Printf("Task %d completed successfully\n", taskNum)
				return nil
			}
		})
	}

	err := errTask.Wait()
	if err != nil {
		fmt.Printf("Error occurred: %v\n", err)
	} else {
		fmt.Println("All tasks completed successfully!")
	}
	fmt.Println()
}

// errorTaskErrorExample demonstrates ErrorTask with error handling
func errorTaskErrorExample() {
	fmt.Println("=== ErrorTask Error Handling Example ===")

	errTask, taskCtx := gotask.NewErrorTask(context.Background(), 2)

	fmt.Println("Submitting tasks where one will fail...")

	for i := 0; i < 5; i++ {
		taskNum := i + 1
		errTask.Go(func() error {
			select {
			case <-taskCtx.Done():
				fmt.Printf("Task %d cancelled due to error in another task\n", taskNum)
				return nil
			case <-time.After(time.Duration(taskNum*200) * time.Millisecond):
				if taskNum == 3 {
					fmt.Printf("Task %d failed!\n", taskNum)
					return fmt.Errorf("task %d encountered an error", taskNum)
				}
				fmt.Printf("Task %d completed successfully\n", taskNum)
				return nil
			}
		})
	}

	err := errTask.Wait()
	if err != nil {
		fmt.Printf("Error occurred: %v\n", err)
		fmt.Println("Remaining tasks were cancelled due to error propagation")
	} else {
		fmt.Println("All tasks completed successfully!")
	}
	fmt.Println()
}

// performanceComparisonExample demonstrates performance characteristics
func performanceComparisonExample() {
	fmt.Println("=== Performance Comparison Example ===")

	const numTasks = 1000
	concurrency := runtime.NumCPU()

	// Test regular Task
	fmt.Printf("Testing Task with %d tasks, concurrency %d...\n", numTasks, concurrency)
	start := time.Now()

	task := gotask.NewTask(concurrency)
	for i := 0; i < numTasks; i++ {
		task.Go(func() {
			// Simulate minimal CPU work
			runtime.Gosched()
		})
	}
	task.Wait()

	taskDuration := time.Since(start)
	fmt.Printf("Task completed in: %v\n", taskDuration)

	// Test ErrorTask
	fmt.Printf("Testing ErrorTask with %d tasks, concurrency %d...\n", numTasks, concurrency)
	start = time.Now()

	errTask, _ := gotask.NewErrorTask(context.Background(), concurrency)
	for i := 0; i < numTasks; i++ {
		errTask.Go(func() error {
			// Simulate minimal CPU work
			runtime.Gosched()
			return nil
		})
	}
	errTask.Wait()

	errorTaskDuration := time.Since(start)
	fmt.Printf("ErrorTask completed in: %v\n", errorTaskDuration)

	// Calculate overhead
	overhead := float64(errorTaskDuration-taskDuration) / float64(taskDuration) * 100
	fmt.Printf("ErrorTask overhead: %.2f%%\n", overhead)
}
