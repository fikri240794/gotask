package gotask

import (
	"context"
	"errors"
	"runtime"
	"sync"
	"testing"
)

func TestErrorTask_Concurrent(t *testing.T) {
	testCases := []struct {
		Name              string
		MaxConcurrentTask int
		TaskLen           int
		ExpectationErr    error
	}{
		{
			Name:              "max concurrent task parameter is 0",
			MaxConcurrentTask: 0,
			TaskLen:           8,
			ExpectationErr:    nil,
		},
		{
			Name:              "max concurrent task parameter is 1",
			MaxConcurrentTask: 1,
			TaskLen:           16,
			ExpectationErr:    nil,
		},
		{
			Name:              "max concurrent task parameter greater than 1",
			MaxConcurrentTask: 2,
			TaskLen:           32,
			ExpectationErr:    nil,
		},
		{
			Name:              "max concurrent task parameter greater than 1 with error",
			MaxConcurrentTask: 2,
			TaskLen:           32,
			ExpectationErr:    errors.New("some error"),
		},
	}

	for i := 0; i < len(testCases); i++ {
		t.Run(testCases[i].Name, func(t *testing.T) {
			var (
				actualErrorTask           ErrorTask
				actualErr                 error
				expectedMaxConcurrentTask int
				actualActiveTaskCount     int
				actualTaskCount           int
				syncMutex                 sync.Mutex
			)

			expectedMaxConcurrentTask = testCases[i].MaxConcurrentTask
			actualErrorTask, _ = NewErrorTask(context.Background(), expectedMaxConcurrentTask)

			if expectedMaxConcurrentTask < 1 {
				expectedMaxConcurrentTask = runtime.NumCPU()
			}

			for j := 0; j < testCases[i].TaskLen; j++ {
				actualErrorTask.Go(func() error {
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

					if testCases[i].ExpectationErr != nil {
						return testCases[i].ExpectationErr
					}

					return nil
				})
			}

			actualErr = actualErrorTask.Wait()

			if testCases[i].ExpectationErr == nil && actualErr != nil {
				t.Errorf("expected error is nil, got: %s", actualErr.Error())
			}

			if testCases[i].ExpectationErr != nil && actualErr == nil {
				t.Errorf("expected error is %s, got: nil", testCases[i].ExpectationErr.Error())
			}

			if testCases[i].ExpectationErr != nil && actualErr != nil && testCases[i].ExpectationErr.Error() != actualErr.Error() {
				t.Errorf("expected error is %s, got: %s", testCases[i].ExpectationErr.Error(), actualErr.Error())
			}

			if testCases[i].TaskLen != actualTaskCount && testCases[i].ExpectationErr == nil {
				t.Errorf("expected task count: %d, got: %d", testCases[i].TaskLen, actualTaskCount)
			}

			if actualTaskCount != 1 && testCases[i].ExpectationErr != nil {
				t.Errorf("expected task count: %d, got: %d", 1, actualTaskCount)
			}
		})
	}
}
