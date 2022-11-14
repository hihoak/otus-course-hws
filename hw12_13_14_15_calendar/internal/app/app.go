package app

import (
	"context"

	"github.com/hihoak/otus-course-hws/hw12_13_14_15_calendar/internal/storage"
	"github.com/rs/zerolog"
)

type App struct {
	Logg  Logger
	Store Storage
}

type Logger interface {
	Info() *zerolog.Event
	Error() *zerolog.Event
	Warn() *zerolog.Event
	Debug() *zerolog.Event
	Fatal() *zerolog.Event
}

type Storage interface {
	AddEvent(ctx context.Context, event *storage.Event) error
	ModifyEvent(ctx context.Context, event *storage.Event) error
	DeleteEvent(ctx context.Context, id string) error
	GetEvent(ctx context.Context, id string) (*storage.Event, error)
	ListEvents(ctx context.Context) ([]*storage.Event, error)
}

func New(logger Logger, storage Storage) *App {
	return &App{
		Logg:  logger,
		Store: storage,
	}
}

func (a *App) CreateEvent(ctx context.Context, id, title string) error {
	// TODO
	return nil
	// return a.storage.CreateEvent(storage.Event{ID: id, Title: title})
}

// TODO
