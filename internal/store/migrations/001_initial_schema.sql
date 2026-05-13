-- +goose Up
CREATE TABLE issue_tags (
  id  TEXT NOT NULL,
  tag TEXT NOT NULL,
  PRIMARY KEY (id, tag)
);

-- +goose Down
DROP TABLE issue_tags;
