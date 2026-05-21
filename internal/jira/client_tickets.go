package jira

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/codedogapp/jirascrap/internal/model"
)

const defaultJQL = `assignee = currentUser() AND statusCategory != Done AND status != 'TO DESCRIBE' ORDER BY status DESC`

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
