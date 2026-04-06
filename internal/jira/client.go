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

type Client struct {
	domain string
	email  string
	token  string
	jql    string
	http   *http.Client
}

func NewClient(cfg *config.Config) *Client {
	return &Client{
		domain: cfg.Domain,
		email:  cfg.Email,
		token:  cfg.APIToken,
		jql:    cfg.JQL,
		http:   &http.Client{Timeout: 15 * time.Second},
	}
}

func (c *Client) FetchTickets() ([]model.Ticket, error) {
	url := fmt.Sprintf("%s/rest/api/3/search/jql", c.domain)
	reqBody := searchRequest{
		JQL: c.jql,
		Fields: []string{
			"summary",
			"status",
			"statusCategory",
			"priority",
			"reporter",
			"created",
			"updated",
			"description",
		},
	}

	reqJSON, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(reqJSON))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth(c.email, c.token)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("network error: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("jira api error [%d]: %s", resp.StatusCode, string(bodyBytes))
	}

	var jiraResp searchResponse
	err = json.NewDecoder(resp.Body).Decode(&jiraResp)
	if err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	var tickets []model.Ticket

	for _, issue := range jiraResp.Issues {
		markdownDescription := ADFToMarkdown(issue.Fields.Description)

		tickets = append(tickets, model.Ticket{
			ID:             issue.Key,
			Summary:        issue.Fields.Summary,
			Reporter:       issue.Fields.Reporter.DisplayName,
			Status:         issue.Fields.Status.Name,
			StatusCategory: issue.Fields.Status.StatusCategory.Name,
			Priority:       issue.Fields.Priority.Name,
			CreatedAt:      issue.Fields.CreatedAt.Time,
			UpdatedAt:      issue.Fields.UpdatedAt.Time,
			Markdown:       markdownDescription,
		})
	}

	return tickets, nil
}
