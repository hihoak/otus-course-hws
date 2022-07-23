package hw06pipelineexecution

import "fmt"

type (
	In  = <-chan interface{}
	Out = In
	Bi  = chan interface{}
)

type Stage func(in In) (out Out)

func ExecutePipeline(in In, done In, stages ...Stage) Out {
	data := func() In {
		dataChan := make(Bi)
		go func() {
			defer func() {
				close(dataChan)
			}()
			for value := range in {
				select {
				case <-done:
					fmt.Printf("Close channel because of stop signal...")
					return
				case dataChan <- value:
				}
			}
		}()
		return dataChan
	}()

	for idx, stage := range stages {
		data = stage(data)
		fmt.Printf("Started stage %d/%d\n", idx+1, len(stages))
	}

	return data
}
