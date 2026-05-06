-- +goose Up
ALTER TABLE ticket_cache ADD COLUMN is_epic INTEGER NOT NULL DEFAULT 0;
ALTER TABLE ticket_cache ADD COLUMN epic_key TEXT DEFAULT NULL;

-- +goose Down
ALTER TABLE ticket_cache DROP COLUMN epic_key;
ALTER TABLE ticket_cache DROP COLUMN is_epic;
