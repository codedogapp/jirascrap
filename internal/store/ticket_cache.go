package store

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/codedogapp/jirascrap/internal/model"
	"github.com/codedogapp/jirascrap/internal/store/sqlcdb"
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

	q := sqlcdb.New(tx)
	ctx := context.Background()

	if err = q.DeleteTopLevelTickets(ctx); err != nil {
		return fmt.Errorf("cache tickets: clear old: %w", err)
	}

	if len(tickets) == 0 {
		return tx.Commit()
	}

	for _, t := range tickets {
		if err := q.UpsertTicket(ctx, ticketToUpsertParams(t)); err != nil {
			return fmt.Errorf("cache ticket %s: %w", t.ID, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("cache tickets: commit: %w", err)
	}
	return nil
}

func (s *SqliteTicketCache) GetCachedTickets() ([]model.Ticket, error) {
	q := sqlcdb.New(s.db)
	rows, err := q.GetCachedTickets(context.Background())
	if err != nil {
		return nil, fmt.Errorf("get cached tickets: %w", err)
	}

	tickets := make([]model.Ticket, 0, len(rows))
	for _, r := range rows {
		tags, _ := r.Tags.(string)
		t, err := cachedTicketRowToModel(r.ID, r.Summary, r.Reporter, r.Status,
			r.StatusCategory, r.Priority, r.Type, r.CreatedAt, r.UpdatedAt,
			r.Markdown, tags)
		if err != nil {
			return nil, fmt.Errorf("get cached tickets: %w", err)
		}
		tickets = append(tickets, t)
	}
	return tickets, nil
}

func (s *SqliteTicketCache) CacheEpicChildren(epicKey string, tickets []model.Ticket) error {
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("cache epic %s children: begin tx: %w", epicKey, err)
	}
	defer tx.Rollback()

	q := sqlcdb.New(tx)
	ctx := context.Background()

	if err = q.DeleteEpicChildren(ctx, sql.NullString{String: epicKey, Valid: true}); err != nil {
		return fmt.Errorf("cache epic %s children: clear old: %w", epicKey, err)
	}

	for _, t := range tickets {
		if err := q.UpsertTicketWithEpic(ctx, ticketToUpsertWithEpicParams(t, epicKey)); err != nil {
			return fmt.Errorf("cache epic %s child %s: %w", epicKey, t.ID, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("cache epic %s children: commit: %w", epicKey, err)
	}
	return nil
}

func (s *SqliteTicketCache) GetAllCachedEpicChildren() (map[string][]model.Ticket, error) {
	q := sqlcdb.New(s.db)
	rows, err := q.GetAllEpicChildren(context.Background())
	if err != nil {
		return nil, fmt.Errorf("get epic children: %w", err)
	}

	result := make(map[string][]model.Ticket)
	for _, r := range rows {
		epicID := r.EpicID.String
		tags, _ := r.Tags.(string)
		t, err := cachedTicketRowToModel(r.ID, r.Summary, r.Reporter, r.Status,
			r.StatusCategory, r.Priority, r.Type, r.CreatedAt, r.UpdatedAt,
			r.Markdown, tags)
		if err != nil {
			return nil, fmt.Errorf("get epic children: %w", err)
		}
		t.EpicID = &epicID
		result[epicID] = append(result[epicID], t)
	}
	return result, nil
}

func ticketToUpsertParams(t model.Ticket) sqlcdb.UpsertTicketParams {
	typ := t.Type
	if typ == "" {
		typ = "task"
	}
	return sqlcdb.UpsertTicketParams{
		ID:             t.ID,
		Summary:        t.Summary,
		Reporter:       t.Reporter,
		Status:         t.Status,
		StatusCategory: t.StatusCategory,
		Priority:       t.Priority,
		Type:           typ,
		CreatedAt:      t.CreatedAt.Format(time.RFC3339),
		UpdatedAt:      t.UpdatedAt.Format(time.RFC3339),
		Markdown:       t.Markdown,
	}
}

func ticketToUpsertWithEpicParams(t model.Ticket, epicKey string) sqlcdb.UpsertTicketWithEpicParams {
	typ := t.Type
	if typ == "" {
		typ = "task"
	}
	return sqlcdb.UpsertTicketWithEpicParams{
		ID:             t.ID,
		Summary:        t.Summary,
		Reporter:       t.Reporter,
		Status:         t.Status,
		StatusCategory: t.StatusCategory,
		Priority:       t.Priority,
		Type:           typ,
		CreatedAt:      t.CreatedAt.Format(time.RFC3339),
		UpdatedAt:      t.UpdatedAt.Format(time.RFC3339),
		Markdown:       t.Markdown,
		EpicID:         sql.NullString{String: epicKey, Valid: true},
	}
}

func cachedTicketRowToModel(
	id, summary, reporter, status, statusCategory, priority, typ,
	createdAt, updatedAt, markdown, tags string,
) (model.Ticket, error) {
	t := model.Ticket{
		ID:             id,
		Summary:        summary,
		Reporter:       reporter,
		Status:         status,
		StatusCategory: statusCategory,
		Priority:       priority,
		Type:           typ,
		Markdown:       markdown,
	}

	var err error
	t.CreatedAt, err = parseTime(createdAt)
	if err != nil {
		return model.Ticket{}, err
	}
	t.UpdatedAt, err = parseTime(updatedAt)
	if err != nil {
		return model.Ticket{}, err
	}

	if tags != "" {
		t.Tags = strings.Split(tags, ",")
	}
	return t, nil
}
