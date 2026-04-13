package jira

import (
	"encoding/json"
	"testing"
)

func TestJiraTime_UnmarshalJSON_Valid(t *testing.T) {
	input := `"2024-03-15T10:30:00.000+0000"`
	var jt jiraTime
	if err := json.Unmarshal([]byte(input), &jt); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if jt.IsZero() {
		t.Error("expected non-zero time")
	}
	if jt.Year() != 2024 || jt.Month() != 3 || jt.Day() != 15 {
		t.Errorf("got %v, want 2024-03-15", jt)
	}
}

func TestJiraTime_UnmarshalJSON_WithOffset(t *testing.T) {
	input := `"2024-06-01T14:00:00.000+0530"`
	var jt jiraTime
	if err := json.Unmarshal([]byte(input), &jt); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if jt.Hour() != 14 {
		t.Errorf("got hour %d, want 14", jt.Hour())
	}
}

func TestJiraTime_UnmarshalJSON_Null(t *testing.T) {
	input := `"null"`
	var jt jiraTime
	if err := json.Unmarshal([]byte(input), &jt); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !jt.IsZero() {
		t.Error("expected zero time for null")
	}
}

func TestJiraTime_UnmarshalJSON_Empty(t *testing.T) {
	input := `""`
	var jt jiraTime
	if err := json.Unmarshal([]byte(input), &jt); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !jt.IsZero() {
		t.Error("expected zero time for empty string")
	}
}

func TestJiraTime_UnmarshalJSON_Malformed(t *testing.T) {
	input := `"not-a-date"`
	var jt jiraTime
	if err := json.Unmarshal([]byte(input), &jt); err == nil {
		t.Error("expected error for malformed date")
	}
}

func TestJiraTime_UnmarshalJSON_InStruct(t *testing.T) {
	jsonStr := `{
		"summary": "test",
		"created": "2024-01-10T09:15:30.000-0500",
		"updated": "2024-02-20T16:45:00.000+0100"
	}`
	var fields struct {
		Summary string   `json:"summary"`
		Created jiraTime `json:"created"`
		Updated jiraTime `json:"updated"`
	}
	if err := json.Unmarshal([]byte(jsonStr), &fields); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fields.Created.Year() != 2024 || fields.Created.Month() != 1 {
		t.Errorf("created: got %v", fields.Created.Time)
	}
	if fields.Updated.Year() != 2024 || fields.Updated.Month() != 2 {
		t.Errorf("updated: got %v", fields.Updated.Time)
	}
}
