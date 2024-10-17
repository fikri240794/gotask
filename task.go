package gotask

import (
	"runtime"
	"sync"
)

type Task interface {
	Go(task func())
	Wait()
}

type task struct {
	wg sync.WaitGroup
	c  chan struct{}
}

func NewTask(maxConcurrentTask int) Task {
	if maxConcurrentTask < 1 {
		maxConcurrentTask = runtime.NumCPU()
	}

	return &task{
		c: make(chan struct{}, maxConcurrentTask),
	}
}

func (t *task) Go(task func()) {
	t.c <- struct{}{}
	t.wg.Add(1)

	go func(taskToDo func()) {
		defer func() {
			<-t.c
			t.wg.Done()
		}()

		taskToDo()
	}(task)
}

func (t *task) Wait() {
	t.wg.Wait()
}
