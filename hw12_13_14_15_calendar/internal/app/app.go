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
	"google.golang.org/genproto/googleapis/type/datetime"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/durationpb"
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
	AddEvent(ctx context.Context, event *storage.Event) error
	ModifyEvent(ctx context.Context, event *storage.Event) error
	DeleteEvent(ctx context.Context, id string) error
	GetEvent(ctx context.Context, id string) (*storage.Event, error)
	ListEvents(ctx context.Context) ([]*storage.Event, error)
	ListEventsForDays(ctx context.Context, date *time.Time, forDays int64) ([]*storage.Event, error)
}

func New(logger Logger, storage Storage) *App {
	return &App{
		Logg:  logger,
		Store: storage,
	}
}

func ConvertEventToPb(event *storage.Event) *desc.Event {
	return &desc.Event{
		Id:          event.ID,
		Title:       event.Title,
		StartDate:   ConvertFromTimeToPbDateTime(event.StartDate),
		EndDate:     ConvertFromTimeToPbDateTime(event.EndDate),
		Description: event.Description,
		UserId:      event.UserID,
		NotifyDate:  ConvertFromTimeToPbDateTime(event.NotifyDate),
	}
}

func ConvertEventsToPb(events []*storage.Event) []*desc.Event {
	res := make([]*desc.Event, len(events))
	for idx, event := range events {
		res[idx] = ConvertEventToPb(event)
	}
	return res
}

func ConvertFromTimeToPbDateTime(t *time.Time) *datetime.DateTime {
	if t == nil {
		return nil
	}
	_, offsetInSeconds := t.Zone()
	return &datetime.DateTime{
		Year:    int32(t.Year()),
		Month:   int32(t.Month()),
		Day:     int32(t.Day()),
		Hours:   int32(t.Hour()),
		Minutes: int32(t.Minute()),
		Seconds: int32(t.Second()),
		Nanos:   int32(t.Nanosecond()),
		TimeOffset: &datetime.DateTime_UtcOffset{
			UtcOffset: &durationpb.Duration{
				Seconds: int64(offsetInSeconds),
				Nanos:   0,
			},
		},
	}
}

func ConvertFromPbDateTimeToTime(pbDateTime *datetime.DateTime) *time.Time {
	if pbDateTime == nil {
		return nil
	}
	goTime := time.Date(
		int(pbDateTime.Year),
		time.Month(pbDateTime.Month),
		int(pbDateTime.Day),
		int(pbDateTime.Hours),
		int(pbDateTime.Minutes),
		int(pbDateTime.Seconds),
		0,
		time.UTC,
	)
	if pbDateTime.GetUtcOffset() != nil {
		goTime.Add(pbDateTime.GetUtcOffset().AsDuration())
	}

	return &goTime
}

func validateAndConvertAddEventRequestToEvent(timeNow *time.Time, event *desc.AddEventRequest) (*storage.Event, error) {
	if event.GetTitle() == "" {
		return nil, fmt.Errorf("title cannot be empty")
	}

	var startDate, endDate *time.Time
	if event.GetEndDate() != nil {
		endDate = ConvertFromPbDateTimeToTime(event.EndDate)
	}

	switch {
	case event.GetStartDate() != nil:
		startDate = ConvertFromPbDateTimeToTime(event.StartDate)
	case endDate != nil:
		startDate = endDate
	default:
		startDate = timeNow
	}

	if endDate != nil && endDate.Before(*startDate) {
		return nil, fmt.Errorf("end_date cannot be before start date")
	}

	if event.GetUserId() == "" {
		return nil, fmt.Errorf("user_id cannot be empty")
	}

	return &storage.Event{
		Title:       event.GetTitle(),
		StartDate:   startDate,
		EndDate:     endDate,
		Description: event.GetDescription(),
		UserID:      event.GetUserId(),
		NotifyDate:  ConvertFromPbDateTimeToTime(event.GetNotifyDate()),
	}, nil
}

func (a *App) CreateEvent(ctx context.Context, req *desc.AddEventRequest) (*desc.Empty, error) {
	a.Logg.Info().Msg("CreateEvent - start creating event")
	timeNow := time.Now()
	event, validationErr := validateAndConvertAddEventRequestToEvent(&timeNow, req)
	if validationErr != nil {
		validationErr = status.Errorf(codes.InvalidArgument, validationErr.Error())
		a.Logg.Error().Err(validationErr).Msg("CreateEvent - failed validation")
		return &desc.Empty{}, validationErr
	}
	err := a.Store.AddEvent(ctx, event)
	if err != nil {
		a.Logg.Error().Err(err).Msgf("Can't create event with title '%s'", event.Title)
		return &desc.Empty{}, fmt.Errorf("can't create event with Title '%s': %w", event.Title, err)
	}
	a.Logg.Info().Msgf("Successfully create event with Title '%s'", event.Title)
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

func (a *App) ListEventForDays(
	ctx context.Context,
	req *desc.ListEventForDaysRequest,
) (*desc.ListEventResponse, error) {
	a.Logg.Info().Msg("ListEvent - start listing events")
	if req.GetDate() == nil {
		return nil, status.Error(codes.InvalidArgument, "empty 'date' field")
	}
	events, err := a.Store.ListEventsForDays(ctx, ConvertFromPbDateTimeToTime(req.GetDate()), req.GetForDays())
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
