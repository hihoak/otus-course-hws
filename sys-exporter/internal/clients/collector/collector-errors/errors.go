package collectorerrors

import (
	"fmt"
	"strings"
	"sync"
)

var multiErrorMu = sync.Mutex{}

type ExportError struct {
	FuncName string
	Reason   string
}

func (e ExportError) String() string {
	return fmt.Sprintf("%s - %s\n", e.FuncName, e.Reason)
}

type MultiError struct {
	fails []*ExportError
}

func (e *MultiError) Append(err *ExportError) {
	multiErrorMu.Lock()
	e.fails = append(e.fails, err)
	multiErrorMu.Unlock()
}

func (e MultiError) Error() string {
	res := strings.Builder{}
	for _, f := range e.fails {
		res.WriteString(f.String())
	}
	return res.String()
}
