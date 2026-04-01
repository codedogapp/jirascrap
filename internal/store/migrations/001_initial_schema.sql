-- +goose Up
CREATE TABLE issue_tags (
  id  TEXT NOT NULL,
  tag TEXT NOT NULL,
  PRIMARY KEY (id, tag)
);

CREATE TABLE issue_notes (
  id    TEXT PRIMARY KEY,
  notes TEXT
);

-- +goose Down
DROP TABLE issue_tags;
DROP TABLE issue_notes;
