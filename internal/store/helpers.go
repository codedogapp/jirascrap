package store

import (
	"database/sql"
	"fmt"

	"github.com/codedogapp/jirascrap/internal/store/sqlcdb"
)

// withTx runs fn within a transaction. It commits on success and rolls back on error.
func withTx(db *sql.DB, fn func(q *sqlcdb.Queries) error) error {
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	if err := fn(sqlcdb.New(tx)); err != nil {
		return err
	}
	return tx.Commit()
}
