-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS events
(
    id    varchar(128) primary key NOT NULL,
    title varchar(128)             NOT NULL
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS events;
-- +goose StatementEnd
