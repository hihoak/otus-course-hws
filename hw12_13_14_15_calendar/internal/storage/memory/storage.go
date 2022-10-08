package memorystorage

import (
	"context"
	"fmt"
	"sync"

	"github.com/hihoak/otus-course-hws/hw12_13_14_15_calendar/internal/app"
	"github.com/hihoak/otus-course-hws/hw12_13_14_15_calendar/internal/logger"
	errs "github.com/hihoak/otus-course-hws/hw12_13_14_15_calendar/internal/pkg/storage_errors"
	"github.com/hihoak/otus-course-hws/hw12_13_14_15_calendar/internal/storage"
	"github.com/pkg/errors"
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

func (s *Storage) AddEvent(ctx context.Context, event *storage.Event) error {
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
		return nil, errors.Wrap(errs.ErrNotFoundEvent, fmt.Sprintf("Can't find event with id %s", id))
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
