-- +goose Up
CREATE TABLE ticket_cache (
    id              TEXT PRIMARY KEY,
    summary         TEXT NOT NULL,
    reporter        TEXT NOT NULL DEFAULT '',
    status          TEXT NOT NULL DEFAULT '',
    status_category TEXT NOT NULL DEFAULT '',
    priority        TEXT NOT NULL DEFAULT '',
    created_at      TEXT NOT NULL,
    updated_at      TEXT NOT NULL,
    markdown        TEXT NOT NULL DEFAULT ''
);

-- +goose Down
DROP TABLE ticket_cache;
