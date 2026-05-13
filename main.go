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

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "TUI error: %v\n", err)
		os.Exit(1)
	}

	apiClient := jira.NewClient(cfg)

	// Set up file logging
	logFile, logPath, err := logger.OpenSessionLog(cfg.LogDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: file logging disabled: %v\n", err)
	} else {
		if l, ok := logger.Log.(*logger.Logger); ok {
			l.SetOutput(logFile)
			defer l.Close()
		}
		logger.Log.Info("session started, log file: " + logPath)
	}

	sqliteDB, err := store.Open(cfg.DBPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "TUI error: %v\n", err)
		os.Exit(1)
	}
	defer sqliteDB.Close()

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
