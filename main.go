package main

import (
	"fmt"
	"os"

	"github.com/codedogapp/jirascrap/internal/config"
	"github.com/codedogapp/jirascrap/internal/jira"
	"github.com/codedogapp/jirascrap/internal/tui"
)

func main() {
	config, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "TUI error: %v\n", err)
		os.Exit(1)
	}

	apiClient := jira.NewClient(config)

	err = tui.Run(apiClient)
	if err != nil {
		fmt.Fprintf(os.Stderr, "TUI error: %v\n", err)
		os.Exit(1)
	}
}
