package gotask

import (
	"runtime"
	"sync"
	"testing"
)

func TestTask_Concurrent(t *testing.T) {
	testCases := []struct {
		Name              string
		MaxConcurrentTask int
		TaskLen           int
	}{
		{
			Name:              "max concurrent task parameter is 0",
			MaxConcurrentTask: 0,
			TaskLen:           8,
		},
		{
			Name:              "max concurrent task parameter is 1",
			MaxConcurrentTask: 1,
			TaskLen:           16,
		},
		{
			Name:              "max concurrent task parameter greater than 1",
			MaxConcurrentTask: 2,
			TaskLen:           32,
		},
	}

	for i := 0; i < len(testCases); i++ {
		t.Run(testCases[i].Name, func(t *testing.T) {
			var (
				actualTask                Task
				expectedMaxConcurrentTask int
				actualActiveTaskCount     int
				actualTaskCount           int
				syncMutex                 sync.Mutex
			)

			expectedMaxConcurrentTask = testCases[i].MaxConcurrentTask
			actualTask = NewTask(expectedMaxConcurrentTask)

			if expectedMaxConcurrentTask < 1 {
				expectedMaxConcurrentTask = runtime.NumCPU()
			}

			for j := 0; j < testCases[i].TaskLen; j++ {
				actualTask.Go(func() {
					defer func() {
						syncMutex.Lock()
						actualActiveTaskCount--
						syncMutex.Unlock()
					}()

					syncMutex.Lock()
					actualTaskCount++
					syncMutex.Unlock()

					syncMutex.Lock()
					actualActiveTaskCount++
					syncMutex.Unlock()
					if expectedMaxConcurrentTask < actualActiveTaskCount {
						t.Errorf("expected maximum task active: %d, got: %d", expectedMaxConcurrentTask, actualActiveTaskCount)
					}
				})
			}

			actualTask.Wait()

			if testCases[i].TaskLen != actualTaskCount {
				t.Errorf("expected task count: %d, got: %d", testCases[i].TaskLen, actualTaskCount)
			}
		})
	}
}
