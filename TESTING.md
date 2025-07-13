# Testing Guide for GoTask

This document provides comprehensive instructions for running and understanding the test suite for GoTask.

## Overview

The GoTask project includes a complete test suite with **100% code coverage** that tests:

- **Functional correctness**: All API methods and edge cases
- **Concurrency safety**: Thread safety and race condition detection  
- **Performance characteristics**: Throughput, latency, and resource usage
- **Error handling**: Error propagation and context cancellation
- **Resource management**: Memory allocation and goroutine lifecycle

## Test Files Structure

```
├── task_test.go           # Core Task interface tests
├── error_task_test.go     # ErrorTask interface tests  
├── performance_test.go    # Performance benchmarks and comparisons
└── coverage_test.go       # Edge cases and 100% coverage tests
```

## Running Tests

### Basic Test Execution

```bash
# Run all tests with verbose output
go test -v ./...

# Run tests with coverage report
go test -v -cover ./...

# Generate detailed coverage report
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out -o coverage.html
```

### Functional Tests

```bash
# Run only Task interface tests
go test -v -run="TestTask" ./...

# Run only ErrorTask interface tests  
go test -v -run="TestErrorTask" ./...

# Run only interface compliance tests
go test -v -run="TestTaskInterface|TestErrorTaskInterface" ./...

# Run concurrency limit tests
go test -v -run="ConcurrencyLimit" ./...

# Run error handling tests
go test -v -run="Error" ./...
```

### Performance Tests

```bash
# Run all performance benchmarks
go test -v -bench=. -benchmem ./...

# Run Task-specific benchmarks
go test -v -bench="BenchmarkTask" -benchmem ./...

# Run ErrorTask-specific benchmarks  
go test -v -bench="BenchmarkErrorTask" -benchmem ./...

# Run performance comparison benchmarks
go test -v -bench="BenchmarkComparison" -benchmem ./...

# Run high-throughput scenario benchmarks
go test -v -bench="BenchmarkHighThroughput" -benchmem ./...

# Run memory allocation benchmarks
go test -v -bench="BenchmarkMemoryAllocation" -benchmem ./...

# Run resource contention benchmarks
go test -v -bench="BenchmarkResourceContention" -benchmem ./...
```

### Performance Regression Tests

```bash
# Run performance regression tests (longer running)
go test -v -run="TestPerformanceRegression" ./...

# Skip performance tests in CI/short mode
go test -v -short ./...
```

### Coverage Analysis

```bash
# Generate coverage report and open in browser
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# Get coverage statistics by function
go tool cover -func=coverage.out

# Check for 100% coverage
go test -cover ./... | grep "coverage: 100.0%"
```

### Race Condition Detection

```bash
# Run tests with race detector (important for concurrency testing)
go test -race -v ./...

# Run specific concurrency tests with race detector
go test -race -v -run="Concurrent|ActiveCount|TotalSubmitted" ./...
```

### Memory and Performance Profiling

```bash
# CPU profiling during benchmarks
go test -bench=. -cpuprofile=cpu.prof ./...
go tool pprof cpu.prof

# Memory profiling during benchmarks  
go test -bench=. -memprofile=mem.prof ./...
go tool pprof mem.prof

# Block profiling for goroutine analysis
go test -bench=. -blockprofile=block.prof ./...
go tool pprof block.prof
```

## Test Categories Explained

### 1. Functional Tests

**Purpose**: Verify that all public APIs work correctly under normal conditions.

**Key Test Cases**:
- Constructor behavior with various concurrency limits
- Task submission and execution
- Wait() functionality
- Counter methods (ActiveCount, TotalSubmitted)
- Error propagation and context cancellation

**Example**:
```bash
go test -v -run="TestNewTask|TestTask_EmptyWorkload" ./...
```

### 2. Concurrency Tests

**Purpose**: Ensure thread safety and correct behavior under concurrent access.

**Key Test Cases**:
- Concurrency limit enforcement
- Thread-safe counter access
- Race condition detection
- Simultaneous task submission

**Example**:
```bash
go test -race -v -run="TestTask_ConcurrencyLimit|TestConcurrentAccessToCounters" ./...
```

### 3. Error Handling Tests

**Purpose**: Verify error propagation, context cancellation, and cleanup.

**Key Test Cases**:
- Error propagation to Wait()
- Context cancellation on first error
- Multiple simultaneous errors
- Error handling edge cases

**Example**:
```bash
go test -v -run="TestErrorTask_ErrorHandling|TestErrorTask_FirstErrorWins" ./...
```

### 4. Performance Tests

**Purpose**: Measure throughput, latency, and resource usage characteristics.

**Key Metrics**:
- Tasks per second throughput
- Memory allocations per operation
- CPU utilization patterns
- Goroutine overhead

**Example**:
```bash
go test -bench="BenchmarkTask_Throughput" -benchmem ./...
```

### 5. Edge Case Tests

**Purpose**: Test boundary conditions and unusual scenarios.

**Key Test Cases**:
- Zero/negative concurrency values
- Cancelled contexts
- Rapid task submission
- Empty workloads

**Example**:
```bash
go test -v -run="TestTask_ZeroConcurrency|TestErrorTask_CancelledContext" ./...
```

## Performance Baselines

The test suite includes performance regression tests with the following baselines:

- **Without Race Detector**: 100+ tasks/millisecond minimum
- **With Race Detector**: 50+ tasks/millisecond minimum (race detector adds 2-10x overhead)
- **Memory Efficiency**: Minimal allocations per task
- **Concurrency Overhead**: <50% compared to basic Task (ErrorTask vs Task)

**Note**: Performance baselines are conservative to account for different hardware and testing conditions. The race detector significantly impacts performance, so lower baselines are used when it's enabled.

## Interpreting Test Results

### Coverage Report
- **Target**: 100% line coverage
- **Files**: All `.go` files except `examples/`
- **Missing coverage**: Indicates untested code paths

### Benchmark Results
```
BenchmarkTask_Throughput-8    1000000    1234 ns/op    56 B/op    2 allocs/op
```
- `1000000`: Number of iterations
- `1234 ns/op`: Nanoseconds per operation
- `56 B/op`: Bytes allocated per operation  
- `2 allocs/op`: Number of allocations per operation

### Race Detector Output
- **No output**: No race conditions detected
- **Warning messages**: Potential race conditions found

## Continuous Integration

For CI environments, use:

```bash
# Complete test suite for CI
go test -race -cover -v ./...

# Performance regression check
go test -run="TestPerformanceRegression" ./...

# Quick validation (skips long-running tests)
go test -short -v ./...
```

## Troubleshooting

### High Memory Usage
If tests consume too much memory:
```bash
# Run with memory limit
GOMAXPROCS=2 go test -v ./...

# Check for memory leaks
go test -memprofile=mem.prof -bench=. ./...
```

### Slow Test Execution  
If tests run slowly:
```bash
# Parallel execution
go test -parallel 4 -v ./...

# Skip performance benchmarks
go test -v -run="^(Test)" ./...
```

### Race Conditions
If race detector finds issues:
```bash
# Get detailed race condition reports
go test -race -v ./... 2>&1 | tee race_report.txt
```

## Best Practices

1. **Always run with race detector** when testing concurrent code
2. **Monitor performance regression** with regular benchmark runs
3. **Verify 100% coverage** before code changes
4. **Test with various GOMAXPROCS** values to ensure scalability
5. **Profile memory usage** under high load scenarios
