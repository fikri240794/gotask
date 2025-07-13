# GoTask - High-Performance Concurrent Task Execution

[![Go Version](https://img.shields.io/badge/go-1.18+-blue.svg)](https://golang.org)

A lightweight, high-performance Go library for managing concurrent goroutines with controlled concurrency limits. GoTask provides two main interfaces: `Task` for simple concurrent execution and `ErrorTask` for error-aware concurrent execution with context cancellation.

## Features

- **Controlled Concurrency**: Limit the number of simultaneously running goroutines
- **High Performance**: Optimized for minimal overhead and maximum throughput
- **Error Handling**: Built-in error propagation and context cancellation
- **Monitoring**: Real-time tracking of active and submitted tasks
- **Context Support**: Full context.Context integration for cancellation and timeouts
- **Zero Dependencies**: Pure Go implementation with no external dependencies

## Performance Characteristics

- **Default Concurrency**: Intelligently defaults to `2 × CPU cores` for mixed I/O/CPU workloads
- **Low Overhead**: Minimal memory allocation and fast task scheduling
- **Scalable**: Tested with thousands of concurrent tasks
- **Resource Efficient**: Optimized for low-spec hardware with high traffic

## Installation

```bash
go get github.com/fikri240794/gotask
```

## Quick Start

### Basic Task Usage

```go
package main

import (
    "fmt"
    "time"
    "github.com/fikri240794/gotask"
)

func main() {
    // Create task with concurrency limit of 2
    task := gotask.NewTask(2)

    // Submit multiple tasks
    for i := 0; i < 5; i++ {
        taskNum := i + 1
        task.Go(func() {
            fmt.Printf("Task %d executing\n", taskNum)
            time.Sleep(100 * time.Millisecond)
        })
    }

    // Wait for all tasks to complete
    task.Wait()
    fmt.Println("All tasks completed!")
}
```

### ErrorTask with Error Handling

```go
package main

import (
    "context"
    "fmt"
    "errors"
    "github.com/fikri240794/gotask"
)

func main() {
    // Create error-aware task with context
    errTask, taskCtx := gotask.NewErrorTask(context.Background(), 3)

    // Submit tasks that may fail
    for i := 0; i < 5; i++ {
        taskNum := i + 1
        errTask.Go(func() error {
            // Use taskCtx to respect cancellation
            select {
            case <-taskCtx.Done():
                return nil // Cancelled
            default:
                if taskNum == 3 {
                    return errors.New("task failed")
                }
                fmt.Printf("Task %d completed\n", taskNum)
                return nil
            }
        })
    }

    // Wait and handle errors
    if err := errTask.Wait(); err != nil {
        fmt.Printf("Error occurred: %v\n", err)
    }
}
```

## API Reference

### Task Interface

```go
type Task interface {
    // Go executes function in goroutine, respecting concurrency limit
    Go(task func())
    
    // Wait blocks until all submitted tasks complete
    Wait()
    
    // ActiveCount returns current number of running tasks
    ActiveCount() int64
    
    // TotalSubmitted returns total number of submitted tasks
    TotalSubmitted() int64
}
```

### ErrorTask Interface

```go
type ErrorTask interface {
    // Go executes function with error handling
    Go(task func() error)
    
    // Wait blocks until completion or first error
    Wait() error
    
    // ActiveCount returns current number of running tasks
    ActiveCount() int64
    
    // TotalSubmitted returns total number of submitted tasks
    TotalSubmitted() int64
    
    // Context returns the cancellable context
    Context() context.Context
}
```

## Concurrency Guidelines

### Choosing the Right Concurrency Level

The `maxConcurrentTask` parameter should be chosen based on your workload:

- **CPU-bound tasks**: Use `runtime.NumCPU()` or slightly less
- **I/O-bound tasks**: Use `2-10 × runtime.NumCPU()` depending on I/O wait time
- **Mixed workloads**: Start with `2 × runtime.NumCPU()` (default when `maxConcurrentTask <= 0`)
- **Low-spec hardware with high traffic**: Use `1-2 × runtime.NumCPU()` to prevent resource exhaustion

### Performance Tips

1. **Batch small tasks** to reduce scheduling overhead
2. **Use ErrorTask** when you need error handling and cancellation
3. **Monitor active tasks** in production using `ActiveCount()`
4. **Profile memory usage** under high load to tune concurrency limits
5. **Use context timeouts** with ErrorTask for better resource management

## Examples

See the [examples](examples/) directory for comprehensive usage examples including:

- Basic task execution
- Task monitoring and metrics
- Error handling and cancellation
- Performance comparison scenarios
- Real-world usage patterns

## Testing

Run the complete test suite:

```bash
# Run all tests with coverage
go test -v -cover ./...

# Run performance benchmarks
go test -v -bench=. -benchmem ./...

# Run specific test categories
go test -v -run=TestTask ./...           # Task tests
go test -v -run=TestErrorTask ./...      # ErrorTask tests
go test -v -run=BenchmarkTask ./...      # Task benchmarks
go test -v -run=BenchmarkErrorTask ./... # ErrorTask benchmarks

# Performance regression tests
go test -v -run=TestPerformanceRegression ./...
```

### Test Coverage

The test suite provides 100% code coverage with tests for:

- **Functional Testing**: All API methods and edge cases
- **Concurrency Testing**: Thread safety and race condition detection
- **Performance Testing**: Throughput and latency benchmarks
- **Error Handling**: Error propagation and context cancellation
- **Resource Management**: Memory allocation and goroutine lifecycle
- **Regression Testing**: Performance baseline validation
