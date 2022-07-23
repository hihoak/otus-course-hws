package hw05parallelexecution

import (
	"errors"
	"sync"
)

var ErrErrorsLimitExceeded = errors.New("errors limit exceeded")

type Task func() error

// Run starts tasks in n goroutines and stops its work when receiving m errors from tasks.
func Run(tasks []Task, n, m int) error {
	// thanks for 2 solution to understand how to fix data race in simple 'recover' realisation of closing pipes
	// https://urlis.net/aua6y
	tasksChan := make(chan Task)
	errsChan := make(chan error)
	stopChan := make(chan struct{})

	// filling chan with Tasks in goroutine for not to waste time
	// declaring Senders
	wgSenders := &sync.WaitGroup{}
	wgSenders.Add(1)
	go sender(wgSenders, tasks, tasksChan, stopChan)

	// declaring Workers
	wgWorkers := &sync.WaitGroup{}
	for i := 0; i < n; i++ {
		wgWorkers.Add(1)
		go worker(wgWorkers, tasksChan, errsChan, stopChan)
	}

	// declaring Receivers
	// creating resChan, that will protect as from Data Races, one of graceful pipes closings
	resChan := make(chan error, 1)
	wgReceivers := &sync.WaitGroup{}
	wgReceivers.Add(1)
	go receiver(wgReceivers, resChan, errsChan, stopChan, len(tasks), m)

	// waiting for all goroutines
	wgReceivers.Wait()
	wgWorkers.Wait()
	wgSenders.Wait()

	return <-resChan
}

func sender(wg *sync.WaitGroup, tasks []Task, tasksChan chan<- Task, stopChan <-chan struct{}) {
	defer func() {
		wg.Done()
		close(tasksChan)
	}()

	for _, task := range tasks {
		select {
		case <-stopChan:
			return
		case tasksChan <- task:
		}
	}
}

func worker(wg *sync.WaitGroup, tasksChan <-chan Task, errsChan chan<- error, stopChan <-chan struct{}) {
	defer wg.Done()

	for task := range tasksChan {
		select {
		case <-stopChan:
			return
		case errsChan <- task():
		}
	}
}

func receiver(wg *sync.WaitGroup, resChan chan<- error, errsChan <-chan error, stopChan chan<- struct{},
	totalTasks int, maxErrors int,
) {
	defer func() {
		close(stopChan)
		wg.Done()
	}()
	errCounter := 0
	doneTasks := 0
	for err := range errsChan {
		if err != nil {
			errCounter++
		}
		// if m <= 0 then we don't care about number of errors
		if maxErrors > 0 && errCounter == maxErrors {
			resChan <- ErrErrorsLimitExceeded
			return
		}
		// counting done tasks
		doneTasks++
		if doneTasks == totalTasks {
			break
		}
	}
	resChan <- nil
}
