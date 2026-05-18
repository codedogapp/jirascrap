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
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	neturl "net/url"
	"time"

	"github.com/codedogapp/jirascrap/internal/config"
	"github.com/codedogapp/jirascrap/internal/model"
)

const defaultJQL = `assignee = currentUser() AND statusCategory != Done AND status != 'TO DESCRIBE' ORDER BY status DESC`

// TicketClient defines the operations used by the TUI to interact with Jira.
type TicketClient interface {
	FetchTickets(ctx context.Context) ([]model.Ticket, error)
	FetchEpicChildren(ctx context.Context, epicKey string) ([]model.Ticket, error)
	FetchAllEpicChildren(ctx context.Context, tickets []model.Ticket) (map[string][]model.Ticket, error)
	FetchTransitions(ctx context.Context, issueKey string) ([]Transition, error)
	DoTransition(ctx context.Context, issueKey string, transitionID string) error
	FetchComments(ctx context.Context, issueKey string, maxResults int) ([]model.Comment, int, error)
	PostComment(ctx context.Context, issueKey string, body any) error
	SearchUsers(ctx context.Context, query string) ([]model.User, error)
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

func (c *Client) FetchTickets(ctx context.Context) ([]model.Ticket, error) {
	return c.fetchTickets(ctx, defaultJQL)
}

func (c *Client) FetchEpicChildren(ctx context.Context, epicKey string) ([]model.Ticket, error) {
	jql := fmt.Sprintf(`"Epic Link" = %s OR parent = %s`, epicKey, epicKey)
	return c.fetchTickets(ctx, jql)
}

func (c *Client) fetchTickets(ctx context.Context, jql string) ([]model.Ticket, error) {
	url := fmt.Sprintf("%s/rest/api/3/search/jql", c.domain)
	reqBody := searchRequest{
		MaxResults: c.maxResults,
		JQL:        jql,
		Fields: []string{
			"summary",
			"status",
			"statusCategory",
			"priority",
			"reporter",
			"created",
			"updated",
			"description",
			"issuetype",
		},
	}

	respBody, err := c.doRequest(ctx, "POST", url, reqBody)
	if err != nil {
		return nil, err
	}

	var jiraResp searchResponse
	if err := json.Unmarshal(respBody, &jiraResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	var tickets []model.Ticket
	for _, issue := range jiraResp.Issues {
		tickets = append(tickets, model.Ticket{
			ID:             issue.Key,
			Summary:        issue.Fields.Summary,
			Reporter:       issue.Fields.Reporter.DisplayName,
			Status:         issue.Fields.Status.Name,
			StatusCategory: issue.Fields.Status.StatusCategory.Name,
			Priority:       issue.Fields.Priority.Name,
			Type:           issue.Fields.IssueType.Name,
			CreatedAt:      issue.Fields.CreatedAt.Time,
			UpdatedAt:      issue.Fields.UpdatedAt.Time,
			Markdown:       ADFToMarkdown(issue.Fields.Description),
		})
	}

	return tickets, nil
}

func (c *Client) FetchTransitions(ctx context.Context, issueKey string) ([]Transition, error) {
	url := fmt.Sprintf("%s/rest/api/3/issue/%s/transitions", c.domain, issueKey)

	respBody, err := c.doRequest(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	var result transitionsResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	var transitions []Transition
	for _, t := range result.Transitions {
		transitions = append(
			transitions,
			Transition{
				ID:               t.ID,
				Name:             t.Name,
				ToStatus:         t.To.Name,
				ToStatusCategory: t.To.StatusCategory.Name,
			},
		)
	}

	return transitions, nil
}

func (c *Client) DoTransition(ctx context.Context, issueKey string, transitionID string) error {
	url := fmt.Sprintf("%s/rest/api/3/issue/%s/transitions", c.domain, issueKey)

	_, err := c.doRequest(
		ctx,
		"POST",
		url,
		transitionRequest{Transition: transitionRef{ID: transitionID}},
		http.StatusOK,
		http.StatusNoContent,
	)
	return err
}

func (c *Client) FetchComments(ctx context.Context, issueKey string, maxResults int) ([]model.Comment, int, error) {
	url := fmt.Sprintf("%s/rest/api/3/issue/%s/comment?orderBy=-created&maxResults=%d", c.domain, issueKey, maxResults)

	respBody, err := c.doRequest(ctx, "GET", url, nil)
	if err != nil {
		return nil, 0, err
	}

	var result commentsResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, 0, fmt.Errorf("failed to decode comments: %w", err)
	}

	comments := make([]model.Comment, 0, len(result.Comments))
	for comment := range result.Comments {
		entry := result.Comments[comment]
		comments = append(comments, model.Comment{
			ID:        entry.ID,
			Author:    entry.Author.DisplayName,
			CreatedAt: entry.Created.Time,
			Markdown:  ADFToMarkdown(entry.Body),
		})
	}

	return comments, result.Total, nil
}

func (c *Client) PostComment(ctx context.Context, issueKey string, body any) error {
	url := fmt.Sprintf("%s/rest/api/3/issue/%s/comment", c.domain, issueKey)
	_, err := c.doRequest(ctx, "POST", url, commentPostRequest{Body: body}, http.StatusCreated)
	return err
}

func (c *Client) SearchUsers(ctx context.Context, query string) ([]model.User, error) {
	url := fmt.Sprintf("%s/rest/api/3/user/search?query=%s&maxResults=5", c.domain, neturl.QueryEscape(query))

	respBody, err := c.doRequest(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	var entries []userSearchEntry
	if err := json.Unmarshal(respBody, &entries); err != nil {
		return nil, fmt.Errorf("failed to decode user search: %w", err)
	}

	users := make([]model.User, 0, len(entries))
	for _, e := range entries {
		users = append(users, model.User{
			AccountID:   e.AccountID,
			DisplayName: e.DisplayName,
		})
	}

	return users, nil
}

func (c *Client) FetchAllEpicChildren(ctx context.Context, tickets []model.Ticket) (map[string][]model.Ticket, error) {
	type result struct {
		epicKey  string
		children []model.Ticket
		err      error
	}

	var epics []model.Ticket
	for _, t := range tickets {
		if t.IsEpic() {
			epics = append(epics, t)
		}
	}

	if len(epics) == 0 {
		return map[string][]model.Ticket{}, nil
	}

	sem := make(chan struct{}, c.maxConcurrent)
	ch := make(chan result, len(epics))

	launched := 0
	for _, t := range epics {
		if ctx.Err() != nil {
			break
		}
		sem <- struct{}{}
		launched++
		go func(epicKey string) {
			defer func() {
				<-sem
				if r := recover(); r != nil {
					ch <- result{epicKey: epicKey, err: fmt.Errorf("panic fetching epic %s: %v", epicKey, r)}
				}
			}()
			children, err := c.FetchEpicChildren(ctx, epicKey)
			ch <- result{epicKey: epicKey, children: children, err: err}
		}(t.ID)
	}

	out := make(map[string][]model.Ticket, len(epics))
	var errs []error
	for range launched {
		r := <-ch
		if r.err != nil {
			errs = append(errs, fmt.Errorf("epic %s: %w", r.epicKey, r.err))
			continue
		}
		out[r.epicKey] = r.children
	}

	if ctx.Err() != nil {
		errs = append(errs, ctx.Err())
	}

	return out, errors.Join(errs...)
}
