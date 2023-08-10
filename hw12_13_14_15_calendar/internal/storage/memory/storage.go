package memorystorage

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/hihoak/otus-course-hws/hw12_13_14_15_calendar/internal/app"
	"github.com/hihoak/otus-course-hws/hw12_13_14_15_calendar/internal/logger"
	errs "github.com/hihoak/otus-course-hws/hw12_13_14_15_calendar/internal/pkg/storage_errors"
	"github.com/hihoak/otus-course-hws/hw12_13_14_15_calendar/internal/storage"
	"github.com/rs/xid"
)

type Storage struct {
	app.Storage
	data map[string]*storage.Event

	mu  sync.RWMutex
	log app.Logger
}

func New(log *logger.Logger) *Storage {
	return &Storage{
		data: make(map[string]*storage.Event),
		mu:   sync.RWMutex{},
		log:  log,
	}
}

func (s *Storage) Connect(ctx context.Context) error {
	return nil
}

func (s *Storage) Close(ctx context.Context) error {
	return nil
}

func (s *Storage) AddEvent(ctx context.Context, title string, notifyDate, timeNow time.Time) error {
	event := &storage.Event{
		Title: title,
	}
	event.ID = xid.New().String()
	s.log.Debug().Msgf("Start adding event with id %s", event.ID)
	s.mu.Lock()
	s.data[event.ID] = event
	s.mu.Unlock()
	s.log.Debug().Msgf("Successfully add event with id %s", event.ID)
	return nil
}

func (s *Storage) ModifyEvent(ctx context.Context, event *storage.Event) error {
	s.log.Debug().Msgf("Start modifying event with id %s", event.ID)
	s.mu.Lock()
	s.data[event.ID] = event
	s.mu.Unlock()
	s.log.Debug().Msgf("Successfully modified event with id %s", event.ID)
	return nil
}

func (s *Storage) DeleteEvent(ctx context.Context, id string) error {
	s.log.Debug().Msgf("Start deleting event with id %s", id)
	s.mu.Lock()
	delete(s.data, id)
	s.mu.Unlock()
	s.log.Debug().Msgf("Successfully deleted event with id %s", id)
	return nil
}

func (s *Storage) GetEvent(ctx context.Context, id string) (*storage.Event, error) {
	s.log.Debug().Msgf("Start getting event with id %s", id)
	event, ok := s.data[id]
	if !ok {
		return nil, fmt.Errorf("can't find event with id %s: %w", id, errs.ErrNotFoundEvent)
	}
	s.log.Debug().Msgf("Successfully find event with id %s", id)
	return event, nil
}

func (s *Storage) ListEvents(ctx context.Context) ([]*storage.Event, error) {
	s.log.Debug().Msg("Start listing all events")
	events := make([]*storage.Event, 0, len(s.data))
	for _, event := range s.data {
		events = append(events, event)
	}
	s.log.Debug().Msgf("Successfully listed all events, total: %d", len(events))
	return events, nil
}

func (s *Storage) ListEventsToNotify(
	ctx context.Context,
	fromTime time.Time,
	countOfEvents int,
) ([]*storage.Event, error) {
	s.log.Debug().Msg("Start list events to notify")
	res := make([]*storage.Event, 0, countOfEvents)
	for _, event := range s.data {
		if !event.ScheduledToNotify && !event.IsSent && event.NotifyDate.Before(fromTime) {
			res = append(res, event)
		}
	}
	s.log.Debug().Msgf("Got '%d' events to notify", len(res))
	return res, nil
}

func (s *Storage) DeleteOldEventsBeforeTime(
	_ context.Context,
	fromTime time.Time,
	maxLiveTime time.Duration,
) error {
	s.log.Debug().Msg("Start delete old events")
	for key, event := range s.data {
		if !event.EndDate.Before(fromTime.Add(-maxLiveTime)) {
			s.mu.Lock()
			delete(s.data, key)
			s.mu.Unlock()
		}
	}
	s.log.Debug().Msgf("Successfully delete old events")
	return nil
}

func (s *Storage) SetSentStatusToEvents(_ context.Context, ids []string) error {
	s.log.Debug().Msg("Start set statuses to events")
	for _, id := range ids {
		if _, ok := s.data[id]; ok {
			s.data[id].IsSent = true
		}
	}
	s.log.Debug().Msgf("Successfully set sent statuses to events")
	return nil
}
