package app

import (
	"context"
	"errors"
	"fmt"
	"time"

	storageerrors "github.com/hihoak/otus-course-hws/hw12_13_14_15_calendar/internal/pkg/storage_errors"
	"github.com/hihoak/otus-course-hws/hw12_13_14_15_calendar/internal/storage"
	desc "github.com/hihoak/otus-course-hws/hw12_13_14_15_calendar/pkg/api/event"
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
	// common methods
	Connect(ctx context.Context) error
	Close(ctx context.Context) error

	// Main app eventsa
	AddEvent(ctx context.Context, title string, NotifyDate, timeNow time.Time) error
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
	err := a.Store.AddEvent(ctx, req.GetTitle(), time.Now(), time.Now())
	if err != nil {
		a.Logg.Error().Err(err).Msgf("Can't create event with title '%s'", req.GetTitle())
		return &desc.Empty{}, fmt.Errorf("can't create event with title '%s': %w", req.GetTitle(), err)
	}
	a.Logg.Info().Msgf("Successfully create event with title '%s'", req.GetTitle())
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
					fmt.Errorf("can't find event with ID '%s': %w", req.GetId(), err).Error())
		}
		return nil,
			status.Error(codes.Internal, fmt.Errorf("can't get event with ID '%s': %w", req.GetId(), err).Error())
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
				fmt.Errorf("can't delete event with ID '%s': %w", req.GetId(), err).Error())
	}
	a.Logg.Info().Msgf("Successfully delete event with ID '%s'", req.GetId())
	return &desc.Empty{}, nil
}

func (a *App) ListEvent(ctx context.Context, req *desc.ListEventRequest) (*desc.ListEventResponse, error) {
	a.Logg.Info().Msg("ListEvent - start listing events")
	events, err := a.Store.ListEvents(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal,
			fmt.Errorf("can't list events: %w", err).Error())
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
				fmt.Errorf("can't modify event with id '%s': %w", req.GetId(), err).Error())
	}
	a.Logg.Info().Msgf("Successfully modify event with id '%s'", req.GetId())
	return &desc.Empty{}, nil
}
