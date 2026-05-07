-- +goose Up
CREATE TABLE epics (
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

ALTER TABLE ticket_cache ADD COLUMN epic_id TEXT DEFAULT NULL;

-- +goose Down
ALTER TABLE ticket_cache DROP COLUMN epic_id;
DROP TABLE epics;
