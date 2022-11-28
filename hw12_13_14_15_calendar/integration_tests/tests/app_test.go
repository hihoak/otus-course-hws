//go:build integration_tests
// +build integration_tests

package tests

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"sort"
	"testing"
	"time"

	jsoniter "github.com/json-iterator/go"

	"github.com/hihoak/otus-course-hws/hw12_13_14_15_calendar/internal/app"

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

const (
	fixturesDir = "fixtures"

	listFixture = "lists.sql"
	sendFixture = "send.sql"
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

func ConvertTimeToSQLTimeWithTimezone(t time.Time) string {
	return t.Format("2006-01-02 15:04:05.999-07")
}

func NewClient(t *testing.T) {
	cl := http.Client{
		Timeout: time.Second * 2,
	}
	req := desc.AddEventRequest{
		Title:  "hello, Otus!",
		UserId: "otus otusov otusovich",
	}
	reqBody, marshallErr := jsoniter.Marshal(&req)
	require.NoError(t, marshallErr)

	resp, err := cl.Post(apiURL+"/event/create", "application/json", bytes.NewBuffer(reqBody))
	require.NoError(t, err)
	require.Equal(t, 200, resp.StatusCode)
}

func applyFixture(t *testing.T, db *sqlx.DB, name string) {
	query, err := os.ReadFile(fmt.Sprintf("%s/%s", fixturesDir, name))
	require.NoError(t, err)
	db.MustExec(string(query))
}

func Test(t *testing.T) {
	ctx := context.Background()
	db := ConnectToDB(ctx, t)
	cl := http.Client{
		Timeout: time.Second * 2,
	}
	flushAll(ctx, t, db)
	defer DisconnectFromDB(ctx, t, db)
	t.Run("create event test", func(t *testing.T) {
		defer flushAll(ctx, t, db)
		req := desc.AddEventRequest{
			Title:  "hello, Otus!",
			UserId: "otus otusov otusovich",
		}
		reqBody, marshallErr := jsoniter.Marshal(&req)
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
		defer rows.Close()
		events := fromSQLRowsToEvents(t, rows)
		if len(events) != 1 {
			require.Failf(t, "wrong number of events", "wrong number of events expect 1 but got: %v", events)
		}

		require.Equal(t, expectedEvent, events[0])
	})

	t.Run("test lists events for one day/week/month", func(t *testing.T) {
		defer flushAll(ctx, t, db)
		applyFixture(t, db, listFixture)

		testCases := []struct {
			Name        string
			ForDays     int64
			FromTime    time.Time
			ExpectedLen int
		}{
			{
				Name:        "for 1 day",
				ForDays:     1,
				FromTime:    time.Date(2000, 3, 6, 0, 0, 0, 0, time.UTC),
				ExpectedLen: 2,
			},
			{
				Name:        "for 7 days",
				ForDays:     7,
				FromTime:    time.Date(2000, 3, 7, 0, 0, 0, 0, time.UTC),
				ExpectedLen: 1,
			},
			{
				Name:        "for 30 days",
				ForDays:     30,
				FromTime:    time.Date(2000, 3, 6, 0, 0, 0, 0, time.UTC),
				ExpectedLen: 4,
			},
		}

		for _, tc := range testCases {
			pbTime := app.ConvertFromTimeToPbDateTime(&tc.FromTime)

			req := desc.ListEventForDaysRequest{
				Date:    pbTime,
				ForDays: tc.ForDays,
			}
			reqBody, marshallErr := jsoniter.Marshal(&req)
			require.NoError(t, marshallErr)
			resp, err := cl.Post(apiURL+"/event/list/for-days", "application/json", bytes.NewBuffer(reqBody))
			require.NoError(t, err)
			resultEvents := desc.ListEventResponse{}
			data, err := ioutil.ReadAll(resp.Body)
			require.NoError(t, err)
			require.NoError(t, resp.Body.Close())
			require.NoError(t, jsoniter.Unmarshal(data, &resultEvents))
			resultEventsPb := resultEvents.GetEvents()
			// getting actual info
			query := `
			SELECT * 
			FROM events
			WHERE start_date >= $1 
			  and start_date < $2`
			rows, err := db.Queryx(query, ConvertTimeToSQLTimeWithTimezone(tc.FromTime),
				ConvertTimeToSQLTimeWithTimezone(tc.FromTime.Add(time.Hour*24*time.Duration(req.ForDays))))
			require.NoError(t, err)
			expectedEvents := fromSQLRowsToEvents(t, rows)
			require.NoError(t, rows.Close())

			expectedEventsPb := app.ConvertEventsToPb(expectedEvents)
			require.Equal(t, tc.ExpectedLen, len(resultEventsPb))
			require.Equal(t, len(expectedEventsPb), len(resultEventsPb))
			sort.Slice(expectedEventsPb, func(i, j int) bool {
				return expectedEventsPb[i].Id < expectedEventsPb[j].Id
			})
			sort.Slice(resultEventsPb, func(i, j int) bool {
				return resultEventsPb[i].Id < resultEventsPb[j].Id
			})
			for idx := range expectedEvents {
				require.Equal(t, expectedEventsPb[idx].Id, resultEventsPb[idx].Id)
			}
		}
	})

	t.Run("test send notifications", func(t *testing.T) {
		defer flushAll(ctx, t, db)
		applyFixture(t, db, sendFixture)

		q := `
		SELECT *
		FROM events
		WHERE id = 'send-test-123'`
		require.Eventually(t, func() bool {
			rows, err := db.Queryx(q)
			require.NoError(t, err)
			events := fromSQLRowsToEvents(t, rows)
			defer rows.Close()
			require.Equal(t, 1, len(events))
			return events[0].ScheduledToNotify && events[0].IsSent
		}, time.Second*10, time.Millisecond*250)
	})
}
