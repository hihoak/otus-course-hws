//go:build integration_tests
// +build integration_tests

package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/hihoak/otus-course-hws/hw12_13_14_15_calendar/internal/sender"

	"google.golang.org/grpc/credentials/insecure"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"google.golang.org/grpc"

	"github.com/hihoak/otus-course-hws/hw12_13_14_15_calendar/internal/app"

	errs "github.com/hihoak/otus-course-hws/hw12_13_14_15_calendar/internal/pkg/storage_errors"

	"github.com/hihoak/otus-course-hws/hw12_13_14_15_calendar/internal/storage"
	_ "github.com/lib/pq"

	desc "github.com/hihoak/otus-course-hws/hw12_13_14_15_calendar/pkg/api/event"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
)

var (
	apiURL         = os.Getenv("CALENDAR_API_URL")
	dbDSN          = os.Getenv("DB_DSN")
	outputFilePath = os.Getenv("SENDER_OUTPUT_FILE")
)

const (
	fixturesDir = "fixtures"

	listFixture = "lists.sql"
	sendFixture = "send.sql"

	testTitle  = "hello, Otus!"
	testUserId = "otus otusov otusovich"
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
			require.NoError(t, fmt.Errorf("%s: %w", scanErr.Error(), errs.ErrListEventsToNotify))
		}
		if event.StartDate != nil {
			*event.StartDate = event.StartDate.In(time.UTC)
		}
		if event.EndDate != nil {
			*event.EndDate = event.EndDate.In(time.UTC)
		}
		if event.NotifyDate != nil {
			*event.NotifyDate = event.NotifyDate.In(time.UTC)
		}
		events = append(events, &event)
	}
	return events
}

func NewClient(t *testing.T) desc.EventServiceClient {
	cl, err := grpc.Dial(apiURL, grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)
	return desc.NewEventServiceClient(cl)
}

func applyFixture(t *testing.T, db *sqlx.DB, name string) {
	query, err := os.ReadFile(fmt.Sprintf("%s/%s", fixturesDir, name))
	require.NoError(t, err)
	db.MustExec(string(query))
}

func checkStatusCode(t *testing.T, err error, code codes.Code) {
	require.Equal(t, status.Code(err), code)
}

func Test(t *testing.T) {
	ctx := context.Background()
	db := ConnectToDB(ctx, t)
	cl := NewClient(t)
	flushAll(ctx, t, db)
	defer DisconnectFromDB(ctx, t, db)
	testStartDate := time.Date(2000, 3, 6, 0, 0, 0, 0, time.UTC)
	t.Run("create event test, invalid argument", func(t *testing.T) {
		defer flushAll(ctx, t, db)
		req := &desc.AddEventRequest{
			Title:  testTitle,
			UserId: testUserId,
		}
		_, err := cl.CreateEvent(ctx, req)
		checkStatusCode(t, err, codes.InvalidArgument)
	})

	t.Run("create event test, success", func(t *testing.T) {
		defer flushAll(ctx, t, db)
		req := &desc.AddEventRequest{
			Title:     "hello, Otus!",
			UserId:    "otus otusov otusovich",
			StartDate: app.ConvertFromTimeToPbDateTime(&testStartDate),
		}
		createResp, err := cl.CreateEvent(ctx, req)
		require.NoError(t, err)

		expectedEvent := &storage.Event{
			ID:        createResp.GetId(),
			Title:     testTitle,
			UserID:    testUserId,
			StartDate: &testStartDate,
		}

		getResp, err := cl.GetEvent(ctx, &desc.GetEventRequest{Id: createResp.GetId()})
		require.NoError(t, err)
		require.Equal(t, expectedEvent, app.ConvertPbToEvent(getResp.GetEvent()))

		query := `
SELECT id, title, user_id, start_date 
FROM events 
WHERE id = $1;
`

		rows, err := db.QueryxContext(ctx, query, createResp.GetId())
		require.NoError(t, err)
		defer rows.Close()
		events := fromSQLRowsToEvents(t, rows)
		require.Equal(t, 1, len(events))
		require.Equal(t, expectedEvent, events[0])
	})

	t.Run("test lists events for one day/week/month and then delete them", func(t *testing.T) {
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
			flushAll(ctx, t, db)
			applyFixture(t, db, listFixture)
			pbTime := app.ConvertFromTimeToPbDateTime(&tc.FromTime)

			listReq := &desc.ListEventForDaysRequest{
				Date:    pbTime,
				ForDays: tc.ForDays,
			}
			resp, err := cl.ListEventForDays(ctx, listReq)
			require.NoError(t, err)
			require.Equal(t, tc.ExpectedLen, len(resp.GetEvents()))

			for _, event := range resp.GetEvents() {
				_, err := cl.DeleteEvent(ctx, &desc.DeleteEventRequest{Id: event.Id})
				require.NoError(t, err)
			}

			resp, err = cl.ListEventForDays(ctx, listReq)
			require.NoError(t, err)
			require.Equal(t, 0, len(resp.GetEvents()))
		}
	})

	t.Run("test lists events for one month and then modify them to exclude all", func(t *testing.T) {
		listFromTime := time.Date(2000, 3, 6, 0, 0, 0, 0, time.UTC)
		testCases := []struct {
			Name        string
			ForDays     int64
			FromTime    time.Time
			NewTimes    []time.Time
			ExpectedLen int
		}{
			{
				Name:        "for 30 days",
				ForDays:     30,
				FromTime:    listFromTime,
				ExpectedLen: 4,
				NewTimes: []time.Time{
					listFromTime.Add(time.Hour * 24 * 60),  // modify first event to +60 days
					listFromTime.Add(time.Hour * 24 * 31),  // to +31 day
					listFromTime.Add(-time.Hour),           // to  -1 hour
					listFromTime.Add(-time.Hour * 24 * 48), // to -48 days
				},
			},
		}

		for _, tc := range testCases {
			flushAll(ctx, t, db)
			applyFixture(t, db, listFixture)
			pbTime := app.ConvertFromTimeToPbDateTime(&tc.FromTime)

			listReq := &desc.ListEventForDaysRequest{
				Date:    pbTime,
				ForDays: tc.ForDays,
			}
			resp, err := cl.ListEventForDays(ctx, listReq)
			require.NoError(t, err)
			require.Equal(t, tc.ExpectedLen, len(resp.GetEvents()))

			for idx, event := range resp.GetEvents() {
				_, err := cl.ModifyEvent(ctx, &desc.ModifyEventRequest{
					Event: app.ConvertEventToPb(&storage.Event{
						ID:        event.GetId(),
						StartDate: &tc.NewTimes[idx],
					}),
				})
				require.NoError(t, err)
			}

			// expect 0 events after modify
			resp, err = cl.ListEventForDays(ctx, listReq)
			require.NoError(t, err)
			require.Equal(t, 0, len(resp.GetEvents()))
		}
	})

	t.Run("test lists events for one month and then modify them to new ones", func(t *testing.T) {
		listFromTime := time.Date(2000, 3, 6, 0, 0, 0, 0, time.UTC)
		newStartTime := listFromTime.Add(time.Hour)
		newEndTime := listFromTime.Add(time.Hour * 2)
		newNotifyTime := listFromTime.Add(time.Minute * 30)
		testCases := []struct {
			Name        string
			ForDays     int64
			FromTime    time.Time
			ModifyData  []*storage.Event
			ExpectedLen int
		}{
			{
				Name:        "for 30 days",
				ForDays:     30,
				FromTime:    listFromTime,
				ExpectedLen: 4,
				ModifyData: []*storage.Event{
					// ids you can see in fixtures file
					{
						ID:          "3",
						Title:       "new title 1",
						Description: "new description 1",
						UserID:      "new user 1",
					},
					{
						ID:         "4",
						StartDate:  &newStartTime,
						EndDate:    &newEndTime,
						NotifyDate: &newNotifyTime,
					},
					{
						ID: "5",
						// nothing new in third event
					},
					{
						ID: "6",
						// nothing new in fourth event
					},
				},
			},
		}

		for _, tc := range testCases {
			flushAll(ctx, t, db)
			applyFixture(t, db, listFixture)
			pbTime := app.ConvertFromTimeToPbDateTime(&tc.FromTime)

			listReq := &desc.ListEventForDaysRequest{
				Date:    pbTime,
				ForDays: tc.ForDays,
			}
			oldResp, err := cl.ListEventForDays(ctx, listReq)
			require.NoError(t, err)
			require.Equal(t, tc.ExpectedLen, len(oldResp.GetEvents()))

			for idx := range oldResp.GetEvents() {
				_, err := cl.ModifyEvent(ctx, &desc.ModifyEventRequest{
					Event: app.ConvertEventToPb(tc.ModifyData[idx]),
				})
				if tc.ModifyData[idx].ID == "5" || tc.ModifyData[idx].ID == "6" {
					require.Error(t, err) // nothing to update error
				} else {
					require.NoError(t, err)
				}
			}

			newResp, err := cl.ListEventForDays(ctx, listReq)
			require.NoError(t, err)
			require.Equal(t, tc.ExpectedLen, len(newResp.GetEvents()))

			oldEvents := oldResp.GetEvents()
			sort.Slice(oldEvents, func(i, j int) bool {
				return oldEvents[i].Id < oldEvents[j].Id
			})
			newEvents := newResp.GetEvents()
			sort.Slice(newEvents, func(i, j int) bool {
				return newEvents[i].Id < newEvents[j].Id
			})
			for idx, event := range newResp.GetEvents() {
				if event.Id == "5" || event.Id == "6" {
					require.Equal(t, oldEvents[idx], event)
				} else {
					require.NotEqual(t, oldEvents[idx], event)
				}
			}
		}
	})

	t.Run("test send notifications", func(t *testing.T) {
		defer flushAll(ctx, t, db)
		splitPath := strings.Split(outputFilePath, "/")
		require.NoError(t, os.MkdirAll(strings.Join(splitPath[:len(splitPath)-1], "/"), 0777))
		outFile, err := os.OpenFile(outputFilePath, os.O_CREATE|os.O_TRUNC|os.O_RDWR, 0777)
		require.NoError(t, err)
		require.NoError(t, outFile.Truncate(0))
		applyFixture(t, db, sendFixture)

		sendTestEventID := "send-test-123"
		expectedEvent, err := cl.GetEvent(ctx, &desc.GetEventRequest{Id: sendTestEventID})
		require.NoError(t, err)

		var notification *sender.Notification
		require.Eventually(t, func() bool {
			data, err := io.ReadAll(outFile)
			require.NoError(t, err)
			if len(data) == 0 {
				return false
			}
			splittedData := bytes.Split(bytes.Trim(data, "\x00"), []byte("\n"))
			for _, event := range splittedData {
				require.NoError(t, json.Unmarshal(bytes.TrimSpace(event), &notification))
				if expectedEvent.GetEvent().GetId() == notification.EventID {
					return true
				}
			}
			return false
		}, time.Second*10, time.Millisecond*250)
	})
}
