package store

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/codedogapp/jirascrap/internal/model"
)

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

	var err error

	t.CreatedAt, err = parseTime(createdAt)
	if err != nil {
		return model.Ticket{}, fmt.Errorf("scan ticket: %w", err)
	}

	t.UpdatedAt, err = parseTime(updatedAt)
	if err != nil {
		return model.Ticket{}, fmt.Errorf("scan ticket: %w", err)
	}

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

// parseTime parses an RFC3339 timestamp, returning an error on failure.
func parseTime(s string) (time.Time, error) {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return time.Time{}, fmt.Errorf("parse time %q: %w", s, err)
	}
	return t, nil
}
