-- +goose Up

-- Recreate issue_tags with a CHECK constraint on non-empty values.
-- FK to tickets is intentionally omitted: issue_tags persists user data
-- while tickets is a volatile cache that gets cleared on sync.
CREATE TABLE issue_tags_new (
  id  TEXT NOT NULL CHECK(id != ''),
  tag TEXT NOT NULL CHECK(tag != ''),
  PRIMARY KEY (id, tag)
);
INSERT INTO issue_tags_new SELECT id, tag FROM issue_tags;
DROP TABLE issue_tags;
ALTER TABLE issue_tags_new RENAME TO issue_tags;

-- Recreate issue_todos with CHECK constraints.
-- FK to tickets is intentionally omitted for the same reason as above.
CREATE TABLE issue_todos_new (
  id        INTEGER PRIMARY KEY AUTOINCREMENT,
  ticket_id TEXT NOT NULL CHECK(ticket_id != ''),
  title     TEXT NOT NULL CHECK(title != ''),
  done      INTEGER NOT NULL DEFAULT 0 CHECK(done IN (0, 1))
);
INSERT INTO issue_todos_new SELECT id, ticket_id, title, done FROM issue_todos;
DROP TABLE issue_todos;
ALTER TABLE issue_todos_new RENAME TO issue_todos;
CREATE INDEX IF NOT EXISTS idx_issue_todos_ticket_id ON issue_todos(ticket_id);

-- +goose Down
CREATE TABLE issue_tags_old (
  id  TEXT NOT NULL,
  tag TEXT NOT NULL,
  PRIMARY KEY (id, tag)
);
INSERT INTO issue_tags_old SELECT id, tag FROM issue_tags;
DROP TABLE issue_tags;
ALTER TABLE issue_tags_old RENAME TO issue_tags;

CREATE TABLE issue_todos_old (
  id        INTEGER PRIMARY KEY AUTOINCREMENT,
  ticket_id TEXT NOT NULL,
  title     TEXT NOT NULL,
  done      INTEGER NOT NULL DEFAULT 0
);
INSERT INTO issue_todos_old SELECT id, ticket_id, title, done FROM issue_todos;
DROP TABLE issue_todos;
ALTER TABLE issue_todos_old RENAME TO issue_todos;
