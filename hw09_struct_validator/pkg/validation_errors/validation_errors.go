package validationerrors

import (
	"fmt"
	"strings"
)

type ValidationError struct {
	Field string
	Err   error
}

func (v ValidationError) String() string {
	if v.Field != "" {
		return fmt.Sprintf("wrong field '%s': %v", v.Field, v.Err)
	}
	return v.Err.Error()
}

func (v ValidationError) Error() string {
	return v.String()
}

type ValidationErrors []ValidationError

func (v ValidationErrors) Error() string {
	output := strings.Builder{}
	for idx, err := range v {
		output.WriteString(fmt.Sprintf("%d) %s\n", idx+1, err.String()))
	}
	return output.String()
}
