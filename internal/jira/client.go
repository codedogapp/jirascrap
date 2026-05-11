package jira

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/codedogapp/jirascrap/internal/config"
	"github.com/codedogapp/jirascrap/internal/model"
)

const defaultJQL = `assignee = currentUser() AND statusCategory != Done AND status != 'TO DESCRIBE' ORDER BY status DESC`

type Client struct {
	domain string
	email  string
	token  string
	http   *http.Client
}

func NewClient(cfg *config.Config) *Client {
	return &Client{
		domain: cfg.Domain,
		email:  cfg.Email,
		token:  cfg.APIToken,
		http:   &http.Client{Timeout: 15 * time.Second},
	}
}

// doRequest executes an authenticated request and returns the response body.
// It checks that the status code is one of the accepted codes.
func (c *Client) doRequest(method, url string, body any, acceptedStatus ...int) ([]byte, error) {
	var reqBody io.Reader
	if body != nil {
		jsonBytes, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request: %w", err)
		}
		reqBody = bytes.NewBuffer(jsonBytes)
	}

	req, err := http.NewRequest(method, url, reqBody)
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
		return nil, fmt.Errorf("network error: %w", err)
	}
	defer resp.Body.Close()

	if len(acceptedStatus) == 0 {
		acceptedStatus = []int{http.StatusOK}
	}
	accepted := false
	for _, s := range acceptedStatus {
		if resp.StatusCode == s {
			accepted = true
			break
		}
	}
	if !accepted {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("jira api error [%d]: %s", resp.StatusCode, string(respBody))
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}
	return respBody, nil
}

func (c *Client) FetchTickets() ([]model.Ticket, error) {
	return c.fetchTickets(defaultJQL)
}

func (c *Client) FetchEpicChildren(epicKey string) ([]model.Ticket, error) {
	jql := fmt.Sprintf(`"Epic Link" = %s OR parent = %s`, epicKey, epicKey)
	return c.fetchTickets(jql)
}

func (c *Client) fetchTickets(jql string) ([]model.Ticket, error) {
	url := fmt.Sprintf("%s/rest/api/3/search/jql", c.domain)
	reqBody := searchRequest{
		MaxResults: 100,
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

	respBody, err := c.doRequest("POST", url, reqBody)
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

func (c *Client) FetchTransitions(issueKey string) ([]Transition, error) {
	url := fmt.Sprintf("%s/rest/api/3/issue/%s/transitions", c.domain, issueKey)

	respBody, err := c.doRequest("GET", url, nil)
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

func (c *Client) DoTransition(issueKey string, transitionID string) error {
	url := fmt.Sprintf("%s/rest/api/3/issue/%s/transitions", c.domain, issueKey)

	_, err := c.doRequest("POST", url,
		transitionRequest{Transition: transitionRef{ID: transitionID}},
		http.StatusOK, http.StatusNoContent,
	)
	return err
}

func (c *Client) FetchAllEpicChildren(tickets []model.Ticket) (map[string][]model.Ticket, error) {
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

	const maxConcurrent = 5
	sem := make(chan struct{}, maxConcurrent)
	ch := make(chan result, len(epics))

	for _, t := range epics {
		sem <- struct{}{}
		go func(epicKey string) {
			defer func() { <-sem }()
			children, err := c.FetchEpicChildren(epicKey)
			ch <- result{epicKey: epicKey, children: children, err: err}
		}(t.ID)
	}

	out := make(map[string][]model.Ticket, len(epics))
	var firstErr error
	for range epics {
		r := <-ch
		if r.err != nil {
			if firstErr == nil {
				firstErr = r.err
			}
			continue
		}
		out[r.epicKey] = r.children
	}

	return out, firstErr
}
