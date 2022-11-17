package collectorerrors

import (
	"fmt"
	"strings"
	"sync"
)

type ExportError struct {
	FuncName string
	Reason   string
}

func (e *ExportError) String() string {
	return fmt.Sprintf("%s - %s\n", e.FuncName, e.Reason)
}

type MultiError struct {
	fails []*ExportError

	mu *sync.Mutex
}

func (e *MultiError) Append(err *ExportError) {
	if e.mu == nil {
		e.mu = &sync.Mutex{}
	}
	e.mu.Lock()
	e.fails = append(e.fails, err)
	e.mu.Unlock()
}

func (e *MultiError) Error() string {
	res := strings.Builder{}
	for _, f := range e.fails {
		res.WriteString(f.String())
	}
	return res.String()
}
