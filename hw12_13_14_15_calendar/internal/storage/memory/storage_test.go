package memorystorage

import (
	"context"
	"testing"

	"github.com/hihoak/otus-course-hws/hw12_13_14_15_calendar/internal/storage"

	"github.com/stretchr/testify/require"

	"github.com/hihoak/otus-course-hws/hw12_13_14_15_calendar/internal/logger"
)

const testTitle = "hello"

func getAllEvents(data map[string]*storage.Event) []*storage.Event {
	res := make([]*storage.Event, len(data))
	idx := 0
	for _, event := range data {
		res[idx] = event
		idx++
	}
	return res
}

func fillWithData(st *Storage, data []*storage.Event) {
	for _, d := range data {
		st.data[d.ID] = d
	}
}

func compareEvent(expected *storage.Event, actual *storage.Event) bool {
	return expected.ID == actual.ID && expected.Title == actual.Title
}

func compareSlices(expected []*storage.Event, actual []*storage.Event) bool {
	if len(expected) != len(actual) {
		return false
	}
	tempActual := make([]*storage.Event, len(actual))
	copy(tempActual, actual)
	for _, exp := range expected {
		find := false
		for idx, act := range tempActual {
			if compareEvent(exp, act) {
				tempActual = append(tempActual[:idx], tempActual[idx+1:]...)
				find = true
				break
			}
		}
		if !find {
			return false
		}
	}
	return true
}

func TestStorage(t *testing.T) {
	t.Run("test storage - ADD", func(t *testing.T) {
		st := New(logger.New("debug"))
		err := st.AddEvent(context.Background(), testTitle)
		require.NoError(t, err)
		require.Equal(t, 1, len(st.data))
		events := getAllEvents(st.data)
		require.Equal(t, testTitle, events[0].Title)
	})

	t.Run("test storage - LIST", func(t *testing.T) {
		st := New(logger.New("debug"))
		testEvents := []*storage.Event{
			{
				ID:    "hello",
				Title: "title",
			},
			{
				ID:    "hello2",
				Title: "title2",
			},
			{
				ID:    "hello3",
				Title: "title3",
			},
		}
		fillWithData(st, testEvents)
		require.Equal(t, len(testEvents), len(st.data))
		events, err := st.ListEvents(context.Background())
		require.NoError(t, err)
		require.True(t, compareSlices(testEvents, events))
	})

	t.Run("")
}
