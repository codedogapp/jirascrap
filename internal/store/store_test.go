package store

import (
	"database/sql"
	"testing"

	"github.com/codedogapp/jirascrap/internal/store/migrations"
	"github.com/pressly/goose/v3"
	_ "modernc.org/sqlite"
)

type testStores struct {
	Tags    *SqliteTagStore
	Todos   *SqliteTodoStore
	Tickets *SqliteTicketCache
}

func setupTestDB(t *testing.T) testStores {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() {
		db.Close()
	})

	goose.SetBaseFS(migrations.FS)
	if err := goose.SetDialect("sqlite"); err != nil {
		t.Fatalf("set dialect: %v", err)
	}
	goose.SetLogger(goose.NopLogger())
	if err := goose.Up(db, "."); err != nil {
		t.Fatalf("run migrations: %v", err)
	}

	return testStores{
		Tags:    NewSqliteTagStore(db),
		Todos:   NewSqliteTodoStore(db),
		Tickets: NewSqliteTicketCache(db),
	}
}
