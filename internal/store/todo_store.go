package store

import (
	"database/sql"
	"fmt"

	"github.com/codedogapp/jirascrap/internal/model"
)

// TodoStore manages per-ticket todo items.
type TodoStore interface {
	GetTodos(ticketID string) ([]model.Todo, error)
	SaveTodos(ticketID string, todos []model.Todo) error
}

// SqliteTodoStore implements TodoStore using SQLite.
type SqliteTodoStore struct {
	db *sql.DB
}

func NewSqliteTodoStore(db *sql.DB) *SqliteTodoStore {
	return &SqliteTodoStore{db: db}
}

func (s *SqliteTodoStore) GetTodos(ticketID string) ([]model.Todo, error) {
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

func (s *SqliteTodoStore) SaveTodos(ticketID string, todos []model.Todo) error {
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
