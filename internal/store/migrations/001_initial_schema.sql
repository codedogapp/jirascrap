-- +goose Up
CREATE TABLE tickets (
  id TEXT PRIMARY KEY,
  markdown TEXT,
  tags TEXT DEFAULT ''
);

-- +goose Down
DROP TABLE tickets;
