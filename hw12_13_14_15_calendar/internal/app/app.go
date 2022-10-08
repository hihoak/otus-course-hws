package app

import (
	"context"
	"fmt"

	storageerrors "github.com/hihoak/otus-course-hws/hw12_13_14_15_calendar/internal/pkg/storage_errors"
	"github.com/hihoak/otus-course-hws/hw12_13_14_15_calendar/internal/storage"
	desc "github.com/hihoak/otus-course-hws/hw12_13_14_15_calendar/pkg/api/event"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

//go:generate mockgen -destination mocks/apps_mocks.go -source app.go -package appsmocks
type App struct {
	desc.EventServiceServer
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

func ConvertEventToPb(event *storage.Event) *desc.Event {
	return &desc.Event{
		Id:    event.ID,
		Title: event.Title,
	}
}

func ConvertEventsToPb(events []*storage.Event) []*desc.Event {
	res := make([]*desc.Event, len(events))
	for idx, event := range events {
		res[idx] = ConvertEventToPb(event)
	}
	return res
}

func (a *App) CreateEvent(ctx context.Context, req *desc.AddEventRequest) (*desc.Empty, error) {
	a.Logg.Info().Msg("CreateEvent - start creating event")
	err := a.Store.AddEvent(ctx, &storage.Event{
		ID:    req.GetId(),
		Title: req.GetTitle(),
	})
	if err != nil {
		a.Logg.Error().Err(err).Msgf("Can't create event with ID '%s'", req.GetId())
		return &desc.Empty{}, errors.Wrap(err, fmt.Sprintf("Can't create event with ID '%s'", req.GetId()))
	}
	a.Logg.Info().Msgf("Successfully create event with ID '%s'", req.GetId())
	return &desc.Empty{}, nil
}

func (a *App) GetEvent(ctx context.Context, req *desc.GetEventRequest) (*desc.GetEventResponse, error) {
	a.Logg.Info().Msg("GetEvent - start getting event")
	event, err := a.Store.GetEvent(ctx, req.GetId())
	if err != nil {
		a.Logg.Error().Err(err).Msgf("Can't get event with ID '%s'", req.GetId())
		if errors.Is(err, storageerrors.ErrNotFoundEvent) {
			return nil,
				status.Error(codes.NotFound,
					errors.Wrap(err, fmt.Sprintf("Can't find event with ID '%s'", req.GetId())).Error())
		}
		return nil,
			status.Error(codes.Internal,
				errors.Wrap(err, fmt.Sprintf("Can't get event with ID '%s'", req.GetId())).Error())
	}
	a.Logg.Info().Msgf("Successfully get event with ID '%s'", req.GetId())
	return &desc.GetEventResponse{
		Event: ConvertEventToPb(event),
	}, nil
}

func (a *App) DeleteEvent(ctx context.Context, req *desc.DeleteEventRequest) (*desc.Empty, error) {
	a.Logg.Info().Msgf("DeleteEvent - start deleting event with id '%s'", req.GetId())
	err := a.Store.DeleteEvent(ctx, req.GetId())
	if err != nil {
		return nil,
			status.Error(codes.Internal,
				errors.Wrap(err, fmt.Sprintf("Can't delete event with ID '%s'", req.GetId())).Error())
	}
	a.Logg.Info().Msgf("Successfully delete event with ID '%s'", req.GetId())
	return &desc.Empty{}, nil
}

func (a *App) ListEvent(ctx context.Context, req *desc.ListEventRequest) (*desc.ListEventResponse, error) {
	a.Logg.Info().Msg("ListEvent - start listing events")
	events, err := a.Store.ListEvents(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, errors.Wrap(err, "Can't list events").Error())
	}
	a.Logg.Info().Msg("Successfully list events")
	return &desc.ListEventResponse{
		Events: ConvertEventsToPb(events),
	}, nil
}

func (a *App) ModifyEvent(ctx context.Context, req *desc.ModifyEventRequest) (*desc.Empty, error) {
	a.Logg.Info().Msg("ModifyEvent - start listing events")
	err := a.Store.ModifyEvent(ctx, &storage.Event{
		ID:    req.GetId(),
		Title: req.GetTitle(),
	})
	if err != nil {
		return nil,
			status.Error(codes.Internal,
				errors.Wrap(err, fmt.Sprintf("Can't modify event with id '%s'", req.GetId())).Error())
	}
	a.Logg.Info().Msgf("Successfully modify event with id '%s'", req.GetId())
	return &desc.Empty{}, nil
}
