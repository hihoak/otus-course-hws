package sqlstorage

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/hihoak/otus-course-hws/hw12_13_14_15_calendar/internal/app"
	"github.com/hihoak/otus-course-hws/hw12_13_14_15_calendar/internal/logger"
	errs "github.com/hihoak/otus-course-hws/hw12_13_14_15_calendar/internal/pkg/storage_errors"
	"github.com/hihoak/otus-course-hws/hw12_13_14_15_calendar/internal/storage"
	"github.com/jmoiron/sqlx"
	// needs github.com/lib/pq.
	_ "github.com/lib/pq"
	"github.com/pkg/errors"
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
		return errors.Wrap(errs.ErrConnectionFailed, err.Error())
	}
	err = db.PingContext(ctx)
	if err != nil {
		return errors.Wrap(errs.ErrPingFailed, err.Error())
	}
	s.db = db
	s.log.Info().Msg("Successfully connected to database")
	return nil
}

func (s *Storage) Close(ctx context.Context) error {
	s.log.Info().Msg("Start closing connection to database...")
	if err := s.db.Close(); err != nil {
		return errors.Wrap(errs.ErrCloseConnectionFailed, err.Error())
	}
	s.log.Info().Msg("Successfully close connection to database")
	return s.db.Close()
}

func (s *Storage) AddEvent(ctx context.Context, event *storage.Event) error {
	query := `
		INSERT INTO events (id, title, start_date, end_date, description, user_id, notify_date)
        VALUES (:id, :title, :start_date, :end_date, :description, :user_id, :notify_date)`
	event.ID = xid.New().String()
	s.log.Debug().Msgf("Start adding event with id %s", event.ID)
	ctx, cancel := context.WithTimeout(ctx, s.connectionTimeout)
	defer cancel()
	_, err := s.db.NamedExecContext(ctx, query, event)
	if err != nil {
		return errors.Wrap(errs.ErrAddEvent, err.Error())
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
		return errors.Wrap(errs.ErrUpdateEvent, err.Error())
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
		return errors.Wrap(errs.ErrDeleteEvent, err.Error())
	}
	s.log.Debug().Msgf("Successfully deleted event with id %s", id)
	return nil
}

func (s *Storage) GetEvent(ctx context.Context, id string) (*storage.Event, error) {
	s.log.Debug().Msgf("Start getting event with id %s", id)
	query := `
	SELECT id, title, start_date, end_date, description, user_id, notify_date 
	FROM events
	WHERE id=$1;
	`
	ctx, cancel := context.WithTimeout(ctx, s.connectionTimeout)
	defer cancel()
	row := s.db.QueryRowxContext(ctx, query, id)
	var event storage.Event
	if err := row.StructScan(&event); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.Wrap(errs.ErrNotFoundEvent, err.Error())
		}
		return nil, errors.Wrap(errs.ErrGetEvent, err.Error())
	}
	s.log.Debug().Msgf("Successfully got event with id %s", id)
	return &event, nil
}

func (s *Storage) ListEvents(ctx context.Context) ([]*storage.Event, error) {
	s.log.Debug().Msg("Start listing events")
	query := `
	SELECT id, title, start_date, end_date, description, user_id, notify_date
	FROM events;
	`
	ctx, cancel := context.WithTimeout(ctx, s.connectionTimeout)
	defer cancel()
	rows, err := s.db.QueryxContext(ctx, query)
	if err != nil {
		return nil, errors.Wrap(errs.ErrListEvents, err.Error())
	}
	events, err := s.fromSQLRowsToEvents(rows)
	if err != nil {
		return nil, errors.Wrap(err, "ListEvents - failed to scan events from rows")
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
		return nil, errors.Wrap(errs.ErrListEventsToNotify, err.Error())
	}
	events, err := s.fromSQLRowsToEvents(rows)
	if err != nil {
		return nil, errors.Wrap(err, "ListEventsToNotify - failed to scan events from rows")
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
		return nil, errors.Wrap(err, fmt.Sprintf("failed to delete old events with query: %s", query.String()))
	}
	events, err := s.fromSQLRowsToEvents(rows)
	if err != nil {
		return nil, errors.Wrap(err, "ListEventsToNotify - failed to scan events from rows")
	}
	s.log.Debug().Msgf("DeleteOldEventsBeforeTime: delete '%d' old events", len(events))
	return events, nil
}
