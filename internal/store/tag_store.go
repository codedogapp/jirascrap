package store

import (
	"database/sql"
	"fmt"
)

// TagStore manages ticket tags/labels.
type TagStore interface {
	SaveTags(id string, tags []string) error
	GetUniqueTags() ([]string, error)
}

// SqliteTagStore implements TagStore using SQLite.
type SqliteTagStore struct {
	db *sql.DB
}

func NewSqliteTagStore(db *sql.DB) *SqliteTagStore {
	return &SqliteTagStore{db: db}
}

func (s *SqliteTagStore) SaveTags(id string, tags []string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("save tags for %s: begin tx: %w", id, err)
	}
	defer tx.Rollback()

	if _, err = tx.Exec(`DELETE FROM issue_tags WHERE id = ?`, id); err != nil {
		return fmt.Errorf("save tags for %s: delete old: %w", id, err)
	}

	if len(tags) > 0 {
		stmt, err := tx.Prepare(`INSERT INTO issue_tags (id, tag) VALUES (?, ?)`)
		if err != nil {
			return fmt.Errorf("save tags for %s: prepare: %w", id, err)
		}
		defer stmt.Close()

		for _, tag := range tags {
			if _, err = stmt.Exec(id, tag); err != nil {
				return fmt.Errorf("save tag %q for %s: %w", tag, id, err)
			}
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("save tags for %s: commit: %w", id, err)
	}
	return nil
}

func (s *SqliteTagStore) GetUniqueTags() ([]string, error) {
	rows, err := s.db.Query(`SELECT DISTINCT tag FROM issue_tags ORDER BY tag ASC`)
	if err != nil {
		return nil, fmt.Errorf("get unique tags: %w", err)
	}
	defer rows.Close()

	var tags []string
	for rows.Next() {
		var tag string
		if err := rows.Scan(&tag); err != nil {
			return nil, fmt.Errorf("get unique tags: scan: %w", err)
		}
		tags = append(tags, tag)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("get unique tags: rows: %w", err)
	}

	return tags, nil
}
