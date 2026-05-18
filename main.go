package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/codedogapp/jirascrap/internal/config"
	"github.com/codedogapp/jirascrap/internal/jira"
	"github.com/codedogapp/jirascrap/internal/logger"
	"github.com/codedogapp/jirascrap/internal/store"
	"github.com/codedogapp/jirascrap/internal/tui"
)

// gooseLogger routes goose migration logs through the app logger.
type gooseLogger struct{}

func (gooseLogger) Fatalf(format string, v ...any) {
	logger.Log.Error(fmt.Sprintf(format, v...))
}

func (gooseLogger) Printf(format string, v ...any) {
	logger.Log.Info(fmt.Sprintf(format, v...))
}

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "TUI error: %v\n", err)
		os.Exit(1)
	}

	apiClient := jira.NewClient(cfg)

	sqliteDB, err := store.Open(cfg.DBPath, gooseLogger{})
	if err != nil {
		fmt.Fprintf(os.Stderr, "TUI error: %v\n", err)
		os.Exit(1)
	}
	defer sqliteDB.Close()

	// Wire logger to persist to SQLite
	logStore := store.NewSqliteLogStore(sqliteDB.DB)
	logger.Log.SetPersister(logStore)
	logger.Log.Info("session started")

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	tagStore := store.NewSqliteTagStore(sqliteDB.DB)
	todoStore := store.NewSqliteTodoStore(sqliteDB.DB)
	ticketCache := store.NewSqliteTicketCache(sqliteDB.DB)

	err = tui.Run(ctx, apiClient, tagStore, todoStore, ticketCache, cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "TUI error: %v\n", err)
		os.Exit(1)
	}
}
