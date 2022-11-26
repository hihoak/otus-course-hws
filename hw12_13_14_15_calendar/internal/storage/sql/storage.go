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

func (s *Storage) AddEvent(ctx context.Context, title string, notifyDate, timeNow time.Time) error {
	query := `
		INSERT INTO events (id, title)
        VALUES (:id, :title)`
	dontSend := notifyDate.Before(timeNow)
	event := &storage.Event{
		Title:             title,
		ScheduledToNotify: dontSend,
		IsSent:            dontSend,
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
	fromTime time.Time, countOfEvents int,
) ([]*storage.Event, error) {
	s.log.Debug().Msg("ListEventsToNotify - start method")
	query := strings.Builder{}
	query.WriteString(`
	SELECT id, title, start_date, user_id
	FROM events `)
	sqlFromTimeStr := s.timeToSQLTimeWithTimezone(fromTime)
	query.WriteString(
		fmt.Sprintf("WHERE notify_date <= '%s' and not scheduled_to_notify and not is_sent LIMIT %d;",
			sqlFromTimeStr, countOfEvents))
	s.log.Debug().Msgf("ListEventsToNotify - try to execute query: '%s'", query.String())

	var err error
	ctx, cancel := context.WithTimeout(ctx, s.connectionTimeout)
	defer cancel()
	selectRows, err := s.db.QueryxContext(ctx, query.String())
	if err != nil {
		return nil, fmt.Errorf("%s: %w", err.Error(), errs.ErrListEventsToNotify)
	}
	defer func() {
		if closeErr := selectRows.Close(); closeErr != nil {
			s.log.Error().Err(closeErr).Msg("ListEventsToNotify: failed to close rows")
		}
	}()
	events, err := s.fromSQLRowsToEvents(selectRows)
	if err != nil {
		return nil, fmt.Errorf("ListEventsToNotify - failed to scan events from rows: %w", err)
	}

	if len(events) == 0 {
		s.log.Debug().Msg("ListEventsToNotify: not found events")
		return events, nil
	}

	modifyQuery := `
	UPDATE events 
	SET scheduled_to_notify = true
	WHERE id in (?)
`
	allIDs := make([]string, len(events))
	for idx, event := range events {
		allIDs[idx] = event.ID
	}
	modifyQuery, args, err := sqlx.In(modifyQuery, allIDs)
	if err != nil {
		return nil, fmt.Errorf("ListEventsToNotify: failed to prepare query: %w", err)
	}
	modifyQuery = s.db.Rebind(modifyQuery)

	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("ListEventsToNotify: failed to create transaction: %w", err)
	}
	defer func() {
		s.rollbackOrCommit(tx, err)
	}()
	updateRows, err := tx.QueryxContext(ctx, modifyQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("ListEventsToNotify: failed to set notified status to events: %w", err)
	}
	defer func() {
		if closeErr := updateRows.Close(); closeErr != nil {
			s.log.Error().Err(closeErr).Msg("ListEventsToNotify: failed to close rows")
		}
	}()
	s.log.Debug().Msgf("ListEventsToNotify: got '%d' events to notify", len(events))
	return events, nil
}

func (s *Storage) DeleteOldEventsBeforeTime(
	ctx context.Context,
	fromTime time.Time,
	maxLiveTime time.Duration,
) error {
	s.log.Debug().Msg("DeleteOldEventsBeforeTime: start deleting old events")
	query := strings.Builder{}
	query.WriteString("DELETE FROM events ")
	query.WriteString(fmt.Sprintf("WHERE '%s' - end_date > '%s'",
		s.timeToSQLTimeWithTimezone(fromTime),
		s.durationToSQLInterval(maxLiveTime)))
	s.log.Debug().Msgf("DeleteOldEventsBeforeTime: deleting with query: %s", query.String())
	rows, err := s.db.QueryxContext(ctx, query.String())
	if err != nil {
		return fmt.Errorf("failed to delete old events with query: %s: %w", query.String(), err)
	}
	defer func() {
		if closeErr := rows.Close(); closeErr != nil {
			s.log.Error().Err(closeErr).Msg("DeleteOldEventsBeforeTime: failed to close rows")
		}
	}()
	s.log.Debug().Msgf("DeleteOldEventsBeforeTime: delete old events")
	return nil
}

func (s *Storage) rollbackOrCommit(tx *sqlx.Tx, err error) {
	if err == nil {
		if errCommit := tx.Commit(); errCommit != nil {
			s.log.Error().Err(errCommit).Msg("ListEventsToNotify: failed to commit transaction")
		} else {
			s.log.Debug().Msg("ListEventsToNotify: successfully commit transaction")
			return
		}
	}
	if errRollback := tx.Rollback(); errRollback != nil {
		s.log.Error().Err(errRollback).Msg("ListEventsToNotify: failed to rollback transaction")
		return
	}
	s.log.Debug().Msg("ListEventsToNotify: successfully rollback transaction")
}

func (s *Storage) SetSentStatusToEvents(ctx context.Context, ids []string) error {
	s.log.Debug().Msgf("SetSentStatusToEvents: start set sent statuses to ids: %v", ids)
	if len(ids) == 0 {
		s.log.Debug().Msg("SetSentStatusToEvents: empty ids: nothing todo")
		return nil
	}
	query := `
	UPDATE events
	SET is_sent = true
	WHERE id in (?)`

	query, args, err := sqlx.In(query, ids)
	if err != nil {
		return fmt.Errorf("SetSentStatusToEvents: failed to prepare query: %w", err)
	}

	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("SetSentStatusToEvents: failed to start trransaction: %w", err)
	}
	defer func() {
		s.rollbackOrCommit(tx, err)
	}()
	rows, err := s.db.QueryxContext(ctx, s.db.Rebind(query), args...)
	if err != nil {
		return fmt.Errorf("SetSentStatusToEvents: failed to set sent status: %w", err)
	}
	defer func() {
		if closeErr := rows.Close(); closeErr != nil {
			s.log.Error().Err(closeErr).Msg("SetSentStatusToEvents: failed to close rows")
		}
	}()

	s.log.Debug().Msgf("SetSentStatusToEvents: successfully set sent status to all ids!!!")
	return nil
}
