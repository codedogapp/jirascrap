-- Schema representing the final state of the database after all migrations.
-- Used by sqlc for codegen only. Goose migrations remain the runtime source of truth.

CREATE TABLE issue_tags (
  id  TEXT NOT NULL CHECK(id != ''),
  tag TEXT NOT NULL CHECK(tag != ''),
  PRIMARY KEY (id, tag)
);

CREATE TABLE issue_todos (
  id        INTEGER PRIMARY KEY AUTOINCREMENT,
  ticket_id TEXT NOT NULL CHECK(ticket_id != ''),
  title     TEXT NOT NULL CHECK(title != ''),
  done      INTEGER NOT NULL DEFAULT 0 CHECK(done IN (0, 1))
);
CREATE INDEX idx_issue_todos_ticket_id ON issue_todos(ticket_id);

CREATE TABLE tickets (
  id              TEXT PRIMARY KEY,
  summary         TEXT NOT NULL,
  reporter        TEXT NOT NULL DEFAULT '',
  status          TEXT NOT NULL DEFAULT '',
  status_category TEXT NOT NULL DEFAULT '',
  priority        TEXT NOT NULL DEFAULT '',
  type            TEXT NOT NULL DEFAULT 'task',
  created_at      TEXT NOT NULL,
  updated_at      TEXT NOT NULL,
  markdown        TEXT NOT NULL DEFAULT '',
  epic_id         TEXT DEFAULT NULL
);
CREATE INDEX idx_tickets_epic_id ON tickets(epic_id);
