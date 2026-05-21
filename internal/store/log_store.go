package store

import (
	"context"
	"database/sql"

	"github.com/codedogapp/jirascrap/internal/store/sqlcdb"
)

// SqliteLogStore persists log entries to the database.
type SqliteLogStore struct {
	db *sql.DB
}

func NewSqliteLogStore(db *sql.DB) *SqliteLogStore {
	return &SqliteLogStore{db: db}
}

func (s *SqliteLogStore) InsertLog(level, message string) error {
	q := sqlcdb.New(s.db)
	return q.InsertLog(context.Background(), sqlcdb.InsertLogParams{
		Level:   level,
		Message: message,
	})
}
