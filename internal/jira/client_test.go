package jira

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/codedogapp/jirascrap/internal/model"
)

func testClient(domain string, httpClient *http.Client) *Client {
	return &Client{
		domain:        domain,
		email:         "test@example.com",
		token:         "test-token",
		http:          httpClient,
		maxResults:    100,
		maxConcurrent: 5,
	}
}

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

	client := testClient(server.URL, server.Client())

	tickets, err := client.FetchTickets(context.Background())
	if err != nil {
		t.Fatalf("FetchTickets: %v", err)
	}

	// Verify request body
	if receivedReq.JQL != defaultJQL {
		t.Errorf("request JQL = %q, want %q", receivedReq.JQL, defaultJQL)
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
		domain:        server.URL,
		email:         "user@test.com",
		token:         "bad-token",
		http:          server.Client(),
		maxResults:    100,
		maxConcurrent: 5,
	}

	_, err := client.FetchTickets(context.Background())
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
		domain:        server.URL,
		email:         "user@test.com",
		token:         "token",
		http:          server.Client(),
		maxResults:    100,
		maxConcurrent: 5,
	}

	tickets, err := client.FetchTickets(context.Background())
	if err != nil {
		t.Fatalf("FetchTickets: %v", err)
	}
	if len(tickets) != 0 {
		t.Errorf("expected 0 tickets, got %d", len(tickets))
	}
}

func TestFetchTransitions_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("method = %s, want GET", r.Method)
		}
		if r.URL.Path != "/rest/api/3/issue/PROJ-1/transitions" {
			t.Errorf("path = %q", r.URL.Path)
		}

		user, pass, ok := r.BasicAuth()
		if !ok || user != "test@example.com" || pass != "test-token" {
			t.Errorf("auth: user=%q pass=%q ok=%v", user, pass, ok)
		}

		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{
			"transitions": [
				{
					"id": "21",
					"name": "Done",
					"to": {"name": "Done", "statusCategory": {"name": "Done"}}
				},
				{
					"id": "31",
					"name": "In Progress",
					"to": {"name": "In Progress", "statusCategory": {"name": "In Progress"}}
				}
			]
		}`))
	}))
	defer server.Close()

	client := testClient(server.URL, server.Client())

	transitions, err := client.FetchTransitions(context.Background(), "PROJ-1")
	if err != nil {
		t.Fatalf("FetchTransitions: %v", err)
	}
	if len(transitions) != 2 {
		t.Fatalf("expected 2 transitions, got %d", len(transitions))
	}

	if transitions[0].ID != "21" || transitions[0].Name != "Done" {
		t.Errorf("transition[0] = %+v", transitions[0])
	}
	if transitions[0].ToStatus != "Done" || transitions[0].ToStatusCategory != "Done" {
		t.Errorf("transition[0] to = %q / %q", transitions[0].ToStatus, transitions[0].ToStatusCategory)
	}
	if transitions[1].ID != "31" || transitions[1].Name != "In Progress" {
		t.Errorf("transition[1] = %+v", transitions[1])
	}
}

func TestFetchTransitions_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"errorMessages":["Issue not found"]}`))
	}))
	defer server.Close()

	client := &Client{
		domain:        server.URL,
		email:         "user@test.com",
		token:         "token",
		http:          server.Client(),
		maxResults:    100,
		maxConcurrent: 5,
	}

	_, err := client.FetchTransitions(context.Background(), "BAD-1")
	if err == nil {
		t.Fatal("expected error for 404 response")
	}
}

func TestDoTransition_Success(t *testing.T) {
	var receivedBody transitionRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/rest/api/3/issue/PROJ-1/transitions" {
			t.Errorf("path = %q", r.URL.Path)
		}

		json.NewDecoder(r.Body).Decode(&receivedBody)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client := testClient(server.URL, server.Client())

	err := client.DoTransition(context.Background(), "PROJ-1", "21")
	if err != nil {
		t.Fatalf("DoTransition: %v", err)
	}
	if receivedBody.Transition.ID != "21" {
		t.Errorf("transition ID = %q, want 21", receivedBody.Transition.ID)
	}
}

func TestDoTransition_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"errorMessages":["Invalid transition"]}`))
	}))
	defer server.Close()

	client := &Client{
		domain:        server.URL,
		email:         "user@test.com",
		token:         "token",
		http:          server.Client(),
		maxResults:    100,
		maxConcurrent: 5,
	}

	err := client.DoTransition(context.Background(), "PROJ-1", "999")
	if err == nil {
		t.Fatal("expected error for 400 response")
	}
}

func TestRetry_ServerError(t *testing.T) {
	var attempts atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := attempts.Add(1)
		if n < 3 {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"error":"internal"}`))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(searchResponse{Issues: []issue{}})
	}))
	defer server.Close()

	client := &Client{
		domain:        server.URL,
		email:         "user@test.com",
		token:         "token",
		http:          server.Client(),
		maxResults:    100,
		maxConcurrent: 5,
	}

	tickets, err := client.FetchTickets(context.Background())
	if err != nil {
		t.Fatalf("expected success after retries, got: %v", err)
	}
	if len(tickets) != 0 {
		t.Errorf("expected 0 tickets, got %d", len(tickets))
	}
	if got := attempts.Load(); got != 3 {
		t.Errorf("expected 3 attempts, got %d", got)
	}
}

func TestRetry_429_WithRetryAfter(t *testing.T) {
	var attempts atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := attempts.Add(1)
		if n == 1 {
			w.Header().Set("Retry-After", "1")
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(searchResponse{Issues: []issue{}})
	}))
	defer server.Close()

	client := &Client{
		domain:        server.URL,
		email:         "user@test.com",
		token:         "token",
		http:          server.Client(),
		maxResults:    100,
		maxConcurrent: 5,
	}

	_, err := client.FetchTickets(context.Background())
	if err != nil {
		t.Fatalf("expected success after retry, got: %v", err)
	}
	if got := attempts.Load(); got != 2 {
		t.Errorf("expected 2 attempts, got %d", got)
	}
}

func TestRetry_NoRetryOn4xx(t *testing.T) {
	var attempts atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts.Add(1)
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"message":"Unauthorized"}`))
	}))
	defer server.Close()

	client := &Client{
		domain:        server.URL,
		email:         "user@test.com",
		token:         "bad-token",
		http:          server.Client(),
		maxResults:    100,
		maxConcurrent: 5,
	}

	_, err := client.FetchTickets(context.Background())
	if err == nil {
		t.Fatal("expected error for 401")
	}
	if got := attempts.Load(); got != 1 {
		t.Errorf("expected 1 attempt (no retry on 4xx), got %d", got)
	}
}

func TestRetry_ContextCancelled(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	client := &Client{
		domain:        server.URL,
		email:         "user@test.com",
		token:         "token",
		http:          server.Client(),
		maxResults:    100,
		maxConcurrent: 5,
	}

	_, err := client.FetchTickets(ctx)
	if err == nil {
		t.Fatal("expected error for cancelled context")
	}
}

func TestFetchAllEpicChildren_Concurrent(t *testing.T) {
	var requestCount atomic.Int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount.Add(1)
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{
"issues": [{
"key": "CHILD-1",
"fields": {
"summary": "Child ticket",
"reporter": {"displayName": "Bob"},
"status": {"name": "Open", "statusCategory": {"name": "To Do"}},
"priority": {"name": "Medium"},
"created": "2024-01-01T00:00:00.000+0000",
"updated": "2024-01-02T00:00:00.000+0000",
"description": null
}
}]
}`))
	}))
	defer server.Close()

	client := testClient(server.URL, server.Client())
	client.maxConcurrent = 2

	tickets := []model.Ticket{
		{ID: "EPIC-1", Type: "Epic"},
		{ID: "EPIC-2", Type: "Epic"},
		{ID: "EPIC-3", Type: "Epic"},
		{ID: "TASK-1", Type: "Task"}, // not an epic, should be skipped
	}

	result, err := client.FetchAllEpicChildren(context.Background(), tickets)
	if err != nil {
		t.Fatalf("FetchAllEpicChildren: %v", err)
	}

	// Should have fetched 3 epics (not the task)
	if int(requestCount.Load()) != 3 {
		t.Errorf("expected 3 requests, got %d", requestCount.Load())
	}

	if len(result) != 3 {
		t.Fatalf("expected 3 epic entries, got %d", len(result))
	}

	for _, epicKey := range []string{"EPIC-1", "EPIC-2", "EPIC-3"} {
		children, ok := result[epicKey]
		if !ok {
			t.Errorf("missing children for %s", epicKey)
			continue
		}
		if len(children) != 1 || children[0].ID != "CHILD-1" {
			t.Errorf("unexpected children for %s: %+v", epicKey, children)
		}
	}
}

func TestFetchAllEpicChildren_CollectsErrors(t *testing.T) {
	callCount := atomic.Int32{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := callCount.Add(1)
		if n == 2 {
			w.WriteHeader(http.StatusForbidden)
			w.Write([]byte("forbidden"))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"issues": []}`))
	}))
	defer server.Close()

	client := testClient(server.URL, server.Client())
	client.maxConcurrent = 1 // serial to make failure deterministic

	tickets := []model.Ticket{
		{ID: "EPIC-1", Type: "Epic"},
		{ID: "EPIC-2", Type: "Epic"},
	}

	result, err := client.FetchAllEpicChildren(context.Background(), tickets)
	if err == nil {
		t.Fatal("expected error from failed epic fetch")
	}

	// Should still have partial results
	if len(result) != 1 {
		t.Errorf("expected 1 successful result, got %d", len(result))
	}
}

func TestFetchComments_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("method = %s, want GET", r.Method)
		}
		if r.URL.Path != "/rest/api/3/issue/PROJ-1/comment" {
			t.Errorf("path = %q", r.URL.Path)
		}
		if r.URL.Query().Get("maxResults") != "20" {
			t.Errorf("maxResults = %q", r.URL.Query().Get("maxResults"))
		}

		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{
			"total": 2,
			"comments": [
				{
					"id": "100",
					"author": {"displayName": "Alice"},
					"created": "2024-03-01T10:00:00.000+0000",
					"body": {"type":"doc","version":1,"content":[{"type":"paragraph","content":[{"type":"text","text":"First comment"}]}]}
				},
				{
					"id": "101",
					"author": {"displayName": "Bob"},
					"created": "2024-03-02T12:00:00.000+0000",
					"body": {"type":"doc","version":1,"content":[{"type":"paragraph","content":[{"type":"text","text":"Second comment"}]}]}
				}
			]
		}`))
	}))
	defer server.Close()

	client := testClient(server.URL, server.Client())

	comments, total, err := client.FetchComments(context.Background(), "PROJ-1", 20)
	if err != nil {
		t.Fatalf("FetchComments: %v", err)
	}
	if total != 2 {
		t.Errorf("total = %d, want 2", total)
	}
	if len(comments) != 2 {
		t.Fatalf("expected 2 comments, got %d", len(comments))
	}

	if comments[0].ID != "100" {
		t.Errorf("comments[1].ID = %q, want 100", comments[1].ID)
	}
	if comments[0].Author != "Alice" {
		t.Errorf("comments[1].Author = %q", comments[1].Author)
	}
	if comments[1].ID != "101" {
		t.Errorf("comments[0].ID = %q, want 101 (reversed order)", comments[0].ID)
	}
	if comments[1].Author != "Bob" {
		t.Errorf("comments[0].Author = %q", comments[0].Author)
	}
}

func TestFetchComments_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"errorMessages":["Issue not found"]}`))
	}))
	defer server.Close()

	client := testClient(server.URL, server.Client())

	_, _, err := client.FetchComments(context.Background(), "BAD-1", 20)
	if err == nil {
		t.Fatal("expected error for 404 response")
	}
}

func TestPostComment_Success(t *testing.T) {
	var receivedBody map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/rest/api/3/issue/PROJ-1/comment" {
			t.Errorf("path = %q", r.URL.Path)
		}

		json.NewDecoder(r.Body).Decode(&receivedBody)
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{"id":"200"}`))
	}))
	defer server.Close()

	client := testClient(server.URL, server.Client())

	body := BuildCommentADF("Hello", nil)
	err := client.PostComment(context.Background(), "PROJ-1", body)
	if err != nil {
		t.Fatalf("PostComment: %v", err)
	}

	if receivedBody["body"] == nil {
		t.Error("request body missing 'body' field")
	}
}

func TestPostComment_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"errorMessages":["Invalid body"]}`))
	}))
	defer server.Close()

	client := testClient(server.URL, server.Client())

	err := client.PostComment(context.Background(), "PROJ-1", map[string]any{})
	if err == nil {
		t.Fatal("expected error for 400 response")
	}
}

func TestSearchUsers_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("method = %s, want GET", r.Method)
		}
		if !strings.Contains(r.URL.Path, "/rest/api/3/user/search") {
			t.Errorf("path = %q", r.URL.Path)
		}
		if r.URL.Query().Get("query") != "ali" {
			t.Errorf("query = %q, want 'ali'", r.URL.Query().Get("query"))
		}

		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`[
			{"accountId": "abc-123", "displayName": "Alice"},
			{"accountId": "def-456", "displayName": "Alicia"}
		]`))
	}))
	defer server.Close()

	client := testClient(server.URL, server.Client())

	users, err := client.SearchUsers(context.Background(), "ali")
	if err != nil {
		t.Fatalf("SearchUsers: %v", err)
	}
	if len(users) != 2 {
		t.Fatalf("expected 2 users, got %d", len(users))
	}
	if users[0].AccountID != "abc-123" || users[0].DisplayName != "Alice" {
		t.Errorf("users[0] = %+v", users[0])
	}
	if users[1].AccountID != "def-456" || users[1].DisplayName != "Alicia" {
		t.Errorf("users[1] = %+v", users[1])
	}
}

func TestSearchUsers_URLEncoding(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// The query should be properly URL-encoded
		if r.URL.Query().Get("query") != "foo bar" {
			t.Errorf("query = %q, want 'foo bar'", r.URL.Query().Get("query"))
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`[]`))
	}))
	defer server.Close()

	client := testClient(server.URL, server.Client())

	users, err := client.SearchUsers(context.Background(), "foo bar")
	if err != nil {
		t.Fatalf("SearchUsers: %v", err)
	}
	if len(users) != 0 {
		t.Errorf("expected 0 users, got %d", len(users))
	}
}

func TestSearchUsers_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"message":"Unauthorized"}`))
	}))
	defer server.Close()

	client := testClient(server.URL, server.Client())

	_, err := client.SearchUsers(context.Background(), "test")
	if err == nil {
		t.Fatal("expected error for 401 response")
	}
}
