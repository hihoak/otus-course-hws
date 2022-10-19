package sqlstorage

import (
	"fmt"
	"time"

	errs "github.com/hihoak/otus-course-hws/hw12_13_14_15_calendar/internal/pkg/storage_errors"
	"github.com/hihoak/otus-course-hws/hw12_13_14_15_calendar/internal/storage"
	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
)

func (s *Storage) durationToSQLInterval(d time.Duration) string {
	return fmt.Sprintf("%f seconds", d.Seconds())
}

func (s *Storage) timeToSQLTimeWithTimezone(t time.Time) string {
	return t.Format("2006-01-02 15:04:05.999-07")
}

func (s *Storage) fromSQLRowsToEvents(rows *sqlx.Rows) ([]*storage.Event, error) {
	defer func() {
		if closeErr := rows.Close(); closeErr != nil {
			s.log.Error().Err(closeErr).Msg("Failed to close rows")
		}
	}()
	events := make([]*storage.Event, 0)
	for rows.Next() {
		var event storage.Event
		if scanErr := rows.StructScan(&event); scanErr != nil {
			return nil, errors.Wrap(errs.ErrListEventsToNotify, scanErr.Error())
		}
		events = append(events, &event)
	}
	return events, nil
}
