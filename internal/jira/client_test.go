package jira

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestFetchTickets_Success(t *testing.T) {
	rawResponse := `{
		"issues": [{
			"key": "PROJ-1",
			"fields": {
				"summary": "Fix login bug",
				"reporter": {"displayName": "Alice"},
				"status": {"name": "In Progress", "statusCategory": {"name": "In Progress"}},
				"priority": {"name": "High"},
				"created": "2024-01-15T10:00:00.000+0000",
				"updated": "2024-02-20T14:00:00.000+0000",
				"description": {
					"type": "doc",
					"version": 1,
					"content": [{"type": "paragraph", "content": [{"type": "text", "text": "Fix the thing"}]}]
				}
			}
		}]
	}`

	var receivedReq searchRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/rest/api/3/search/jql" {
			t.Errorf("path = %s, want /rest/api/3/search/jql", r.URL.Path)
		}

		user, pass, ok := r.BasicAuth()
		if !ok || user != "test@example.com" || pass != "test-token" {
			t.Errorf("auth: user=%q pass=%q ok=%v", user, pass, ok)
		}

		if ct := r.Header.Get("Content-Type"); ct != "application/json" {
			t.Errorf("Content-Type = %q", ct)
		}

		json.NewDecoder(r.Body).Decode(&receivedReq)

		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(rawResponse))
	}))
	defer server.Close()

	client := &Client{
		domain: server.URL,
		email:  "test@example.com",
		token:  "test-token",
		jql:    "project = PROJ",
		http:   server.Client(),
	}

	tickets, err := client.FetchTickets()
	if err != nil {
		t.Fatalf("FetchTickets: %v", err)
	}

	// Verify request body
	if receivedReq.JQL != "project = PROJ" {
		t.Errorf("request JQL = %q", receivedReq.JQL)
	}
	if receivedReq.MaxResults != 100 {
		t.Errorf("request MaxResults = %d", receivedReq.MaxResults)
	}

	// Verify response mapping
	if len(tickets) != 1 {
		t.Fatalf("expected 1 ticket, got %d", len(tickets))
	}
	tk := tickets[0]
	if tk.ID != "PROJ-1" {
		t.Errorf("ID = %q", tk.ID)
	}
	if tk.Summary != "Fix login bug" {
		t.Errorf("Summary = %q", tk.Summary)
	}
	if tk.Reporter != "Alice" {
		t.Errorf("Reporter = %q", tk.Reporter)
	}
	if tk.Status != "In Progress" {
		t.Errorf("Status = %q", tk.Status)
	}
	if tk.StatusCategory != "In Progress" {
		t.Errorf("StatusCategory = %q", tk.StatusCategory)
	}
	if tk.Priority != "High" {
		t.Errorf("Priority = %q", tk.Priority)
	}
	if tk.Markdown != "Fix the thing" {
		t.Errorf("Markdown = %q", tk.Markdown)
	}
}

func TestFetchTickets_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"message":"Unauthorized"}`))
	}))
	defer server.Close()

	client := &Client{
		domain: server.URL,
		email:  "user@test.com",
		token:  "bad-token",
		jql:    "project = X",
		http:   server.Client(),
	}

	_, err := client.FetchTickets()
	if err == nil {
		t.Fatal("expected error for 401 response")
	}
}

func TestFetchTickets_EmptyResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(searchResponse{Issues: []issue{}})
	}))
	defer server.Close()

	client := &Client{
		domain: server.URL,
		email:  "user@test.com",
		token:  "token",
		jql:    "project = X",
		http:   server.Client(),
	}

	tickets, err := client.FetchTickets()
	if err != nil {
		t.Fatalf("FetchTickets: %v", err)
	}
	if len(tickets) != 0 {
		t.Errorf("expected 0 tickets, got %d", len(tickets))
	}
}
