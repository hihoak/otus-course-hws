-- +goose Up
-- +goose StatementBegin
ALTER TABLE IF EXISTS events
    ALTER COLUMN title TYPE text,
    ADD COLUMN IF NOT EXISTS start_date          timestamp with time zone,
    ADD COLUMN IF NOT EXISTS end_date            timestamp with time zone,
    ADD COLUMN IF NOT EXISTS description         text,
    ADD COLUMN IF NOT EXISTS user_id             varchar(128),
    ADD COLUMN IF NOT EXISTS notify_date         timestamp with time zone,
    ADD COLUMN IF NOT EXISTS scheduled_to_notify bool;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE IF EXISTS events
    ALTER COLUMN title TYPE varchar(128),
    DROP COLUMN IF EXISTS start_date,
    DROP COLUMN IF EXISTS end_date,
    DROP COLUMN IF EXISTS description,
    DROP COLUMN IF EXISTS user_id,
    DROP COLUMN IF EXISTS notify_date,
    DROP COLUMN IF EXISTS scheduled_to_notify;
-- +goose StatementEnd
