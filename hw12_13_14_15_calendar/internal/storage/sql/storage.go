package sqlstorage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/hihoak/otus-course-hws/hw12_13_14_15_calendar/internal/app"
	"github.com/hihoak/otus-course-hws/hw12_13_14_15_calendar/internal/logger"
	errs "github.com/hihoak/otus-course-hws/hw12_13_14_15_calendar/internal/pkg/storage_errors"
	"github.com/hihoak/otus-course-hws/hw12_13_14_15_calendar/internal/storage"
	"github.com/jmoiron/sqlx"
	"github.com/rs/xid"
)

type Storage struct {
	app.Storage
	host     string
	port     string
	user     string
	password string
	dbname   string

	connectionTimeout time.Duration
	operationTimeout  time.Duration

	log app.Logger

	db *sqlx.DB
}

func New(
	log *logger.Logger,
	host, port, user, password, dbname string,
	connectionTimeout, operationTimeout time.Duration,
) *Storage {
	return &Storage{
		host:              host,
		port:              port,
		user:              user,
		password:          password,
		dbname:            dbname,
		connectionTimeout: connectionTimeout,
		operationTimeout:  operationTimeout,

		log: log,
	}
}

func (s *Storage) Connect(ctx context.Context) error {
	s.log.Info().Msgf("Start connection to database %s:%s with timeout %v", s.host, s.port, s.connectionTimeout)
	ctx, cancel := context.WithTimeout(ctx, s.connectionTimeout)
	defer cancel()
	db, err := sqlx.ConnectContext(ctx, "postgres",
		fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
			s.host, s.port, s.user, s.password, s.dbname))
	if err != nil {
		return fmt.Errorf("%s: %w", err.Error(), errs.ErrConnectionFailed)
	}
	err = db.PingContext(ctx)
	if err != nil {
		return fmt.Errorf("%s: %w", err.Error(), errs.ErrPingFailed)
	}
	s.db = db
	s.log.Info().Msg("Successfully connected to database")
	return nil
}

func (s *Storage) Close(ctx context.Context) error {
	s.log.Info().Msg("Start closing connection to database...")
	if err := s.db.Close(); err != nil {
		return fmt.Errorf("%s: %w", err.Error(), errs.ErrCloseConnectionFailed)
	}
	s.log.Info().Msg("Successfully close connection to database")
	return s.db.Close()
}

func (s *Storage) AddEvent(ctx context.Context, title string) error {
	query := `
		INSERT INTO events (id, title)
        VALUES (:id, :title)`
	event := &storage.Event{
		Title: title,
	}
	event.ID = xid.New().String()
	s.log.Debug().Msgf("Start adding event with id %s", event.ID)
	ctx, cancel := context.WithTimeout(ctx, s.connectionTimeout)
	defer cancel()
	_, err := s.db.NamedExecContext(ctx, query, event)
	if err != nil {
		return fmt.Errorf("%s: %w", err.Error(), errs.ErrAddEvent)
	}
	s.log.Debug().Msgf("Successfully add event with id %s", event.ID)
	return err
}

func (s *Storage) ModifyEvent(ctx context.Context, event *storage.Event) error {
	s.log.Debug().Msgf("Start editing event with id %s", event.ID)
	query := `
	UPDATE events
	SET title = :title
	WHERE id = :id;`
	ctx, cancel := context.WithTimeout(ctx, s.connectionTimeout)
	defer cancel()
	_, err := s.db.NamedExecContext(ctx, query, event)
	if err != nil {
		return fmt.Errorf("%s: %w", err.Error(), errs.ErrUpdateEvent)
	}
	s.log.Debug().Msgf("Successfully update event with id %s", event.ID)
	return nil
}

func (s *Storage) DeleteEvent(ctx context.Context, id string) error {
	s.log.Debug().Msgf("Start deleting event with id %s", id)
	query := `DELETE FROM events WHERE id=:id;`
	ctx, cancel := context.WithTimeout(ctx, s.connectionTimeout)
	defer cancel()
	_, err := s.db.NamedExecContext(ctx, query, map[string]interface{}{"id": id})
	if err != nil {
		return fmt.Errorf("%s: %w", err.Error(), errs.ErrDeleteEvent)
	}
	s.log.Debug().Msgf("Successfully deleted event with id %s", id)
	return nil
}

func (s *Storage) GetEvent(ctx context.Context, id string) (*storage.Event, error) {
	s.log.Debug().Msgf("Start getting event with id %s", id)
	query := `
	SELECT id, title 
	FROM events
	WHERE id=$1;
	`
	ctx, cancel := context.WithTimeout(ctx, s.connectionTimeout)
	defer cancel()
	row := s.db.QueryRowxContext(ctx, query, id)
	var event storage.Event
	if err := row.StructScan(&event); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("%s: %w", err.Error(), errs.ErrNotFoundEvent)
		}
		return nil, fmt.Errorf("%s: %w", err.Error(), errs.ErrGetEvent)
	}
	s.log.Debug().Msgf("Successfully got event with id %s", id)
	return &event, nil
}

func (s *Storage) ListEvents(ctx context.Context) ([]*storage.Event, error) {
	s.log.Debug().Msg("Start listing events")
	query := `
	SELECT id, title 
	FROM events;
	`
	ctx, cancel := context.WithTimeout(ctx, s.connectionTimeout)
	defer cancel()
	rows, err := s.db.QueryxContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", err.Error(), errs.ErrListEvents)
	}
	events, scanErr := s.fromSQLRowsToEvents(rows)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", scanErr.Error(), errs.ErrListEvents)
	}
	s.log.Debug().Msgf("Successfully list events")
	return events, nil
}

func (s *Storage) ListEventsToNotify(ctx context.Context,
	fromTime time.Time, period time.Duration,
) ([]*storage.Event, error) {
	s.log.Debug().Msg("ListEventsToNotify - start method")
	query := strings.Builder{}
	query.WriteString(`
	SELECT id, title, start_date, user_id
	FROM events `)
	sqlFromTimeStr := s.timeToSQLTimeWithTimezone(fromTime)
	sqlToTimeStr := s.timeToSQLTimeWithTimezone(fromTime.Add(period))
	query.WriteString(
		fmt.Sprintf("WHERE notify_date >= '%s' AND notify_date <= '%s';",
			sqlFromTimeStr, sqlToTimeStr))
	s.log.Debug().Msgf("ListEventsToNotify - try to execute query: '%s'", query.String())

	ctx, cancel := context.WithTimeout(ctx, s.connectionTimeout)
	defer cancel()
	rows, err := s.db.QueryxContext(ctx, query.String())
	if err != nil {
		return nil, fmt.Errorf("%s: %w", err.Error(), errs.ErrListEventsToNotify)
	}
	events, err := s.fromSQLRowsToEvents(rows)
	if err != nil {
		return nil, fmt.Errorf("ListEventsToNotify - failed to scan events from rows: %w", err)
	}
	s.log.Debug().Msgf("ListEventsToNotify: got '%d' events to notify", len(events))
	return events, nil
}

func (s *Storage) DeleteOldEventsBeforeTime(
	ctx context.Context,
	fromTime time.Time,
	maxLiveTime time.Duration,
) ([]*storage.Event, error) {
	s.log.Debug().Msg("DeleteOldEventsBeforeTime: start deleting old events")
	query := strings.Builder{}
	query.WriteString("DELETE FROM events ")
	query.WriteString(fmt.Sprintf("WHERE '%s' - end_date > '%s' RETURNING *;",
		s.timeToSQLTimeWithTimezone(fromTime),
		s.durationToSQLInterval(maxLiveTime)))
	s.log.Debug().Msgf("DeleteOldEventsBeforeTime: deleting with query: %s", query.String())
	rows, err := s.db.QueryxContext(ctx, query.String())
	if err != nil {
		return nil, fmt.Errorf("failed to delete old events with query: %s: %w", query.String(), err)
	}
	events, err := s.fromSQLRowsToEvents(rows)
	if err != nil {
		return nil, fmt.Errorf("ListEventsToNotify - failed to scan events from rows: %w", err)
	}
	s.log.Debug().Msgf("DeleteOldEventsBeforeTime: delete '%d' old events", len(events))
	return events, nil
}
