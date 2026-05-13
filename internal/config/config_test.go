package config

import (
	"os"
	"strings"
	"testing"
)

func setEnv(t *testing.T, kvs map[string]string) {
	t.Helper()
	for k, v := range kvs {
		t.Setenv(k, v)
	}
}

func requiredEnv() map[string]string {
	return map[string]string{
		"JIRA_BASE_URL":  "https://example.atlassian.net",
		"JIRA_EMAIL":     "user@example.com",
		"JIRA_API_TOKEN": "secret-token",
	}
}

func TestLoad_AllRequired(t *testing.T) {
	setEnv(t, requiredEnv())

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Domain != "https://example.atlassian.net" {
		t.Errorf("Domain = %q", cfg.Domain)
	}
	if cfg.Email != "user@example.com" {
		t.Errorf("Email = %q", cfg.Email)
	}
	if cfg.APIToken != "secret-token" {
		t.Errorf("APIToken = %q", cfg.APIToken)
	}
}

func TestLoad_DefaultDBPath(t *testing.T) {
	setEnv(t, requiredEnv())
	os.Unsetenv("JIRA_DB_PATH")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.DBPath != "./data/jira.db" {
		t.Errorf("DBPath = %q", cfg.DBPath)
	}
}

func TestLoad_CustomDBPath(t *testing.T) {
	env := requiredEnv()
	env["JIRA_DB_PATH"] = "/tmp/test.db"
	setEnv(t, env)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.DBPath != "/tmp/test.db" {
		t.Errorf("DBPath = %q", cfg.DBPath)
	}
}

func TestLoad_MissingBaseURL(t *testing.T) {
	env := requiredEnv()
	delete(env, "JIRA_BASE_URL")
	setEnv(t, env)
	os.Unsetenv("JIRA_BASE_URL")

	_, err := Load()
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "JIRA_BASE_URL") {
		t.Errorf("error should mention JIRA_BASE_URL: %v", err)
	}
}

func TestLoad_MissingEmail(t *testing.T) {
	env := requiredEnv()
	delete(env, "JIRA_EMAIL")
	setEnv(t, env)
	os.Unsetenv("JIRA_EMAIL")

	_, err := Load()
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "JIRA_EMAIL") {
		t.Errorf("error should mention JIRA_EMAIL: %v", err)
	}
}

func TestLoad_MissingAPIToken(t *testing.T) {
	env := requiredEnv()
	delete(env, "JIRA_API_TOKEN")
	setEnv(t, env)
	os.Unsetenv("JIRA_API_TOKEN")

	_, err := Load()
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "JIRA_API_TOKEN") {
		t.Errorf("error should mention JIRA_API_TOKEN: %v", err)
	}
}

func TestLoad_AllMissing(t *testing.T) {
	os.Unsetenv("JIRA_BASE_URL")
	os.Unsetenv("JIRA_EMAIL")
	os.Unsetenv("JIRA_API_TOKEN")

	_, err := Load()
	if err == nil {
		t.Fatal("expected error")
	}
	errStr := err.Error()
	for _, key := range []string{"JIRA_BASE_URL", "JIRA_EMAIL", "JIRA_API_TOKEN"} {
		if !strings.Contains(errStr, key) {
			t.Errorf("error should mention %s: %v", key, err)
		}
	}
}

func TestLoad_ValidateHTTPS(t *testing.T) {
	env := requiredEnv()
	env["JIRA_BASE_URL"] = "http://example.atlassian.net"
	setEnv(t, env)

	_, err := Load()
	if err == nil {
		t.Fatal("expected error for non-HTTPS URL")
	}
	if !strings.Contains(err.Error(), "HTTPS") {
		t.Errorf("error should mention HTTPS: %v", err)
	}
}

func TestLoad_ValidateTrailingSlash(t *testing.T) {
	env := requiredEnv()
	env["JIRA_BASE_URL"] = "https://example.atlassian.net/"
	setEnv(t, env)

	_, err := Load()
	if err == nil {
		t.Fatal("expected error for trailing slash")
	}
	if !strings.Contains(err.Error(), "trailing slash") {
		t.Errorf("error should mention trailing slash: %v", err)
	}
}

func TestValidate_ValidConfig(t *testing.T) {
	cfg := &Config{
		Domain:   "https://example.atlassian.net",
		Email:    "user@example.com",
		APIToken: "token",
	}
	if err := cfg.Validate(); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}
