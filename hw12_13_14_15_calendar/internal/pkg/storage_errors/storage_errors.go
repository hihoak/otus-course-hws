package storageerrors

import (
	"errors"
)

var (
	ErrNotFoundEvent         = errors.New("event is not found in storage")
	ErrAlreadyExistsEvent    = errors.New("event already exists")
	ErrConnectionFailed      = errors.New("failed to connect to database")
	ErrPingFailed            = errors.New("failed to ping database")
	ErrCloseConnectionFailed = errors.New("failed to close connection to database")
	ErrAddEvent              = errors.New("failed to add event to database")
	ErrUpdateEvent           = errors.New("failed to update event in database")
	ErrDeleteEvent           = errors.New("failed to delete event from database")
	ErrGetEvent              = errors.New("failed to get event from database")
	ErrListEvents            = errors.New("failed to list events from database")
)
