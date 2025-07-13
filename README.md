<div align="center">

# 🚀 GoTask

### ⚡ Concurrent Task Execution Library for Go

*🎯 A simple Go library for managing concurrent goroutines with concurrency limits and error handling*

</div>

---

## 🌟 Why GoTask?

GoTask provides two interfaces for concurrent execution: 
- **`Task`** - Simple concurrent execution with concurrency control
- **`ErrorTask`** - Error-aware execution with context cancellation

Designed for applications that need controlled concurrency and proper error handling. 💪

## ✨ Features

<table>
<tr>
<td>

### 🎛️ **Controlled Concurrency**
Limit simultaneously running goroutines with configurable defaults

</td>
<td>

### ⚡ **Task Management** 
Structured approach to concurrent task execution

</td>
</tr>
<tr>
<td>

### 🛡️ **Error Handling**
Built-in error propagation and context cancellation

</td>
<td>

### 📊 **Task Monitoring**
Track active and submitted tasks with counters

</td>
</tr>
<tr>
<td>

### 🔧 **Context Support**
Full `context.Context` integration for cancellation

</td>
<td>

### 🪶 **Simple API**
Clean interface with no external dependencies

</td>
</tr>
</table>

## 📦 Installation

```bash
go get github.com/fikri240794/gotask
```

## 🚀 Quick Start

<details>
<summary><b>🎯 Basic Task Usage</b> - Click to expand</summary>

```go
package main

import (
    "fmt"
    "time"
    "github.com/fikri240794/gotask"
)

func main() {
    // 🎛️ Create task with concurrency limit of 2
    task := gotask.NewTask(2)

    // 📋 Submit multiple tasks
    for i := 0; i < 5; i++ {
        taskNum := i + 1
        task.Go(func() {
            fmt.Printf("🏃 Task %d executing\n", taskNum)
            time.Sleep(100 * time.Millisecond)
        })
    }

    // ⏳ Wait for all tasks to complete
    task.Wait()
    fmt.Println("✅ All tasks completed!")
    
    // 📊 Check statistics
    fmt.Printf("📈 Total submitted: %d\n", task.TotalSubmitted())
}
```

**Output:**
```
🏃 Task 1 executing
🏃 Task 2 executing
🏃 Task 3 executing
🏃 Task 4 executing
🏃 Task 5 executing
✅ All tasks completed!
📈 Total submitted: 5
```

</details>

<details>
<summary><b>🛡️ ErrorTask with Smart Error Handling</b> - Click to expand</summary>

```go
package main

import (
    "context"
    "fmt"
    "errors"
    "github.com/fikri240794/gotask"
)

func main() {
    // 🎯 Create error-aware task with context
    errTask, taskCtx := gotask.NewErrorTask(context.Background(), 3)

    // 📋 Submit tasks that may fail
    for i := 0; i < 5; i++ {
        taskNum := i + 1
        errTask.Go(func() error {
            // 🔍 Use taskCtx to respect cancellation
            select {
            case <-taskCtx.Done():
                fmt.Printf("❌ Task %d cancelled\n", taskNum)
                return nil // Cancelled gracefully
            default:
                if taskNum == 3 {
                    fmt.Printf("💥 Task %d failed!\n", taskNum)
                    return errors.New("simulated failure")
                }
                fmt.Printf("✅ Task %d completed\n", taskNum)
                return nil
            }
        })
    }

    // ⏳ Wait and handle errors
    if err := errTask.Wait(); err != nil {
        fmt.Printf("🚨 Error occurred: %v\n", err)
        fmt.Printf("📊 Total submitted: %d\n", errTask.TotalSubmitted())
    }
}
```

**Output:**
```
✅ Task 1 completed
✅ Task 2 completed
💥 Task 3 failed!
❌ Task 4 cancelled
❌ Task 5 cancelled
🚨 Error occurred: simulated failure
📊 Total submitted: 5
```

</details>

## 📖 API Reference

<table>
<tr>
<td width="50%">

### 🎯 **Task Interface**

```go
type Task interface {
    // 🚀 Execute function in goroutine
    Go(task func())
    
    // ⏳ Wait for all tasks to complete
    Wait()
    
    // 📊 Get current active tasks
    ActiveCount() int64
    
    // 📈 Get total submitted tasks  
    TotalSubmitted() int64
}
```

**💡 Perfect for:** Simple concurrent operations without error handling

</td>
<td width="50%">

### 🛡️ **ErrorTask Interface**

```go
type ErrorTask interface {
    // 🚀 Execute function with error handling
    Go(task func() error)
    
    // ⏳ Wait until completion or first error
    Wait() error
    
    // 📊 Get current active tasks
    ActiveCount() int64
    
    // 📈 Get total submitted tasks
    TotalSubmitted() int64
    
    // 🔧 Get cancellable context
    Context() context.Context
}
```

**💡 Perfect for:** Error-aware operations with cancellation support

</td>
</tr>
</table>

---
