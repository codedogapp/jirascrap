-- +goose Up
ALTER TABLE ticket_cache RENAME TO tickets;
ALTER TABLE tickets ADD COLUMN type TEXT NOT NULL DEFAULT 'task';
ALTER TABLE tickets ADD COLUMN epic_id TEXT DEFAULT NULL;

-- +goose Down
ALTER TABLE tickets DROP COLUMN epic_id;
ALTER TABLE tickets DROP COLUMN type;
ALTER TABLE tickets RENAME TO ticket_cache;
