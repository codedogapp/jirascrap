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
	return withTx(
		s.db,
		func(q *sqlcdb.Queries) error {
			ctx := context.Background()

			if err := q.DeleteTopLevelTickets(ctx); err != nil {
				return fmt.Errorf("cache tickets: clear old: %w", err)
			}

			for _, t := range tickets {
				if err := q.UpsertTicket(ctx, ticketToUpsertParams(t, nil)); err != nil {
					return fmt.Errorf("cache ticket %s: %w", t.ID, err)
				}
			}
			return nil
		},
	)
}

func (s *SqliteTicketCache) GetCachedTickets() ([]model.Ticket, error) {
	q := sqlcdb.New(s.db)
	rows, err := q.GetCachedTickets(context.Background())
	if err != nil {
		return nil, fmt.Errorf("get cached tickets: %w", err)
	}

	tickets := make([]model.Ticket, 0, len(rows))
	for _, r := range rows {
		t, err := scanTicketRow(rowDataFromCached(r))
		if err != nil {
			return nil, fmt.Errorf("get cached tickets: %w", err)
		}
		tickets = append(tickets, t)
	}
	return tickets, nil
}

func (s *SqliteTicketCache) CacheEpicChildren(epicKey string, tickets []model.Ticket) error {
	return withTx(
		s.db,
		func(q *sqlcdb.Queries) error {
			ctx := context.Background()

			if err := q.DeleteEpicChildren(ctx, &epicKey); err != nil {
				return fmt.Errorf("cache epic %s children: clear old: %w", epicKey, err)
			}

			for _, t := range tickets {
				if err := q.UpsertTicket(ctx, ticketToUpsertParams(t, &epicKey)); err != nil {
					return fmt.Errorf("cache epic %s child %s: %w", epicKey, t.ID, err)
				}
			}
			return nil
		},
	)
}

func (s *SqliteTicketCache) GetAllCachedEpicChildren() (map[string][]model.Ticket, error) {
	q := sqlcdb.New(s.db)
	rows, err := q.GetAllEpicChildren(context.Background())
	if err != nil {
		return nil, fmt.Errorf("get epic children: %w", err)
	}

	result := make(map[string][]model.Ticket)
	for _, r := range rows {
		t, err := scanTicketRow(rowDataFromEpic(r))
		if err != nil {
			return nil, fmt.Errorf("get epic children: %w", err)
		}
		t.EpicID = r.EpicID
		result[*r.EpicID] = append(result[*r.EpicID], t)
	}

	return result, nil
}

func ticketToUpsertParams(t model.Ticket, epicID *string) sqlcdb.UpsertTicketParams {
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
		EpicID:         epicID,
	}
}

// ticketRowData holds raw DB column values for constructing a model.Ticket.
type ticketRowData struct {
	ID, Summary, Reporter          string
	Status, StatusCategory         string
	Priority, Type                 string
	CreatedAt, UpdatedAt, Markdown string
	Tags                           string
}

func rowDataFromCached(r sqlcdb.GetCachedTicketsRow) ticketRowData {
	tags, _ := r.Tags.(string)
	return ticketRowData{
		ID: r.ID, Summary: r.Summary, Reporter: r.Reporter,
		Status: r.Status, StatusCategory: r.StatusCategory,
		Priority: r.Priority, Type: r.Type,
		CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt,
		Markdown: r.Markdown, Tags: tags,
	}
}

func rowDataFromEpic(r sqlcdb.GetAllEpicChildrenRow) ticketRowData {
	tags, _ := r.Tags.(string)
	return ticketRowData{
		ID: r.ID, Summary: r.Summary, Reporter: r.Reporter,
		Status: r.Status, StatusCategory: r.StatusCategory,
		Priority: r.Priority, Type: r.Type,
		CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt,
		Markdown: r.Markdown, Tags: tags,
	}
}

// scanTicketRow converts raw DB column values into a model.Ticket.
func scanTicketRow(d ticketRowData) (model.Ticket, error) {
	t := model.Ticket{
		ID:             d.ID,
		Summary:        d.Summary,
		Reporter:       d.Reporter,
		Status:         d.Status,
		StatusCategory: d.StatusCategory,
		Priority:       d.Priority,
		Type:           d.Type,
		Markdown:       d.Markdown,
	}

	var err error
	t.CreatedAt, err = time.Parse(time.RFC3339, d.CreatedAt)
	if err != nil {
		return model.Ticket{}, err
	}
	t.UpdatedAt, err = time.Parse(time.RFC3339, d.UpdatedAt)
	if err != nil {
		return model.Ticket{}, err
	}

	if d.Tags != "" {
		t.Tags = strings.Split(d.Tags, ",")
	}
	return t, nil
}
