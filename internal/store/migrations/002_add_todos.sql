-- +goose Up
CREATE TABLE issue_todos (
  id        INTEGER PRIMARY KEY AUTOINCREMENT,
  ticket_id TEXT NOT NULL,
  title     TEXT NOT NULL,
  done      INTEGER NOT NULL DEFAULT 0
);

-- +goose Down
DROP TABLE issue_todos;
