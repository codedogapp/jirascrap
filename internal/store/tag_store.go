package store

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/codedogapp/jirascrap/internal/store/sqlcdb"
)

// TagStore manages ticket tags/labels.
type TagStore interface {
	SaveTags(id string, tags []string) error
	GetUniqueTags() ([]string, error)
}

// SqliteTagStore implements TagStore using SQLite.
type SqliteTagStore struct {
	db *sql.DB
}

func NewSqliteTagStore(db *sql.DB) *SqliteTagStore {
	return &SqliteTagStore{db: db}
}

func (s *SqliteTagStore) SaveTags(id string, tags []string) error {
	return withTx(
		s.db,
		func(q *sqlcdb.Queries) error {
			ctx := context.Background()

			if err := q.DeleteTagsByID(ctx, id); err != nil {
				return fmt.Errorf("save tags for %s: delete old: %w", id, err)
			}

			for _, tag := range tags {
				if err := q.InsertTag(ctx, sqlcdb.InsertTagParams{ID: id, Tag: tag}); err != nil {
					return fmt.Errorf("save tag %q for %s: %w", tag, id, err)
				}
			}
			return nil
		},
	)
}

func (s *SqliteTagStore) GetUniqueTags() ([]string, error) {
	q := sqlcdb.New(s.db)
	tags, err := q.GetUniqueTags(context.Background())
	if err != nil {
		return nil, fmt.Errorf("get unique tags: %w", err)
	}

	return tags, nil
}
