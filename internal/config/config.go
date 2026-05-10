package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type Config struct {
	Domain           string
	Email            string
	APIToken         string
	DBPath           string
	CopilotWorkspace string
	CopilotModel     string
}

func Load() (*Config, error) {
	cfg := &Config{
		Domain:           os.Getenv("JIRA_BASE_URL"),
		Email:            os.Getenv("JIRA_EMAIL"),
		APIToken:         os.Getenv("JIRA_API_TOKEN"),
		DBPath:           os.Getenv("JIRA_DB_PATH"),
		CopilotWorkspace: os.Getenv("JIRASCRAP_COPILOT_WORKSPACE"),
		CopilotModel:     os.Getenv("JIRASCRAP_COPILOT_MODEL"),
	}

	if cfg.DBPath == "" {
		cfg.DBPath = "./data/jira.db"
	}

	if cfg.CopilotWorkspace == "" {
		cwd, _ := os.Getwd()
		cfg.CopilotWorkspace = cwd
	}

	// Expand ~ and resolve to absolute path for tmux compatibility
	if strings.HasPrefix(cfg.CopilotWorkspace, "~/") {
		home, _ := os.UserHomeDir()
		cfg.CopilotWorkspace = filepath.Join(home, cfg.CopilotWorkspace[2:])
	}
	if abs, err := filepath.Abs(cfg.CopilotWorkspace); err == nil {
		cfg.CopilotWorkspace = abs
	}

	if cfg.CopilotModel == "" {
		cfg.CopilotModel = "claude-haiku-4.5"
	}

	var errs []string
	if cfg.Domain == "" {
		errs = append(errs, "JIRA_BASE_URL is required")
	}
	if cfg.Email == "" {
		errs = append(errs, "JIRA_EMAIL is required")
	}
	if cfg.APIToken == "" {
		errs = append(errs, "JIRA_API_TOKEN is required")
	}

	if len(errs) > 0 {
		var b strings.Builder
		b.WriteString("Missing configuration: \n")
		for _, e := range errs {
			fmt.Fprintf(&b, "- %s\n", e)
		}
		b.WriteString("\nSet these in your shell environment.")
		return nil, errors.New(b.String())
	}

	return cfg, nil
}
