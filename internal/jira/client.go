// Package jira provides an HTTP client for the Jira REST API v3.
//
// API assumptions:
//   - Authentication: Basic auth with email + API token (Atlassian Cloud).
//   - Search: POST /rest/api/3/search/jql — accepts JQL in request body.
//   - Transitions: GET/POST /rest/api/3/issue/{key}/transitions.
//   - Rate limiting: 429 responses include optional Retry-After header (seconds).
//   - Pagination: Not implemented — assumes < maxResults issues per query.
//   - ADF: Description field uses Atlassian Document Format (v1).
//   - Epic children: Found via JQL `"Epic Link" = X OR parent = X`.
package jira

import (
	"context"
	"net/http"
	"time"

	"github.com/codedogapp/jirascrap/internal/config"
	"github.com/codedogapp/jirascrap/internal/model"
)

// Focused interfaces for Jira API domains.
// Consumers should depend on the narrowest interface they need.

// TicketFetcher handles ticket and epic retrieval.
type TicketFetcher interface {
	FetchTickets(ctx context.Context) ([]model.Ticket, error)
	FetchEpicChildren(ctx context.Context, epicKey string) ([]model.Ticket, error)
	FetchAllEpicChildren(ctx context.Context, tickets []model.Ticket) (map[string][]model.Ticket, error)
}

// CommentClient handles comment retrieval and posting.
type CommentClient interface {
	FetchComments(ctx context.Context, issueKey string, maxResults int) ([]model.Comment, int, error)
	PostComment(ctx context.Context, issueKey string, body any) error
}

// UserSearcher handles user lookup for @mentions.
type UserSearcher interface {
	SearchUsers(ctx context.Context, query string) ([]model.User, error)
}

// TransitionClient handles issue status transitions.
type TransitionClient interface {
	FetchTransitions(ctx context.Context, issueKey string) ([]Transition, error)
	DoTransition(ctx context.Context, issueKey string, transitionID string) error
}

// TicketClient composes all Jira API domain interfaces.
// Used by the TUI which needs full access.
type TicketClient interface {
	TicketFetcher
	CommentClient
	UserSearcher
	TransitionClient
}

// Client is the Jira REST API v3 client.
type Client struct {
	domain        string
	email         string
	token         string
	http          *http.Client
	maxResults    int
	maxConcurrent int
}

func NewClient(cfg *config.Config) *Client {
	return &Client{
		domain:        cfg.Domain,
		email:         cfg.Email,
		token:         cfg.APIToken,
		http:          &http.Client{Timeout: 15 * time.Second},
		maxResults:    100,
		maxConcurrent: 5,
	}
}
