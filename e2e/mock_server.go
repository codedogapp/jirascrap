package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"
)

type response struct {
	Issues []issue `json:"issues"`
}

type issue struct {
	Key    string `json:"key"`
	Fields fields `json:"fields"`
}

type fields struct {
	Summary     string `json:"summary"`
	Description any    `json:"description"`
	Reporter    named  `json:"reporter"`
	Status      status `json:"status"`
	Priority    named  `json:"priority"`
	Created     string `json:"created"`
	Updated     string `json:"updated"`
}

type named struct {
	Name        string `json:"name,omitempty"`
	DisplayName string `json:"displayName,omitempty"`
}

type status struct {
	Name           string `json:"name"`
	StatusCategory named  `json:"statusCategory"`
}

func adfDoc(text string) any {
	return map[string]any{
		"type":    "doc",
		"version": 1,
		"content": []any{
			map[string]any{
				"type": "paragraph",
				"content": []any{
					map[string]any{"type": "text", "text": text},
				},
			},
		},
	}
}

var data = response{
	Issues: []issue{
		{
			Key: "PROJ-101",
			Fields: fields{
				Summary: "Fix authentication timeout",
				Description: adfDoc(
					"Users are getting logged out after 5 minutes. We need to extend the session TTL and add a refresh token flow.",
				),
				Reporter: named{DisplayName: "Alice Chen"},
				Status:   status{Name: "In Progress", StatusCategory: named{Name: "In Progress"}},
				Priority: named{Name: "High"},
				Created:  "2024-11-01T09:00:00.000+0000",
				Updated:  "2024-11-15T14:30:00.000+0000",
			},
		},
		{
			Key: "PROJ-102",
			Fields: fields{
				Summary: "Add dark mode support",
				Description: adfDoc(
					"Implement a dark color scheme that respects the user's OS preference. Should also have a manual toggle in settings.",
				),
				Reporter: named{DisplayName: "Bob Martinez"},
				Status:   status{Name: "To Do", StatusCategory: named{Name: "To Do"}},
				Priority: named{Name: "Medium"},
				Created:  "2024-11-05T11:00:00.000+0000",
				Updated:  "2024-11-10T08:15:00.000+0000",
			},
		},
		{
			Key: "PROJ-103",
			Fields: fields{
				Summary: "Database migration fails on PostgreSQL 16",
				Description: adfDoc(
					"The goose migration 005 uses a deprecated syntax that PG16 no longer accepts. Need to rewrite the ALTER TABLE statement.",
				),
				Reporter: named{DisplayName: "Carol Wu"},
				Status:   status{Name: "Done", StatusCategory: named{Name: "Done"}},
				Priority: named{Name: "Highest"},
				Created:  "2024-10-20T16:45:00.000+0000",
				Updated:  "2024-11-12T10:00:00.000+0000",
			},
		},
		{
			Key: "PROJ-104",
			Fields: fields{
				Summary: "Optimize search indexing",
				Description: adfDoc(
					"Full-text search is slow on large datasets. Consider switching to trigram indexes or adding ElasticSearch.",
				),
				Reporter: named{DisplayName: "Dave Kim"},
				Status:   status{Name: "In Review", StatusCategory: named{Name: "In Progress"}},
				Priority: named{Name: "Low"},
				Created:  "2024-11-08T13:20:00.000+0000",
				Updated:  "2024-11-14T17:00:00.000+0000",
			},
		},
		{
			Key: "PROJ-105",
			Fields: fields{
				Summary: "Update API documentation",
				Description: adfDoc(
					"The REST API docs are out of date. Endpoints added in v2.3 are missing. Regenerate from OpenAPI spec.",
				),
				Reporter: named{DisplayName: "Eve Johnson"},
				Status:   status{Name: "Blocked", StatusCategory: named{Name: "Blocked"}},
				Priority: named{Name: "Lowest"},
				Created:  "2024-11-02T10:00:00.000+0000",
				Updated:  "2024-11-13T09:30:00.000+0000",
			},
		},
	},
}

type transitionsResponse struct {
	Transitions []transition `json:"transitions"`
}

type transition struct {
	ID   string       `json:"id"`
	Name string       `json:"name"`
	To   transitionTo `json:"to"`
}

type transitionTo struct {
	Name           string `json:"name"`
	StatusCategory named  `json:"statusCategory"`
}

type commentsResponse struct {
	Total    int       `json:"total"`
	Comments []comment `json:"comments"`
}

type comment struct {
	ID      string `json:"id"`
	Author  named  `json:"author"`
	Created string `json:"created"`
	Body    any    `json:"body"`
}

var commentsByIssue = map[string][]comment{
	"PROJ-101": {
		{
			ID:      "10001",
			Author:  named{DisplayName: "Alice Chen"},
			Created: "2024-11-10T09:30:00.000+0000",
			Body: adfDoc(
				"I've confirmed the session TTL is set to 5 minutes in the auth config. We need to bump it to at least 30 min.",
			),
		},
		{
			ID:      "10002",
			Author:  named{DisplayName: "Bob Martinez"},
			Created: "2024-11-11T14:15:00.000+0000",
			Body: adfDoc(
				"I'll handle the refresh token implementation. Should we use sliding window or fixed expiry?",
			),
		},
		{
			ID:      "10003",
			Author:  named{DisplayName: "Alice Chen"},
			Created: "2024-11-12T10:00:00.000+0000",
			Body:    adfDoc("Let's go with sliding window — better UX for active users."),
		},
	},
	"PROJ-102": {
		{
			ID:      "10004",
			Author:  named{DisplayName: "Eve Johnson"},
			Created: "2024-11-06T11:00:00.000+0000",
			Body:    adfDoc("Make sure we also update the syntax highlighting theme for dark mode."),
		},
	},
	"PROJ-104": {
		{
			ID:      "10005",
			Author:  named{DisplayName: "Dave Kim"},
			Created: "2024-11-09T16:00:00.000+0000",
			Body: adfDoc(
				"Benchmarks show trigram indexes give us 3x improvement on the test dataset. Going with that approach.",
			),
		},
		{
			ID:      "10006",
			Author:  named{DisplayName: "Carol Wu"},
			Created: "2024-11-10T09:00:00.000+0000",
			Body:    adfDoc("LGTM. Let's avoid ElasticSearch for now — trigrams should be sufficient for our scale."),
		},
	},
}

type userEntry struct {
	AccountID   string `json:"accountId"`
	DisplayName string `json:"displayName"`
}

var mockUsers = []userEntry{
	{AccountID: "user-001", DisplayName: "Alice Chen"},
	{AccountID: "user-002", DisplayName: "Bob Martinez"},
	{AccountID: "user-003", DisplayName: "Carol Wu"},
	{AccountID: "user-004", DisplayName: "Dave Kim"},
	{AccountID: "user-005", DisplayName: "Eve Johnson"},
}

var nextCommentID = 20000

// Per-issue transitions based on current status category.
var transitionsByStatus = map[string][]transition{
	"In Progress": {
		{ID: "21", Name: "Done", To: transitionTo{Name: "Done", StatusCategory: named{Name: "Done"}}},
		{ID: "11", Name: "To Do", To: transitionTo{Name: "To Do", StatusCategory: named{Name: "To Do"}}},
		{ID: "41", Name: "In Review", To: transitionTo{Name: "In Review", StatusCategory: named{Name: "In Progress"}}},
	},
	"To Do": {
		{
			ID:   "31",
			Name: "In Progress",
			To:   transitionTo{Name: "In Progress", StatusCategory: named{Name: "In Progress"}},
		},
		{ID: "21", Name: "Done", To: transitionTo{Name: "Done", StatusCategory: named{Name: "Done"}}},
	},
	"Done": {
		{ID: "11", Name: "Reopen", To: transitionTo{Name: "To Do", StatusCategory: named{Name: "To Do"}}},
	},
	"Blocked": {
		{
			ID:   "31",
			Name: "In Progress",
			To:   transitionTo{Name: "In Progress", StatusCategory: named{Name: "In Progress"}},
		},
		{ID: "11", Name: "To Do", To: transitionTo{Name: "To Do", StatusCategory: named{Name: "To Do"}}},
	},
}

func findIssue(key string) *issue {
	for i := range data.Issues {
		if data.Issues[i].Key == key {
			return &data.Issues[i]
		}
	}
	return nil
}

func splitPath(p string) []string {
	var parts []string
	for _, s := range strings.Split(p, "/") {
		if s != "" {
			parts = append(parts, s)
		}
	}
	return parts
}

func main() {
	port := "18932"
	if p := os.Getenv("PORT"); p != "" {
		port = p
	}

	http.HandleFunc("/rest/api/3/search/jql", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(data)
	})

	// GET: return available transitions; POST: execute transition
	// GET: return comments
	http.HandleFunc("/rest/api/3/issue/", func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		const prefix = "/rest/api/3/issue/"
		trimmed := path[len(prefix):]
		parts := splitPath(trimmed)

		if len(parts) != 2 {
			http.NotFound(w, r)
			return
		}

		issueKey := parts[0]
		iss := findIssue(issueKey)
		if iss == nil {
			http.NotFound(w, r)
			return
		}

		switch parts[1] {
		case "comment":
			switch r.Method {
			case "GET":
				comments := commentsByIssue[issueKey]
				if comments == nil {
					comments = []comment{}
				}
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(commentsResponse{
					Total:    len(comments),
					Comments: comments,
				})

			case "POST":
				var body struct {
					Body any `json:"body"`
				}
				_ = json.NewDecoder(r.Body).Decode(&body)

				newComment := comment{
					ID:      fmt.Sprintf("%d", nextCommentID),
					Author:  named{DisplayName: "Demo User"},
					Created: time.Now().Format("2006-01-02T15:04:05.000-0700"),
					Body:    body.Body,
				}
				nextCommentID++
				commentsByIssue[issueKey] = append(commentsByIssue[issueKey], newComment)
				w.WriteHeader(http.StatusCreated)

			default:
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			}

		case "transitions":
			switch r.Method {
			case "GET":
				statusCat := iss.Fields.Status.StatusCategory.Name
				transitions := transitionsByStatus[statusCat]
				if transitions == nil {
					transitions = []transition{}
				}
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(transitionsResponse{Transitions: transitions})

			case "POST":
				var body struct {
					Transition struct {
						ID string `json:"id"`
					} `json:"transition"`
				}
				_ = json.NewDecoder(r.Body).Decode(&body)

				statusCat := iss.Fields.Status.StatusCategory.Name
				for _, t := range transitionsByStatus[statusCat] {
					if t.ID == body.Transition.ID {
						iss.Fields.Status.Name = t.To.Name
						iss.Fields.Status.StatusCategory = t.To.StatusCategory
						break
					}
				}
				w.WriteHeader(http.StatusNoContent)

			default:
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			}

		default:
			http.NotFound(w, r)
		}
	})

	http.HandleFunc("/rest/api/3/user/search", func(w http.ResponseWriter, r *http.Request) {
		query := strings.ToLower(r.URL.Query().Get("query"))
		var results []userEntry
		for _, u := range mockUsers {
			if strings.Contains(strings.ToLower(u.DisplayName), query) {
				results = append(results, u)
			}
		}
		if results == nil {
			results = []userEntry{}
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(results)
	})

	fmt.Fprintf(os.Stderr, "Mock Jira server listening on :%s\n", port)
	srv := &http.Server{
		Addr:              ":" + port,
		ReadHeaderTimeout: 10 * time.Second,
	}
	if err := srv.ListenAndServe(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
