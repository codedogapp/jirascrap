package store

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/codedogapp/jirascrap/internal/model"
	"github.com/codedogapp/jirascrap/internal/store/sqlcdb"
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
	q := sqlcdb.New(s.db)
	rows, err := q.GetTodosByTicket(context.Background(), ticketID)
	if err != nil {
		return nil, fmt.Errorf("get todos for %s: %w", ticketID, err)
	}

	todos := make([]model.Todo, len(rows))
	for i, r := range rows {
		todos[i] = model.Todo{
			ID:    int(r.ID),
			Title: r.Title,
			Done:  r.Done != 0,
		}
	}
	return todos, nil
}

func (s *SqliteTodoStore) SaveTodos(ticketID string, todos []model.Todo) error {
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("save todos for %s: begin tx: %w", ticketID, err)
	}
	defer tx.Rollback()

	q := sqlcdb.New(tx)
	ctx := context.Background()

	if err := q.DeleteTodosByTicket(ctx, ticketID); err != nil {
		return fmt.Errorf("save todos for %s: delete old: %w", ticketID, err)
	}

	for _, t := range todos {
		done := int64(0)
		if t.Done {
			done = 1
		}
		if err := q.InsertTodo(ctx, sqlcdb.InsertTodoParams{
			TicketID: ticketID,
			Title:    t.Title,
			Done:     done,
		}); err != nil {
			return fmt.Errorf("save todo %q for %s: %w", t.Title, ticketID, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("save todos for %s: commit: %w", ticketID, err)
	}
	return nil
}
