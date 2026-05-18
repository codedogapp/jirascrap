package main

import (
	"embed"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"
)

//go:embed testdata/*.json
var fixtures embed.FS

// Types for JSON deserialization.

type issue struct {
	Key    string `json:"key"`
	Fields struct {
		Summary     string `json:"summary"`
		Description any    `json:"description"`
		Reporter    named  `json:"reporter"`
		Status      status `json:"status"`
		Priority    named  `json:"priority"`
		Created     string `json:"created"`
		Updated     string `json:"updated"`
	} `json:"fields"`
}

type named struct {
	Name        string `json:"name,omitempty"`
	DisplayName string `json:"displayName,omitempty"`
}

type status struct {
	Name           string `json:"name"`
	StatusCategory named  `json:"statusCategory"`
}

type transition struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	To   struct {
		Name           string `json:"name"`
		StatusCategory named  `json:"statusCategory"`
	} `json:"to"`
}

type comment struct {
	ID      string `json:"id"`
	Author  named  `json:"author"`
	Created string `json:"created"`
	Body    any    `json:"body"`
}

type userEntry struct {
	AccountID   string `json:"accountId"`
	DisplayName string `json:"displayName"`
}

// Server state loaded from embedded fixtures.
var (
	issues              []issue
	commentsByIssue     map[string][]comment
	transitionsByStatus map[string][]transition
	users               []userEntry
	nextCommentID       = 20000
)

func loadFixtures() {
	issues = mustLoad[struct {
		Issues []issue `json:"issues"`
	}]("testdata/issues.json").Issues

	commentsByIssue = mustLoad[map[string][]comment]("testdata/comments.json")
	transitionsByStatus = mustLoad[map[string][]transition]("testdata/transitions.json")
	users = mustLoad[[]userEntry]("testdata/users.json")
}

func mustLoad[T any](path string) T {
	data, err := fixtures.ReadFile(path)
	if err != nil {
		panic(fmt.Sprintf("load %s: %v", path, err))
	}
	var v T
	if err := json.Unmarshal(data, &v); err != nil {
		panic(fmt.Sprintf("parse %s: %v", path, err))
	}
	return v
}

func findIssue(key string) *issue {
	for i := range issues {
		if issues[i].Key == key {
			return &issues[i]
		}
	}
	return nil
}

func main() {
	loadFixtures()

	port := "18932"
	if p := os.Getenv("PORT"); p != "" {
		port = p
	}

	http.HandleFunc("/rest/api/3/search/jql", handleSearch)
	http.HandleFunc("/rest/api/3/issue/", handleIssue)
	http.HandleFunc("/rest/api/3/user/search", handleUserSearch)

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

func handleSearch(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"issues": issues})
}

func handleIssue(w http.ResponseWriter, r *http.Request) {
	const prefix = "/rest/api/3/issue/"
	parts := splitPath(r.URL.Path[len(prefix):])

	if len(parts) != 2 {
		http.NotFound(w, r)
		return
	}

	iss := findIssue(parts[0])
	if iss == nil {
		http.NotFound(w, r)
		return
	}

	switch parts[1] {
	case "comment":
		handleComment(w, r, parts[0])
	case "transitions":
		handleTransitions(w, r, iss)
	default:
		http.NotFound(w, r)
	}
}

func handleComment(w http.ResponseWriter, r *http.Request, issueKey string) {
	switch r.Method {
	case "GET":
		comments := commentsByIssue[issueKey]
		if comments == nil {
			comments = []comment{}
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"total":    len(comments),
			"comments": comments,
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
}

func handleTransitions(w http.ResponseWriter, r *http.Request, iss *issue) {
	switch r.Method {
	case "GET":
		statusCat := iss.Fields.Status.StatusCategory.Name
		transitions := transitionsByStatus[statusCat]
		if transitions == nil {
			transitions = []transition{}
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"transitions": transitions})

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
}

func handleUserSearch(w http.ResponseWriter, r *http.Request) {
	query := strings.ToLower(r.URL.Query().Get("query"))
	var results []userEntry
	for _, u := range users {
		if strings.Contains(strings.ToLower(u.DisplayName), query) {
			results = append(results, u)
		}
	}
	if results == nil {
		results = []userEntry{}
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(results)
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
