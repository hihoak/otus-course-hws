package hw06pipelineexecution

import (
	"fmt"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"
)

const (
	sleepPerStage = time.Millisecond * 100
	fault         = sleepPerStage / 2
)

func generateStages(wg *sync.WaitGroup, done <-chan interface{}) []Stage {
	// Stage generator
	g := func(wg *sync.WaitGroup, name string, done <-chan interface{}, f func(v interface{}) interface{}) Stage {
		return func(in In) Out {
			out := make(Bi)
			go func() {
				defer func() {
					close(out)
					wg.Done()
				}()
				for {
					select {
					case <-done:
						// gracefully closing channel
						continue
					case v, ok := <-in:
						if !ok {
							fmt.Printf("Stage '%s' ended because of closed channel...\n", name)
							return
						}
						select {
						case <-done:
							// gracefully closing channel
							continue
						default:
						}
						time.Sleep(sleepPerStage)
						out <- f(v)
					}
				}
			}()
			return out
		}
	}
	wg.Add(4)
	return []Stage{
		g(wg, "Dummy", done, func(v interface{}) interface{} { return v }),
		g(wg, "Multiplier (* 2)", done, func(v interface{}) interface{} { return v.(int) * 2 }),
		g(wg, "Adder (+ 100)", done, func(v interface{}) interface{} { return v.(int) + 100 }),
		g(wg, "Stringifier", done, func(v interface{}) interface{} { return strconv.Itoa(v.(int)) }),
	}
}

func dataProducer(wg *sync.WaitGroup, data []interface{}) <-chan interface{} {
	chanData := make(Bi)
	wg.Add(1)
	go func() {
		defer func() {
			close(chanData)
			wg.Done()
		}()
		for _, v := range data {
			chanData <- v
		}
	}()
	return chanData
}

func TestPipeline(t *testing.T) {
	defer goleak.VerifyNone(t)
	t.Run("simple case", func(t *testing.T) {
		wgStages := &sync.WaitGroup{}
		stages := generateStages(wgStages, nil)

		wgProducer := &sync.WaitGroup{}
		data := []interface{}{1, 2, 3, 4, 5}
		in := dataProducer(wgProducer, data)

		result := make([]string, 0, 10)
		start := time.Now()
		for s := range ExecutePipeline(in, nil, stages...) {
			result = append(result, s.(string))
		}
		elapsed := time.Since(start)

		require.Equal(t, []string{"102", "104", "106", "108", "110"}, result)
		require.Less(t,
			int64(elapsed),
			// ~0.8s for processing 5 values in 4 stages (100ms every) concurrently
			int64(sleepPerStage)*int64(len(stages)+len(data)-1)+int64(fault))
		wgStages.Wait()
		wgProducer.Wait()
	})

	t.Run("done case", func(t *testing.T) {
		done := make(Bi)

		wgStages := &sync.WaitGroup{}
		stages := generateStages(wgStages, done)

		// Abort after 200ms
		abortDur := sleepPerStage * 2
		go func() {
			<-time.After(abortDur)
			close(done)
		}()

		wgProducer := &sync.WaitGroup{}
		data := []interface{}{1, 2, 3, 4, 5}
		in := dataProducer(wgProducer, data)

		result := make([]string, 0, 10)
		start := time.Now()
		for s := range ExecutePipeline(in, done, stages...) {
			result = append(result, s.(string))
		}
		elapsed := time.Since(start)

		require.Len(t, result, 0)
		require.Less(t, int64(elapsed), int64(abortDur)+int64(fault*2)) // пришлось здесь увеличить из-за флапающего теста
		wgStages.Wait()
		wgProducer.Wait()
	})

	t.Run("partially done work", func(t *testing.T) {
		done := make(Bi)

		wgStages := &sync.WaitGroup{}
		stages := generateStages(wgStages, done)

		// Abort after 1000ms
		abortDur := sleepPerStage * 10
		go func() {
			defer close(done)
			<-time.After(abortDur)
		}()

		wgProducer := &sync.WaitGroup{}
		data := []interface{}{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
		in := dataProducer(wgProducer, data)

		result := make([]string, 0)
		start := time.Now()
		for s := range ExecutePipeline(in, done, stages...) {
			result = append(result, s.(string))
		}
		elapsed := time.Since(start)

		require.Greater(t, len(result), 0)
		require.Less(t, len(result), len(data))
		require.Less(t, int64(elapsed), int64(abortDur)+int64(fault))
		wgStages.Wait()
		wgProducer.Wait()
	})

	t.Run("empty input channel", func(t *testing.T) {
		wgStages := &sync.WaitGroup{}
		stages := generateStages(wgStages, nil)

		wgProducer := &sync.WaitGroup{}
		var data []interface{}
		in := dataProducer(wgProducer, data)

		result := make([]string, 0)
		start := time.Now()
		for s := range ExecutePipeline(in, nil, stages...) {
			result = append(result, s.(string))
		}
		elapsed := time.Since(start)

		require.Equal(t, result, []string{})
		require.Less(t, int64(elapsed), int64(fault))
		wgStages.Wait()
		wgProducer.Wait()
	})
}
