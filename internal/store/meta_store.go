package store

import (
	"database/sql"
	"time"

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
	// CacheTickets - Full cache replacement, removes tickets no longer returned by Jira
	CacheTickets(tickets []model.Ticket) error
	GetCachedTickets() ([]model.Ticket, error)
	// CacheEpicChildren - Replaces cached children for a single epic
	CacheEpicChildren(epicKey string, tickets []model.Ticket) error
	GetAllCachedEpicChildren() (map[string][]model.Ticket, error)
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

	if err := tagRows.Err(); err != nil {
		return nil, err
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

	if err := rows.Err(); err != nil {
		return nil, err
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

	if err := rows.Err(); err != nil {
		return nil, err
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

func (s *SqliteMetaStore) CacheTickets(tickets []model.Ticket) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec(`DELETE FROM ticket_cache`); err != nil {
		return err
	}

	if len(tickets) == 0 {
		return tx.Commit()
	}

	stmt, err := tx.Prepare(`INSERT INTO ticket_cache (id, summary, reporter, status, status_category, priority, is_epic, created_at, updated_at, markdown) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, t := range tickets {
		if _, err := stmt.Exec(ticketInsertValues(t)...); err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (s *SqliteMetaStore) GetCachedTickets() ([]model.Ticket, error) {
	rows, err := s.db.Query(`SELECT id, summary, reporter, status, status_category, priority, is_epic, created_at, updated_at, markdown FROM ticket_cache`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tickets []model.Ticket
	for rows.Next() {
		t, err := scanTicket(rows)
		if err != nil {
			return nil, err
		}
		tickets = append(tickets, t)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return tickets, nil
}

func (s *SqliteMetaStore) CacheEpicChildren(epicKey string, tickets []model.Ticket) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec(`DELETE FROM epic_children WHERE epic_key = ?`, epicKey); err != nil {
		return err
	}

	if len(tickets) > 0 {
		stmt, err := tx.Prepare(`INSERT INTO epic_children (epic_key, id, summary, reporter, status, status_category, priority, is_epic, created_at, updated_at, markdown) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`)
		if err != nil {
			return err
		}
		defer stmt.Close()

		for _, t := range tickets {
			args := append([]any{epicKey}, ticketInsertValues(t)...)
			if _, err := stmt.Exec(args...); err != nil {
				return err
			}
		}
	}

	return tx.Commit()
}

func (s *SqliteMetaStore) GetAllCachedEpicChildren() (map[string][]model.Ticket, error) {
	rows, err := s.db.Query(`SELECT epic_key, id, summary, reporter, status, status_category, priority, is_epic, created_at, updated_at, markdown FROM epic_children ORDER BY epic_key, rowid`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[string][]model.Ticket)
	for rows.Next() {
		var epicKey string
		var t model.Ticket
		var createdAt, updatedAt string
		var isEpic int
		if err := rows.Scan(&epicKey, &t.ID, &t.Summary, &t.Reporter, &t.Status, &t.StatusCategory, &t.Priority, &isEpic, &createdAt, &updatedAt, &t.Markdown); err != nil {
			return nil, err
		}
		t.IsEpic = isEpic != 0
		t.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
		t.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
		result[epicKey] = append(result[epicKey], t)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return result, nil
}

// Helpers for ticket row scanning/insertion

type rowScanner interface {
	Scan(dest ...any) error
}

func scanTicket(row rowScanner) (model.Ticket, error) {
	var t model.Ticket
	var createdAt, updatedAt string
	var isEpic int
	if err := row.Scan(&t.ID, &t.Summary, &t.Reporter, &t.Status, &t.StatusCategory, &t.Priority, &isEpic, &createdAt, &updatedAt, &t.Markdown); err != nil {
		return model.Ticket{}, err
	}
	t.IsEpic = isEpic != 0
	t.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	t.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
	return t, nil
}

func ticketInsertValues(t model.Ticket) []any {
	isEpic := 0
	if t.IsEpic {
		isEpic = 1
	}
	return []any{t.ID, t.Summary, t.Reporter, t.Status, t.StatusCategory, t.Priority, isEpic, t.CreatedAt.Format(time.RFC3339), t.UpdatedAt.Format(time.RFC3339), t.Markdown}
}
