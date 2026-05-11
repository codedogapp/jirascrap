package store

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/codedogapp/jirascrap/internal/logger"
	"github.com/codedogapp/jirascrap/internal/model"
)

// TagStore manages ticket tags/labels.
type TagStore interface {
	SaveMeta(id string, tags []string) error
	GetUniqueTags() ([]string, error)
}

// TodoStore manages per-ticket todo items.
type TodoStore interface {
	GetTodos(ticketID string) ([]model.Todo, error)
	SaveTodos(ticketID string, todos []model.Todo) error
}

// TicketCache manages the local ticket cache.
type TicketCache interface {
	CacheTickets(tickets []model.Ticket) error
	GetCachedTickets() ([]model.Ticket, error)
	CacheEpicChildren(epicKey string, tickets []model.Ticket) error
	GetAllCachedEpicChildren() (map[string][]model.Ticket, error)
}

// MetaStore combines all store operations. Consumers should prefer the
// narrower interfaces (TagStore, TodoStore, TicketCache) where possible.
type MetaStore interface {
	TagStore
	TodoStore
	TicketCache
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

func (s *SqliteMetaStore) GetUniqueTags() ([]string, error) {
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

func (s *SqliteMetaStore) GetTodos(ticketID string) ([]model.Todo, error) {
	rows, err := s.db.Query(
		`SELECT id, title, done FROM issue_todos WHERE ticket_id = ? ORDER BY id ASC`,
		ticketID,
	)
	if err != nil {
		return nil, fmt.Errorf("get todos for %s: %w", ticketID, err)
	}
	defer rows.Close()

	var todos []model.Todo
	for rows.Next() {
		var id int
		var title string
		var done int
		if err := rows.Scan(&id, &title, &done); err != nil {
			return nil, fmt.Errorf("get todos for %s: scan: %w", ticketID, err)
		}
		todos = append(todos, model.Todo{ID: id, Title: title, Done: done != 0})
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("get todos for %s: rows: %w", ticketID, err)
	}

	return todos, nil
}

func (s *SqliteMetaStore) SaveTodos(ticketID string, todos []model.Todo) error {
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("save todos for %s: begin tx: %w", ticketID, err)
	}
	defer tx.Rollback()

	if _, err := tx.Exec(`DELETE FROM issue_todos WHERE ticket_id = ?`, ticketID); err != nil {
		return fmt.Errorf("save todos for %s: delete old: %w", ticketID, err)
	}

	if len(todos) > 0 {
		stmt, err := tx.Prepare(`INSERT INTO issue_todos (ticket_id, title, done) VALUES (?, ?, ?)`)
		if err != nil {
			return fmt.Errorf("save todos for %s: prepare: %w", ticketID, err)
		}
		defer stmt.Close()

		for _, t := range todos {
			done := 0
			if t.Done {
				done = 1
			}
			if _, err := stmt.Exec(ticketID, t.Title, done); err != nil {
				return fmt.Errorf("save todo %q for %s: %w", t.Title, ticketID, err)
			}
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("save todos for %s: commit: %w", ticketID, err)
	}
	return nil
}

func (s *SqliteMetaStore) CacheTickets(tickets []model.Ticket) error {
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("cache tickets: begin tx: %w", err)
	}
	defer tx.Rollback()

	if _, err = tx.Exec(`DELETE FROM tickets WHERE epic_id IS NULL`); err != nil {
		return fmt.Errorf("cache tickets: clear old: %w", err)
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
		return fmt.Errorf("cache tickets: prepare: %w", err)
	}
	defer stmt.Close()

	for _, t := range tickets {
		if _, err := stmt.Exec(ticketInsertValues(t)...); err != nil {
			return fmt.Errorf("cache ticket %s: %w", t.ID, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("cache tickets: commit: %w", err)
	}
	return nil
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
		return nil, fmt.Errorf("get cached tickets: %w", err)
	}
	defer rows.Close()

	var tickets []model.Ticket
	for rows.Next() {
		t, err := scanTicketWithTags(rows)
		if err != nil {
			return nil, fmt.Errorf("get cached tickets: scan: %w", err)
		}
		tickets = append(tickets, t)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("get cached tickets: rows: %w", err)
	}

	return tickets, nil
}

func (s *SqliteMetaStore) CacheEpicChildren(epicKey string, tickets []model.Ticket) error {
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("cache epic %s children: begin tx: %w", epicKey, err)
	}
	defer tx.Rollback()

	if _, err = tx.Exec(`DELETE FROM tickets WHERE epic_id = ?`, epicKey); err != nil {
		return fmt.Errorf("cache epic %s children: clear old: %w", epicKey, err)
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
			return fmt.Errorf("cache epic %s children: prepare: %w", epicKey, err)
		}
		defer stmt.Close()

		for _, t := range tickets {
			args := append(ticketInsertValues(t), epicKey)
			if _, err := stmt.Exec(args...); err != nil {
				return fmt.Errorf("cache epic %s child %s: %w", epicKey, t.ID, err)
			}
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("cache epic %s children: commit: %w", epicKey, err)
	}
	return nil
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
		return nil, fmt.Errorf("get epic children: %w", err)
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
			return nil, fmt.Errorf("get epic children: scan: %w", err)
		}
		t.EpicID = &epicID
		t.CreatedAt = parseTime(createdAt)
		t.UpdatedAt = parseTime(updatedAt)
		if tags.Valid && tags.String != "" {
			t.Tags = strings.Split(tags.String, ",")
		}
		result[epicID] = append(result[epicID], t)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("get epic children: rows: %w", err)
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
	t.CreatedAt = parseTime(createdAt)
	t.UpdatedAt = parseTime(updatedAt)
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

// parseTime parses an RFC3339 timestamp, logging a warning on failure.
func parseTime(s string) time.Time {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		logger.Log.Warn(fmt.Sprintf("failed to parse time %q: %v", s, err))
		return time.Time{}
	}
	return t
}
