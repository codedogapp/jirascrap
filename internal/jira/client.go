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
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/codedogapp/jirascrap/internal/config"
	"github.com/codedogapp/jirascrap/internal/model"
)

const defaultJQL = `assignee = currentUser() AND statusCategory != Done AND status != 'TO DESCRIBE' ORDER BY status DESC`

const (
	maxRetries     = 3
	initialBackoff = 500 * time.Millisecond
)

// TicketClient defines the operations used by the TUI to interact with Jira.
type TicketClient interface {
	FetchTickets(ctx context.Context) ([]model.Ticket, error)
	FetchEpicChildren(ctx context.Context, epicKey string) ([]model.Ticket, error)
	FetchAllEpicChildren(ctx context.Context, tickets []model.Ticket) (map[string][]model.Ticket, error)
	FetchTransitions(ctx context.Context, issueKey string) ([]Transition, error)
	DoTransition(ctx context.Context, issueKey string, transitionID string) error
}

// ClientOption configures a Client.
type ClientOption func(*Client)

// WithTimeout sets the HTTP client timeout (default: 15s).
func WithTimeout(d time.Duration) ClientOption {
	return func(c *Client) { c.http.Timeout = d }
}

// WithMaxResults sets the max results per Jira search (default: 100).
func WithMaxResults(n int) ClientOption {
	return func(c *Client) { c.maxResults = n }
}

// WithMaxConcurrent sets max concurrent epic fetches (default: 5).
func WithMaxConcurrent(n int) ClientOption {
	return func(c *Client) { c.maxConcurrent = n }
}

type Client struct {
	domain        string
	email         string
	token         string
	http          *http.Client
	maxResults    int
	maxConcurrent int
}

func NewClient(cfg *config.Config, opts ...ClientOption) *Client {
	c := &Client{
		domain:        cfg.Domain,
		email:         cfg.Email,
		token:         cfg.APIToken,
		http:          &http.Client{Timeout: 15 * time.Second},
		maxResults:    100,
		maxConcurrent: 5,
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// doRequest executes an authenticated request with retry for transient failures.
// Retries on 429 (rate limit) and 5xx errors with exponential backoff.
func (c *Client) doRequest(ctx context.Context, method, url string, body any, acceptedStatus ...int) ([]byte, error) {
	var jsonBytes []byte
	if body != nil {
		var err error
		jsonBytes, err = json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request: %w", err)
		}
	}

	if len(acceptedStatus) == 0 {
		acceptedStatus = []int{http.StatusOK}
	}

	var lastErr error
	backoff := initialBackoff

	for attempt := range maxRetries {
		var reqBody io.Reader
		if jsonBytes != nil {
			reqBody = bytes.NewReader(jsonBytes)
		}

		req, err := http.NewRequestWithContext(ctx, method, url, reqBody)
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %w", err)
		}

		req.Header.Set("Accept", "application/json")
		if body != nil {
			req.Header.Set("Content-Type", "application/json")
		}
		req.SetBasicAuth(c.email, c.token)

		resp, err := c.http.Do(req)
		if err != nil {
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				return nil, fmt.Errorf("request cancelled: %w", err)
			}
			lastErr = fmt.Errorf("network error: %w", err)
			if attempt < maxRetries-1 {
				if err := sleepWithContext(ctx, backoff); err != nil {
					return nil, lastErr
				}
				backoff *= 2
			}
			continue
		}

		respBody, readErr := io.ReadAll(resp.Body)
		resp.Body.Close()

		if resp.StatusCode == http.StatusTooManyRequests {
			lastErr = fmt.Errorf("jira api rate limited [429]")
			if attempt < maxRetries-1 {
				wait := retryAfterDuration(resp, backoff)
				if err := sleepWithContext(ctx, wait); err != nil {
					return nil, lastErr
				}
				backoff *= 2
			}
			continue
		}

		if resp.StatusCode >= 500 {
			lastErr = fmt.Errorf("jira api error [%d]: %s", resp.StatusCode, string(respBody))
			if attempt < maxRetries-1 {
				if err := sleepWithContext(ctx, backoff); err != nil {
					return nil, lastErr
				}
				backoff *= 2
			}
			continue
		}

		accepted := false
		for _, s := range acceptedStatus {
			if resp.StatusCode == s {
				accepted = true
				break
			}
		}
		if !accepted {
			return nil, fmt.Errorf("jira api error [%d]: %s", resp.StatusCode, string(respBody))
		}

		if readErr != nil {
			return nil, fmt.Errorf("failed to read response: %w", readErr)
		}
		return respBody, nil
	}

	return nil, lastErr
}

// retryAfterDuration parses the Retry-After header, falling back to the given default.
func retryAfterDuration(resp *http.Response, fallback time.Duration) time.Duration {
	if v := resp.Header.Get("Retry-After"); v != "" {
		if secs, err := strconv.Atoi(v); err == nil && secs > 0 {
			return time.Duration(secs) * time.Second
		}
	}
	return fallback
}

func sleepWithContext(ctx context.Context, d time.Duration) error {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
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
		transitions = append(transitions, Transition{
			ID:               t.ID,
			Name:             t.Name,
			ToStatus:         t.To.Name,
			ToStatusCategory: t.To.StatusCategory.Name,
		})
	}

	return transitions, nil
}

func (c *Client) DoTransition(ctx context.Context, issueKey string, transitionID string) error {
	url := fmt.Sprintf("%s/rest/api/3/issue/%s/transitions", c.domain, issueKey)

	_, err := c.doRequest(ctx, "POST", url,
		transitionRequest{Transition: transitionRef{ID: transitionID}},
		http.StatusOK, http.StatusNoContent,
	)
	return err
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

	for _, t := range epics {
		sem <- struct{}{}
		go func(epicKey string) {
			defer func() { <-sem }()
			children, err := c.FetchEpicChildren(ctx, epicKey)
			ch <- result{epicKey: epicKey, children: children, err: err}
		}(t.ID)
	}

	out := make(map[string][]model.Ticket, len(epics))
	var errs []error
	for range epics {
		r := <-ch
		if r.err != nil {
			errs = append(errs, fmt.Errorf("epic %s: %w", r.epicKey, r.err))
			continue
		}
		out[r.epicKey] = r.children
	}

	return out, errors.Join(errs...)
}
