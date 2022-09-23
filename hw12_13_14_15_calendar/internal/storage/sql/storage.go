package sqlstorage

import (
	"context"
	"fmt"
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

	log   app.Logger
	getID xid.ID

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

		getID: xid.New(),
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
		return errs.ErrConnectionFailed{Err: err}
	}
	err = db.PingContext(ctx)
	if err != nil {
		return errs.ErrPingFailed{Err: err}
	}
	s.db = db
	s.log.Info().Msg("Successfully connected to database")
	return nil
}

func (s *Storage) Close(ctx context.Context) error {
	s.log.Info().Msg("Start closing connection to database...")
	if err := s.db.Close(); err != nil {
		return errs.ErrCloseConnectionFailed{Err: err}
	}
	s.log.Info().Msg("Successfully close connection to database")
	return s.db.Close()
}

func (s *Storage) AddEvent(ctx context.Context, event *storage.Event) error {
	query := `
		INSERT INTO events (id, title)
        VALUES (:id, :title)`
	event.ID = s.getID.String()
	s.log.Debug().Msgf("Start adding event with id %s", event.ID)
	ctx, cancel := context.WithTimeout(ctx, s.connectionTimeout)
	defer cancel()
	_, err := s.db.NamedExecContext(ctx, query, event)
	if err != nil {
		return errs.ErrAddEvent{Err: err}
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
		return errs.ErrUpdateEvent{Err: err}
	}
	s.log.Debug().Msgf("Successfully update event with id %s", event.ID)
	return nil
}

func (s *Storage) DeleteEvent(ctx context.Context, id string) error {
	s.log.Debug().Msgf("Start deleting event with id %s", id)
	query := `DELETE FROM events WHERE id = :id;`
	ctx, cancel := context.WithTimeout(ctx, s.connectionTimeout)
	defer cancel()
	_, err := s.db.ExecContext(ctx, query, id)
	if err != nil {
		return errs.ErrDeleteEvent{Err: err}
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
		return nil, errs.ErrGetEvent{Err: err}
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
	defer func() {
		if closeErr := rows.Close(); closeErr != nil {
			s.log.Error().Err(closeErr).Msg("Failed to close rows")
		}
	}()
	if err != nil {
		return nil, errs.ErrListEvents{Err: err}
	}
	events := make([]*storage.Event, 0)
	for rows.Next() {
		var event storage.Event
		if scanErr := rows.StructScan(&event); scanErr != nil {
			return nil, errs.ErrListEvents{Err: scanErr}
		}
		events = append(events, &event)
	}
	s.log.Debug().Msgf("Successfully list events")
	return events, nil
}
