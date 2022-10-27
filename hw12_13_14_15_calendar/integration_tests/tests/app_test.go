//go:build integration_tests
// +build integration_tests

package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"os"
	"testing"
	"time"

	errs "github.com/hihoak/otus-course-hws/hw12_13_14_15_calendar/internal/pkg/storage_errors"
	"github.com/pkg/errors"

	"github.com/hihoak/otus-course-hws/hw12_13_14_15_calendar/internal/storage"
	_ "github.com/lib/pq"

	desc "github.com/hihoak/otus-course-hws/hw12_13_14_15_calendar/pkg/api/event"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
)

var (
	apiURL = os.Getenv("CALENDAR_API_URL")
	dbDSN  = os.Getenv("DB_DSN")
)

func ConnectToDB(ctx context.Context, t *testing.T) *sqlx.DB {
	sql, err := sqlx.ConnectContext(ctx, "postgres", dbDSN)
	require.NoErrorf(t, err, "failed to connect to database with library")
	return sql
}

func DisconnectFromDB(ctx context.Context, t *testing.T, sql *sqlx.DB) {
	require.NoError(t, sql.Close())
}

func flushAll(ctx context.Context, t *testing.T, sql *sqlx.DB) {
	query := `DELETE FROM events`
	_, err := sql.QueryxContext(ctx, query)
	require.NoError(t, err)
}

func fromSQLRowsToEvents(t *testing.T, rows *sqlx.Rows) []*storage.Event {
	defer func() {
		if closeErr := rows.Close(); closeErr != nil {
			require.NoErrorf(t, closeErr, "Failed to close rows")
		}
	}()
	events := make([]*storage.Event, 0)
	for rows.Next() {
		var event storage.Event
		if scanErr := rows.StructScan(&event); scanErr != nil {
			require.NoError(t, errors.Wrap(errs.ErrListEventsToNotify, scanErr.Error()))
		}
		events = append(events, &event)
	}
	return events
}

func Test(t *testing.T) {
	ctx := context.Background()
	db := ConnectToDB(ctx, t)
	flushAll(ctx, t, db)
	defer DisconnectFromDB(ctx, t, db)
	t.Run("create event test", func(t *testing.T) {
		defer flushAll(ctx, t, db)
		cl := http.Client{
			Timeout: time.Second * 2,
		}
		req := desc.AddEventRequest{
			Title:  "hello, Otus!",
			UserId: "otus otusov otusovich",
		}
		reqBody, marshallErr := json.Marshal(&req)
		require.NoError(t, marshallErr)

		resp, err := cl.Post(apiURL+"/event/create", "application/json", bytes.NewBuffer(reqBody))
		require.NoError(t, err)
		require.Equal(t, 200, resp.StatusCode)

		expectedEvent := &storage.Event{
			Title:  req.GetTitle(),
			UserID: req.GetUserId(),
		}
		query := `
SELECT title, user_id
FROM events 
WHERE title=$1 and user_id=$2;
`

		rows, err := db.QueryxContext(ctx, query,
			expectedEvent.Title, expectedEvent.UserID)
		require.NoError(t, err)
		events := fromSQLRowsToEvents(t, rows)
		if len(events) != 1 {
			require.Failf(t, "wrong number of events", "wrong number of events expect 1 but got: %v", events)
		}

		require.Equal(t, expectedEvent, events[0])
	})
}
