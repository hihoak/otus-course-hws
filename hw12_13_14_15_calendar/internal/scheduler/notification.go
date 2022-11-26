package scheduler

import (
	"time"

	"github.com/hihoak/otus-course-hws/hw12_13_14_15_calendar/internal/storage"
)

type Notification struct {
	EventID        string     `json:"eventId" db:"event_id"`
	EventTitle     string     `db:"event_title"`
	EventStartDate *time.Time `db:"event_start_date"`
	SendToUserID   string     `db:"send_to_user_id"`
}

func FromEventToNotification(event *storage.Event) *Notification {
	return &Notification{
		EventID:        event.ID,
		EventTitle:     event.Title,
		EventStartDate: event.StartDate,
		SendToUserID:   event.UserID,
	}
}
