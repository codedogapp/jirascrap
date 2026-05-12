package main

import (
	"fmt"
	"os"

	"github.com/codedogapp/jirascrap/internal/config"
	"github.com/codedogapp/jirascrap/internal/jira"
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

	sqliteDB, err := store.Open(cfg.DBPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "TUI error: %v\n", err)
		os.Exit(1)
	}
	defer sqliteDB.Close()

	metaStore := store.NewSqliteMetaStore(sqliteDB.DB)

	err = tui.Run(apiClient, metaStore, cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "TUI error: %v\n", err)
		os.Exit(1)
	}
}
