// genschema applies goose migrations to an in-memory SQLite database
// and writes the resulting schema to internal/store/schema.sql for sqlc.
package main

import (
	"database/sql"
	"fmt"
	"os"
	"strings"

	"github.com/codedogapp/jirascrap/internal/store/migrations"
	"github.com/pressly/goose/v3"
	_ "modernc.org/sqlite"
)

const header = `-- Code generated from goose migrations. DO NOT EDIT.
-- Regenerate with: go run ./cmd/genschema

`

func main() {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		fatal("open db: %v", err)
	}
	defer db.Close()

	goose.SetBaseFS(migrations.FS)
	if err := goose.SetDialect("sqlite"); err != nil {
		fatal("set dialect: %v", err)
	}
	goose.SetLogger(goose.NopLogger())
	if err := goose.Up(db, "."); err != nil {
		fatal("run migrations: %v", err)
	}

	rows, err := db.Query(`
		SELECT sql FROM sqlite_master
		WHERE type IN ('table', 'index')
		  AND name NOT LIKE 'goose_%'
		  AND name NOT LIKE 'sqlite_%'
		  AND sql IS NOT NULL
		ORDER BY type DESC, name ASC
	`)
	if err != nil {
		fatal("query schema: %v", err)
	}
	defer rows.Close()

	var stmts []string
	for rows.Next() {
		var stmt string
		if err := rows.Scan(&stmt); err != nil {
			fatal("scan: %v", err)
		}
		stmts = append(stmts, stmt+";\n")
	}
	if err := rows.Err(); err != nil {
		fatal("rows: %v", err)
	}

	out := header + strings.Join(stmts, "\n")
	if err := os.WriteFile("internal/store/schema.sql", []byte(out), 0600); err != nil {
		fatal("write: %v", err)
	}
	fmt.Println("schema.sql generated")
}

func fatal(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "genschema: "+format+"\n", args...)
	os.Exit(1)
}
