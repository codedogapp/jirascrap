package store

import (
	"database/sql"

	"github.com/codedogapp/jirascrap/internal/model"
)

type LocalMeta struct {
	Tags []string
}

type MetaStore interface {
	SaveMeta(id string, tags []string) error
	GetAllMeta() (map[string]LocalMeta, error)
	GetUniqueTags() ([]string, error)
	GetTodos(ticketID string) ([]model.Todo, error)
	SaveTodos(ticketID string, todos []model.Todo) error
}

type SqliteMetaStore struct {
	db *sql.DB
}

func NewSqliteMetaStore(db *sql.DB) *SqliteMetaStore {
	return &SqliteMetaStore{
		db: db,
	}
}

func (s *SqliteMetaStore) SaveMeta(id string, tags []string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

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
			meta = LocalMeta{Tags: []string{}}
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

func (s *SqliteMetaStore) GetTodos(ticketID string) ([]model.Todo, error) {
	rows, err := s.db.Query(
		`SELECT title, done FROM issue_todos WHERE ticket_id = ? ORDER BY id ASC`,
		ticketID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var todos []model.Todo
	for rows.Next() {
		var title string
		var done int
		if err := rows.Scan(&title, &done); err != nil {
			return nil, err
		}
		todos = append(todos, model.Todo{Title: title, Done: done != 0})
	}
	return todos, nil
}

func (s *SqliteMetaStore) SaveTodos(ticketID string, todos []model.Todo) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec(`DELETE FROM issue_todos WHERE ticket_id = ?`, ticketID); err != nil {
		return err
	}

	if len(todos) > 0 {
		stmt, err := tx.Prepare(`INSERT INTO issue_todos (ticket_id, title, done) VALUES (?, ?, ?)`)
		if err != nil {
			return err
		}
		defer stmt.Close()

		for _, t := range todos {
			done := 0
			if t.Done {
				done = 1
			}
			if _, err := stmt.Exec(ticketID, t.Title, done); err != nil {
				return err
			}
		}
	}

	return tx.Commit()
}
