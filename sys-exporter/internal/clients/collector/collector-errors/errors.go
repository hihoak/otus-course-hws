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

type MultiError struct {
	fails []*ExportError

	mu *sync.Mutex
}

func NewMultiError() *MultiError {
	return &MultiError{
		mu:    &sync.Mutex{},
		fails: make([]*ExportError, 0),
	}
}

func (e *MultiError) Append(err *ExportError) {
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

func (e *MultiError) Len() int {
	return len(e.fails)
}

func (e *ExportError) String() string {
	return fmt.Sprintf("%s - %s\n", e.FuncName, e.Reason)
}
