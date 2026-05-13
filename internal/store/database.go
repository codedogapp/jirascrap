package store

//go:generate sqlc generate -f ../../sqlc.yaml

import (
	"database/sql"
	"embed"
	"fmt"
	"time"

	"github.com/codedogapp/jirascrap/internal/logger"
	"github.com/pressly/goose/v3"
	_ "modernc.org/sqlite"
)

const SQLDriver string = "sqlite"

//go:embed migrations/*.sql
var embedMigrations embed.FS

type DB struct {
	*sql.DB
}

func Open(dbPath string) (*DB, error) {
	database, err := sql.Open(SQLDriver, dbPath)
	if err != nil {
		return nil, err
	}

	database.SetMaxOpenConns(5)
	database.SetMaxIdleConns(2)
	database.SetConnMaxLifetime(time.Hour)

	if _, err := database.Exec("PRAGMA foreign_keys = ON"); err != nil {
		return nil, fmt.Errorf("db: failed to enable foreign keys: %w", err)
	}

	goose.SetBaseFS(embedMigrations)

	err = goose.SetDialect(SQLDriver)
	goose.SetLogger(logger.GooseLoggerAdapter{})
	if err != nil {
		return nil, fmt.Errorf("db: failed to set dialect: %w", err)
	}

	err = goose.Up(database, "migrations")
	if err != nil {
		return nil, fmt.Errorf("db: failed to run migrations: %w", err)
	}

	return &DB{database}, nil
}
