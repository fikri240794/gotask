# Tasks
Write your goroutine code in simple way.

## Installation
```bash
go get github.com/fikri240794/gotask
```

## Usage
Example how to use Task:
```go
package main

import "github.com/fikri240794/gotask"

func main() {
	var (
		maxConcurrentTask int
		task              gotask.Task
	)

	maxConcurrentTask = 2 // set limit your async gorotine processes
	task = gotask.NewTask(maxConcurrentTask)

	task.Go(func() {
		// task 1
	})
	task.Go(func() {
		// task 2
	})
	task.Go(func() {
		// task 3
	})
	task.Go(func() {
		// task n...
	})

	task.Wait()
}
```

Example how to use ErrorTask:
```go
package main

import (
	"context"
	"github.com/fikri240794/gotask"
)

func main() {
	var (
		ctx               context.Context
		maxConcurrentTask int
		errTask           gotask.ErrorTask
		errTaskCtx        context.Context
		err               error
	)

	ctx = context.Background() // any context from (from param, request, etc...)
	maxConcurrentTask = 2      // set limit your async gorotine processes
	errTask, errTaskCtx = gotask.NewErrorTask(maxConcurrentTask, ctx) // always create new context for errTask

	errTask.Go(func() error {
		// task 1
		someFunc(errTaskCtx, args...) // use errTaskCtx context for all errTask goroutine
	})
	errTask.Go(func() error {
		// task 2
	})
	errTask.Go(func() error {
		// task 3
	})
	errTask.Go(func() error {
		// task n...
	})

	err = errTask.Wait()
	if err != nil {
		log.Fatal(err)
	}
}
```