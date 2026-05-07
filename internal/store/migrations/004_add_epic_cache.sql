-- +goose Up
ALTER TABLE ticket_cache ADD COLUMN is_epic INTEGER NOT NULL DEFAULT 0;

CREATE TABLE epic_children (
    epic_key        TEXT NOT NULL,
    id              TEXT NOT NULL,
    summary         TEXT NOT NULL,
    reporter        TEXT NOT NULL DEFAULT '',
    status          TEXT NOT NULL DEFAULT '',
    status_category TEXT NOT NULL DEFAULT '',
    priority        TEXT NOT NULL DEFAULT '',
    is_epic         INTEGER NOT NULL DEFAULT 0,
    created_at      TEXT NOT NULL,
    updated_at      TEXT NOT NULL,
    markdown        TEXT NOT NULL DEFAULT '',
    PRIMARY KEY (epic_key, id)
);

-- +goose Down
DROP TABLE epic_children;
ALTER TABLE ticket_cache DROP COLUMN is_epic;
