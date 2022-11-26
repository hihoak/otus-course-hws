-- +goose Up
-- +goose StatementBegin
ALTER TABLE IF EXISTS events
    ADD COLUMN IF NOT EXISTS is_sent          boolean NOT NULL default true,
    ADD COLUMN IF NOT EXISTS scheduled_to_notify boolean NOT NULL default true;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE IF EXISTS events
    DROP COLUMN IF EXISTS is_sent,
    DROP COLUMN IF EXISTS scheduled_to_notify;
-- +goose StatementEnd
