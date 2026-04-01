package store

import (
	"database/sql"
)

type LocalMeta struct {
	Tags  []string
	Notes string
}

type MetaStore interface {
	SaveMeta(id string, tags []string, notes string) error
	GetAllMeta() (map[string]LocalMeta, error)
	GetUniqueTags() ([]string, error)
}

type SqliteMetaStore struct {
	db *sql.DB
}

func NewSqliteMetaStore(db *sql.DB) *SqliteMetaStore {
	return &SqliteMetaStore{
		db: db,
	}
}

func (s *SqliteMetaStore) SaveMeta(id string, tags []string, notes string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	query := `
	INSERT INTO issue_notes (id, notes) VALUES (?, ?)
	ON CONFLICT(id) DO UPDATE SET notes = excluded.notes;
	`

	_, err = tx.Exec(query, id, notes)
	if err != nil {
		return err
	}

	_, err = tx.Exec(`DELETE FROM issue_tags WHERE id = ?`, id)
	if err != nil {
		return err
	}

	if len(tags) > 0 {
		stmt, err := tx.Prepare(`INSERT INTO issue_tags (id, tag) VALUES (?, ?)`)
		if err != nil {
			return err
		}
		defer stmt.Close()

		for _, tag := range tags {
			if _, err = stmt.Exec(id, tag); err != nil {
				return err
			}
		}
	}

	return tx.Commit()
}

func (s *SqliteMetaStore) GetAllMeta() (map[string]LocalMeta, error) {
	metaMap := make(map[string]LocalMeta)

	noteRows, err := s.db.Query(`SELECT id, notes FROM issue_notes`)
	if err != nil {
		return nil, err
	}
	defer noteRows.Close()

	for noteRows.Next() {
		var id, notes string
		err := noteRows.Scan(&id, &notes)
		if err != nil {
			return nil, err
		}
		metaMap[id] = LocalMeta{Notes: notes, Tags: []string{}}
	}

	tagRows, err := s.db.Query(`SELECT id, tag FROM issue_tags`)
	if err != nil {
		return nil, err
	}
	defer tagRows.Close()

	for tagRows.Next() {
		var id, tag string
		err := tagRows.Scan(&id, &tag)
		if err != nil {
			return nil, err
		}

		meta, exists := metaMap[id]
		if !exists {
			meta = LocalMeta{Notes: "", Tags: []string{}}
		}

		meta.Tags = append(meta.Tags, tag)
		metaMap[id] = meta
	}

	return metaMap, nil
}

func (s *SqliteMetaStore) GetUniqueTags() ([]string, error) {
	rows, err := s.db.Query(`SELECT DISTINCT tag FROM issue_tags ORDER BY tag ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tags []string
	for rows.Next() {
		var tag string
		err := rows.Scan(&tag)
		if err != nil {
			return nil, err
		}
		tags = append(tags, tag)
	}

	return tags, nil
}
