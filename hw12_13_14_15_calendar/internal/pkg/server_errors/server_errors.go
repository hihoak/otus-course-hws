package servererrors

import "fmt"

type ErrStartServer struct {
	Err error
}

func (e ErrStartServer) Error() string {
	return fmt.Sprintf("failed to start server: %v", e.Err.Error())
}
