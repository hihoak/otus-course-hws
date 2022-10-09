package storageerrors

import "fmt"

type ErrNotFoundEvent struct {
	ID string
}

func (e ErrNotFoundEvent) Error() string {
	return fmt.Sprintf("event '%s' is not found in storage", e.ID)
}

type ErrConnectionFailed struct {
	Err error
}

func (e ErrConnectionFailed) Error() string {
	return fmt.Sprintf("Failed to connect to database: %s", e.Err.Error())
}

type ErrPingFailed struct {
	Err error
}

func (e ErrPingFailed) Error() string {
	return fmt.Sprintf("Failed to ping database: %s", e.Err.Error())
}

type ErrCloseConnectionFailed struct {
	Err error
}

func (e ErrCloseConnectionFailed) Error() string {
	return fmt.Sprintf("Failed to close connection to database: %s", e.Err.Error())
}

type ErrAddEvent struct {
	Err error
}

func (e ErrAddEvent) Error() string {
	return fmt.Sprintf("Failed to add event to database: %s", e.Err.Error())
}

type ErrUpdateEvent struct {
	Err error
}

func (e ErrUpdateEvent) Error() string {
	return fmt.Sprintf("Failed to update event in database: %s", e.Err.Error())
}

type ErrDeleteEvent struct {
	Err error
}

func (e ErrDeleteEvent) Error() string {
	return fmt.Sprintf("Failed to delete event from database: %s", e.Err.Error())
}

type ErrGetEvent struct {
	Err error
}

func (e ErrGetEvent) Error() string {
	return fmt.Sprintf("Failed to get event from database: %s", e.Err.Error())
}

type ErrListEvents struct {
	Err error
}

func (e ErrListEvents) Error() string {
	return fmt.Sprintf("Failed to list events from database: %s", e.Err.Error())
}
