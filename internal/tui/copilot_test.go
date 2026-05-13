package tui

import (
	"strings"
	"testing"

	"github.com/codedogapp/jirascrap/internal/model"
)

func TestBuildCopilotPrompt_Basic(t *testing.T) {
	ticket := model.Ticket{
		ID:       "PROJ-42",
		Summary:  "Fix login bug",
		Status:   "In Progress",
		Priority: "High",
		Type:     "Bug",
		Reporter: "alice",
	}
	got := buildCopilotPrompt(ticket, nil)

	for _, want := range []string{
		"# PROJ-42: Fix login bug",
		"**Status:** In Progress",
		"**Priority:** High",
		"**Type:** Bug",
		"**Reporter:** alice",
		"Plan the implementation",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("expected prompt to contain %q", want)
		}
	}
}

func TestBuildCopilotPrompt_WithTodos(t *testing.T) {
	ticket := model.Ticket{ID: "T-1", Summary: "test"}
	todos := []model.Todo{
		{Title: "Write tests", Done: false},
		{Title: "Deploy", Done: true},
	}
	got := buildCopilotPrompt(ticket, todos)

	if !strings.Contains(got, "- [ ] Write tests") {
		t.Error("expected unchecked todo")
	}
	if !strings.Contains(got, "- [x] Deploy") {
		t.Error("expected checked todo")
	}
}

func TestBuildCopilotPrompt_WithEpicAndTags(t *testing.T) {
	epicID := "EPIC-1"
	ticket := model.Ticket{
		ID:      "T-1",
		Summary: "test",
		EpicID:  &epicID,
		Tags:    []string{"backend", "urgent"},
	}
	got := buildCopilotPrompt(ticket, nil)

	if !strings.Contains(got, "**Epic:** EPIC-1") {
		t.Error("expected epic in prompt")
	}
	if !strings.Contains(got, "**Tags:** backend, urgent") {
		t.Error("expected tags in prompt")
	}
}

func TestBuildCopilotPrompt_WithDescription(t *testing.T) {
	ticket := model.Ticket{
		ID:       "T-1",
		Summary:  "test",
		Markdown: "Some **description** here.",
	}
	got := buildCopilotPrompt(ticket, nil)

	if !strings.Contains(got, "## Description") {
		t.Error("expected description header")
	}
	if !strings.Contains(got, "Some **description** here.") {
		t.Error("expected markdown content in prompt")
	}
}

func TestShellQuote(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"simple", "'simple'"},
		{"with space", "'with space'"},
		{"it's", "'it'\"'\"'s'"},
		{"a;b", "'a;b'"},
		{"$(cmd)", "'$(cmd)'"},
	}
	for _, tt := range tests {
		got := shellQuote(tt.input)
		if got != tt.want {
			t.Errorf("shellQuote(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
