//nolint:gci
package hw05parallelexecution

import (
	"errors"
	"fmt"
	"math/rand"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"go.uber.org/goleak"

	"github.com/stretchr/testify/require"
)

func TestRun(t *testing.T) {
	defer goleak.VerifyNone(t)
	rand.Seed(time.Now().UnixNano())

	t.Run("if were errors in first M tasks, than finished not more N+M tasks", func(t *testing.T) {
		tasksCount := 50
		tasks := make([]Task, 0, tasksCount)

		var runTasksCount int32

		for i := 0; i < tasksCount; i++ {
			err := fmt.Errorf("error from task %d", i)
			tasks = append(tasks, func() error {
				time.Sleep(time.Millisecond * time.Duration(rand.Intn(100)))
				atomic.AddInt32(&runTasksCount, 1)
				return err
			})
		}

		workersCount := 10
		maxErrorsCount := 23
		err := Run(tasks, workersCount, maxErrorsCount)

		require.Truef(t, errors.Is(err, ErrErrorsLimitExceeded), "actual err - %v", err)
		require.LessOrEqual(t, runTasksCount, int32(workersCount+maxErrorsCount), "extra tasks were started")
	})

	t.Run("tasks without errors", func(t *testing.T) {
		tasksCount := 50
		tasks := make([]Task, 0, tasksCount)

		var runTasksCount int32
		var sumTime time.Duration

		for i := 0; i < tasksCount; i++ {
			taskSleep := time.Millisecond * time.Duration(rand.Intn(100))
			sumTime += taskSleep

			tasks = append(tasks, func() error {
				time.Sleep(taskSleep)
				atomic.AddInt32(&runTasksCount, 1)
				return nil
			})
		}

		workersCount := 5
		maxErrorsCount := 1

		start := time.Now()
		err := Run(tasks, workersCount, maxErrorsCount)
		elapsedTime := time.Since(start)
		require.NoError(t, err)

		require.Equal(t, runTasksCount, int32(tasksCount), "not all tasks were completed")
		require.LessOrEqual(t, int64(elapsedTime), int64(sumTime/2), "tasks were run sequentially?")
	})

	t.Run("Additional test. PossibleErrors == 0", func(t *testing.T) {
		tasksCount := 50
		tasks := make([]Task, 0, tasksCount)

		var runTasksCount int32
		var sumTime time.Duration

		for i := 0; i < tasksCount; i++ {
			taskSleep := time.Millisecond * time.Duration(rand.Intn(100))
			sumTime += taskSleep

			tasks = append(tasks, func() error {
				time.Sleep(taskSleep)
				atomic.AddInt32(&runTasksCount, 1)
				return fmt.Errorf("some error")
			})
		}

		workersCount := 100
		possibleErrors := 0

		err := Run(tasks, workersCount, possibleErrors)
		require.NoError(t, err)
		require.Equal(t, runTasksCount, int32(tasksCount))
	})

	t.Run("Additional test. PossibleErrors < 0", func(t *testing.T) {
		tasksCount := 50
		tasks := make([]Task, 0, tasksCount)

		var runTasksCount int32
		var sumTime time.Duration

		for i := 0; i < tasksCount; i++ {
			taskSleep := time.Millisecond * time.Duration(rand.Intn(100))
			sumTime += taskSleep

			tasks = append(tasks, func() error {
				time.Sleep(taskSleep)
				atomic.AddInt32(&runTasksCount, 1)
				return fmt.Errorf("some error")
			})
		}

		workersCount := 100
		possibleErrors := -100

		err := Run(tasks, workersCount, possibleErrors)
		require.NoError(t, err)
		require.Equal(t, runTasksCount, int32(tasksCount))
	})

	t.Run("tasks without errors. Rewrite without time.Sleep", func(t *testing.T) {
		tasksCount := 50
		tasks := make([]Task, 0, tasksCount)

		var runTasksCount int32

		testMutex := &sync.Mutex{}
		var sumTime time.Duration
		for i := 0; i < tasksCount; i++ {
			taskSleep := time.Millisecond * time.Duration(rand.Intn(100))
			sumTime += taskSleep

			tasks = append(tasks, func() error {
				time.Sleep(taskSleep)
				testMutex.Lock()
				runTasksCount++
				testMutex.Unlock()
				return nil
			})
		}

		workersCount := 5
		maxErrorsCount := 1

		testWaitGroup := &sync.WaitGroup{}
		var err error
		testWaitGroup.Add(1)
		go func() {
			defer testWaitGroup.Done()
			err = Run(tasks, workersCount, maxErrorsCount)
		}()
		require.NoError(t, err)

		require.Eventually(t, func() bool {
			defer testMutex.Unlock()
			testMutex.Lock()
			return runTasksCount == int32(tasksCount)
		}, sumTime/2, time.Millisecond*10,
			"not all tasks were completed, maybe tasks were run sequentially?")
		testWaitGroup.Wait()
	})
}
