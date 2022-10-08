package storageerrors

import (
	"github.com/pkg/errors"
)

var (
	ErrNotFoundEvent         = errors.New("event is not found in storage")
	ErrAlreadyExistsEvent    = errors.New("event already exists")
	ErrConnectionFailed      = errors.New("Failed to connect to database")
	ErrPingFailed            = errors.New("Failed to ping database")
	ErrCloseConnectionFailed = errors.New("Failed to close connection to database")
	ErrAddEvent              = errors.New("Failed to add event to database")
	ErrUpdateEvent           = errors.New("Failed to update event in database")
	ErrDeleteEvent           = errors.New("Failed to delete event from database")
	ErrGetEvent              = errors.New("Failed to get event from database")
	ErrListEvents            = errors.New("Failed to list events from database")
)
