package store

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/codedogapp/jirascrap/internal/model"
)

// TicketCache manages the local ticket cache.
type TicketCache interface {
	CacheTickets(tickets []model.Ticket) error
	GetCachedTickets() ([]model.Ticket, error)
	CacheEpicChildren(epicKey string, tickets []model.Ticket) error
	GetAllCachedEpicChildren() (map[string][]model.Ticket, error)
}

// SqliteTicketCache implements TicketCache using SQLite.
type SqliteTicketCache struct {
	db *sql.DB
}

func NewSqliteTicketCache(db *sql.DB) *SqliteTicketCache {
	return &SqliteTicketCache{db: db}
}

func (s *SqliteTicketCache) CacheTickets(tickets []model.Ticket) error {
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
func (s *SqliteTicketCache) GetCachedTickets() ([]model.Ticket, error) {
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

func (s *SqliteTicketCache) CacheEpicChildren(epicKey string, tickets []model.Ticket) error {
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

func (s *SqliteTicketCache) GetAllCachedEpicChildren() (map[string][]model.Ticket, error) {
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
		var err2 error
		t.CreatedAt, err2 = parseTime(createdAt)
		if err2 != nil {
			return nil, fmt.Errorf("get epic children: %w", err2)
		}
		t.UpdatedAt, err2 = parseTime(updatedAt)
		if err2 != nil {
			return nil, fmt.Errorf("get epic children: %w", err2)
		}
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
