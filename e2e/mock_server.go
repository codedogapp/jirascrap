package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
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
				Summary:     "Fix authentication timeout",
				Description: adfDoc("Users are getting logged out after 5 minutes. We need to extend the session TTL and add a refresh token flow."),
				Reporter:    named{DisplayName: "Alice Chen"},
				Status:      status{Name: "In Progress", StatusCategory: named{Name: "In Progress"}},
				Priority:    named{Name: "High"},
				Created:     "2024-11-01T09:00:00.000+0000",
				Updated:     "2024-11-15T14:30:00.000+0000",
			},
		},
		{
			Key: "PROJ-102",
			Fields: fields{
				Summary:     "Add dark mode support",
				Description: adfDoc("Implement a dark color scheme that respects the user's OS preference. Should also have a manual toggle in settings."),
				Reporter:    named{DisplayName: "Bob Martinez"},
				Status:      status{Name: "To Do", StatusCategory: named{Name: "To Do"}},
				Priority:    named{Name: "Medium"},
				Created:     "2024-11-05T11:00:00.000+0000",
				Updated:     "2024-11-10T08:15:00.000+0000",
			},
		},
		{
			Key: "PROJ-103",
			Fields: fields{
				Summary:     "Database migration fails on PostgreSQL 16",
				Description: adfDoc("The goose migration 005 uses a deprecated syntax that PG16 no longer accepts. Need to rewrite the ALTER TABLE statement."),
				Reporter:    named{DisplayName: "Carol Wu"},
				Status:      status{Name: "Done", StatusCategory: named{Name: "Done"}},
				Priority:    named{Name: "Highest"},
				Created:     "2024-10-20T16:45:00.000+0000",
				Updated:     "2024-11-12T10:00:00.000+0000",
			},
		},
		{
			Key: "PROJ-104",
			Fields: fields{
				Summary:     "Optimize search indexing",
				Description: adfDoc("Full-text search is slow on large datasets. Consider switching to trigram indexes or adding ElasticSearch."),
				Reporter:    named{DisplayName: "Dave Kim"},
				Status:      status{Name: "In Review", StatusCategory: named{Name: "In Progress"}},
				Priority:    named{Name: "Low"},
				Created:     "2024-11-08T13:20:00.000+0000",
				Updated:     "2024-11-14T17:00:00.000+0000",
			},
		},
		{
			Key: "PROJ-105",
			Fields: fields{
				Summary:     "Update API documentation",
				Description: adfDoc("The REST API docs are out of date. Endpoints added in v2.3 are missing. Regenerate from OpenAPI spec."),
				Reporter:    named{DisplayName: "Eve Johnson"},
				Status:      status{Name: "Blocked", StatusCategory: named{Name: "Blocked"}},
				Priority:    named{Name: "Lowest"},
				Created:     "2024-11-02T10:00:00.000+0000",
				Updated:     "2024-11-13T09:30:00.000+0000",
			},
		},
	},
}

type transitionsResponse struct {
	Transitions []transition `json:"transitions"`
}

type transition struct {
	ID   string         `json:"id"`
	Name string         `json:"name"`
	To   transitionTo   `json:"to"`
}

type transitionTo struct {
	Name           string `json:"name"`
	StatusCategory named  `json:"statusCategory"`
}

// Per-issue transitions based on current status category.
var transitionsByStatus = map[string][]transition{
	"In Progress": {
		{ID: "21", Name: "Done", To: transitionTo{Name: "Done", StatusCategory: named{Name: "Done"}}},
		{ID: "11", Name: "To Do", To: transitionTo{Name: "To Do", StatusCategory: named{Name: "To Do"}}},
		{ID: "41", Name: "In Review", To: transitionTo{Name: "In Review", StatusCategory: named{Name: "In Progress"}}},
	},
	"To Do": {
		{ID: "31", Name: "In Progress", To: transitionTo{Name: "In Progress", StatusCategory: named{Name: "In Progress"}}},
		{ID: "21", Name: "Done", To: transitionTo{Name: "Done", StatusCategory: named{Name: "Done"}}},
	},
	"Done": {
		{ID: "11", Name: "Reopen", To: transitionTo{Name: "To Do", StatusCategory: named{Name: "To Do"}}},
	},
	"Blocked": {
		{ID: "31", Name: "In Progress", To: transitionTo{Name: "In Progress", StatusCategory: named{Name: "In Progress"}}},
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
		json.NewEncoder(w).Encode(data)
	})

	// GET: return available transitions; POST: execute transition
	http.HandleFunc("/rest/api/3/issue/", func(w http.ResponseWriter, r *http.Request) {
		// Extract issue key and check path ends with /transitions
		path := r.URL.Path
		// /rest/api/3/issue/PROJ-101/transitions
		const prefix = "/rest/api/3/issue/"
		trimmed := path[len(prefix):]
		parts := splitPath(trimmed)
		if len(parts) != 2 || parts[1] != "transitions" {
			http.NotFound(w, r)
			return
		}
		issueKey := parts[0]
		iss := findIssue(issueKey)
		if iss == nil {
			http.NotFound(w, r)
			return
		}

		switch r.Method {
		case "GET":
			statusCat := iss.Fields.Status.StatusCategory.Name
			transitions := transitionsByStatus[statusCat]
			if transitions == nil {
				transitions = []transition{}
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(transitionsResponse{Transitions: transitions})

		case "POST":
			var body struct {
				Transition struct {
					ID string `json:"id"`
				} `json:"transition"`
			}
			json.NewDecoder(r.Body).Decode(&body)

			// Find the transition and apply it
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
	})

	fmt.Fprintf(os.Stderr, "Mock Jira server listening on :%s\n", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
