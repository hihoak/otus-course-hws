package storage

import "time"

type Event struct {
	ID          string     `db:"id"`
	Title       string     `db:"title"`
	StartDate   *time.Time `db:"start_date"`
	EndDate     *time.Time `db:"end_date"`
	Description string     `db:"description"`
	UserID      string     `db:"user_id"`
	NotifyDate  *time.Time `db:"notify_date"`

	ScheduledToNotify bool `db:"scheduled_to_notify"`
	IsSent            bool `db:"is_sent"`
}
