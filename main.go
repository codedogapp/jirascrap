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
	config, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "TUI error: %v\n", err)
		os.Exit(1)
	}

	apiClient := jira.NewClient(config)

	sqliteDB, err := store.Open(config.DBPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "TUI error: %v\n", err)
		os.Exit(1)
	}

	metaStore := store.NewSqliteMetaStore(sqliteDB.DB)

	// TODO: define config file
	err = tui.Run(apiClient, metaStore, config.Domain)
	if err != nil {
		fmt.Fprintf(os.Stderr, "TUI error: %v\n", err)
		os.Exit(1)
	}
}
