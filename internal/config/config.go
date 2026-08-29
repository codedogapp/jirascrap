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

	var errs []error
	if cfg.Domain == "" {
		errs = append(errs, fmt.Errorf("JIRA_BASE_URL is required"))
	}
	if cfg.Email == "" {
		errs = append(errs, fmt.Errorf("JIRA_EMAIL is required"))
	}
	if cfg.APIToken == "" {
		errs = append(errs, fmt.Errorf("JIRA_API_TOKEN is required"))
	}

	if len(errs) > 0 {
		return nil, fmt.Errorf(
			"missing configuration:\n%w\n\nSet these in your shell environment",
			errors.Join(errs...),
		)
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

// Validate checks semantic correctness of configuration values.
func (c *Config) Validate() error {
	var errs []error

	if !strings.HasPrefix(c.Domain, "https://") && !c.allowHTTP() {
		errs = append(errs, fmt.Errorf("JIRA_BASE_URL must use HTTPS (got %q)", c.Domain))
	}

	if strings.HasSuffix(c.Domain, "/") {
		errs = append(errs, fmt.Errorf("JIRA_BASE_URL should not have trailing slash"))
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}
	return nil
}

func (c *Config) allowHTTP() bool {
	return os.Getenv("JIRASCRAP_ALLOW_HTTP") == "1"
}
