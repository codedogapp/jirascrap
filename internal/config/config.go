package config

import (
	"errors"
	"fmt"
	"os"
	"strings"
)

type Config struct {
	Domain   string
	Email    string
	APIToken string
	DBPath   string
}

func Load() (*Config, error) {
	cfg := &Config{
		Domain:   os.Getenv("JIRA_BASE_URL"),
		Email:    os.Getenv("JIRA_EMAIL"),
		APIToken: os.Getenv("JIRA_API_TOKEN"),
		DBPath:   os.Getenv("JIRA_DB_PATH"),
	}

	if cfg.DBPath == "" {
		cfg.DBPath = "./data/jira.db"
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
