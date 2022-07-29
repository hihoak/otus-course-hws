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
				fmt.Println("Closing data channel...")
				close(dataChan)
			}()
			for {
				select {
				case <-done:
					fmt.Println("Got stop signal")
					return
				case value, ok := <-in:
					if !ok {
						fmt.Println("In channel is closed")
						return
					}
					select {
					case <-done:
						fmt.Println("Got stop signal")
						return
					case dataChan <- value:
					}
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
