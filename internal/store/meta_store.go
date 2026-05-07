package store

import (
	"database/sql"
	"strings"
	"time"

	"github.com/codedogapp/jirascrap/internal/model"
)

type MetaStore interface {
	SaveMeta(id string, tags []string) error

	GetUniqueTags() ([]string, error)

	GetTodos(ticketID string) ([]model.Todo, error)

	SaveTodos(ticketID string, todos []model.Todo) error

	// CacheTickets - Full cache replacement. Stores all tickets in unified
	// `tickets` table with their type. Preserves epic children.
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

	_, err = tx.Exec(`DELETE FROM tickets WHERE epic_id IS NULL`)
	if err != nil {
		return err
	}

	if len(tickets) == 0 {
		return tx.Commit()
	}

	stmt, err := tx.Prepare(`
		INSERT INTO tickets (
			id, 
		 	summary,
			reporter, 
			status,
			status_category,
			priority,
			type,
			created_at,
			updated_at,
			markdown
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`)
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

// GetCachedTickets returns top-level tickets and epics, with tags pre-joined.
func (s *SqliteMetaStore) GetCachedTickets() ([]model.Ticket, error) {
	rows, err := s.db.Query(`
		SELECT t.id, t.summary, t.reporter, t.status, t.status_category,
		       t.priority, t.type, t.created_at, t.updated_at, t.markdown,
		       GROUP_CONCAT(it.tag) AS tags
		FROM tickets t
		LEFT JOIN issue_tags it ON t.id = it.id
		WHERE t.epic_id IS NULL
		   OR LOWER(t.type) = 'epic'
		GROUP BY t.id
		ORDER BY t.updated_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tickets []model.Ticket
	for rows.Next() {
		t, err := scanTicketWithTags(rows)
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

	_, err = tx.Exec(`DELETE FROM tickets WHERE epic_id = ?`, epicKey)
	if err != nil {
		return err
	}

	if len(tickets) > 0 {
		stmt, err := tx.Prepare(`
			INSERT OR REPLACE INTO tickets (
				id, 
			    summary,
			   	reporter,
			   	status,
			   	status_category,
				priority, 
				type, 
				created_at, 
			   	updated_at,
		   		markdown, 
			   	epic_id
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`)
		if err != nil {
			return err
		}
		defer stmt.Close()

		for _, t := range tickets {
			args := append(ticketInsertValues(t), epicKey)
			if _, err := stmt.Exec(args...); err != nil {
				return err
			}
		}
	}

	return tx.Commit()
}

func (s *SqliteMetaStore) GetAllCachedEpicChildren() (map[string][]model.Ticket, error) {
	rows, err := s.db.Query(`
		SELECT t.epic_id, 
		       t.id,
		       t.summary,
		       t.reporter,
		       t.status,
		       t.status_category,
		       t.priority,
		       t.type,
		       t.created_at,
		       t.updated_at,
		       t.markdown,
		       GROUP_CONCAT(it.tag) AS tags
		FROM tickets t
		LEFT JOIN issue_tags it ON t.id = it.id
		WHERE t.epic_id IS NOT NULL
		GROUP BY t.id
		ORDER BY t.epic_id, t.updated_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[string][]model.Ticket)
	for rows.Next() {
		var epicID string
		var t model.Ticket
		var createdAt, updatedAt string
		var tags sql.NullString
		if err := rows.Scan(
			&epicID,
			&t.ID,
			&t.Summary,
			&t.Reporter,
			&t.Status,
			&t.StatusCategory,
			&t.Priority,
			&t.Type,
			&createdAt,
			&updatedAt,
			&t.Markdown,
			&tags,
		); err != nil {
			return nil, err
		}
		t.EpicID = epicID
		t.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
		t.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
		if tags.Valid && tags.String != "" {
			t.Tags = strings.Split(tags.String, ",")
		}
		result[epicID] = append(result[epicID], t)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return result, nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

// scanTicketWithTags scans a ticket row with a GROUP_CONCAT(tag) column appended.
func scanTicketWithTags(row rowScanner) (model.Ticket, error) {
	var t model.Ticket
	var createdAt, updatedAt string
	var tags sql.NullString
	if err := row.Scan(
		&t.ID,
		&t.Summary,
		&t.Reporter,
		&t.Status,
		&t.StatusCategory,
		&t.Priority,
		&t.Type,
		&createdAt,
		&updatedAt,
		&t.Markdown,
		&tags,
	); err != nil {
		return model.Ticket{}, err
	}
	t.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	t.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
	if tags.Valid && tags.String != "" {
		t.Tags = strings.Split(tags.String, ",")
	}
	return t, nil
}

func ticketInsertValues(t model.Ticket) []any {
	typ := t.Type
	if typ == "" {
		typ = "task"
	}
	return []any{
		t.ID,
		t.Summary,
		t.Reporter,
		t.Status,
		t.StatusCategory,
		t.Priority,
		typ,
		t.CreatedAt.Format(time.RFC3339),
		t.UpdatedAt.Format(time.RFC3339),
		t.Markdown,
	}
}
