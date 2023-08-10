-- +goose Up
-- +goose StatementBegin
CREATE INDEX main_idx ON events (id);
CREATE INDEX DeleteOldEventsBeforeTime_idx ON events (end_date);
CREATE INDEX ListEventsToNotify_idx ON events (notify_date, scheduled_to_notify, is_sent);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS main_idx;
DROP INDEX IF EXISTS DeleteOldEventsBeforeTime_idx;
DROP INDEX IF EXISTS ListEventsToNotify_idx;
-- +goose StatementEnd
