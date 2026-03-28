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

	md "github.com/JohannesKaufmann/html-to-markdown/v2"
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

type searchRequest struct {
	JQL        string   `json:"jql"`
	MaxResults int      `json:"maxResults,omitempty"`
	Expand     string   `json:"expand,omitempty"`
	Fields     []string `json:"fields,omitempty"`
}

type searchResponse struct {
	Issues []struct {
		Key    string `json:"key"`
		Fields struct {
			Summary string `json:"summary"`
			// TODO: other fields + json unmarshaller for time
			CreatedAt time.Time `json:"created"`
			UpdatedAt time.Time `json:"updated"`
		} `json:"fields"`
		RenderedFields struct {
			Description string `json:"description"`
		} `json:"renderedFields"`
	}
}

func (c *Client) FetchTickets() ([]model.Ticket, error) {
	url := fmt.Sprintf("%s/rest/api/3/search/jql", c.domain)
	reqBody := searchRequest{
		// TODO: Remove the limit once the TUI is ready
		MaxResults: 5,
		JQL:        c.jql,
		Expand:     "renderedFields",
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
		var markdownDescription string
		htmlDescription := issue.RenderedFields.Description
		if htmlDescription != "" {
			markdownDescription, err = md.ConvertString(htmlDescription)
			if err != nil {
				markdownDescription = "_Error converting description_"
			}
		} else {
			markdownDescription = "_No description needed_"
		}

		tickets = append(tickets, model.Ticket{
			ID:       issue.Key,
			Markdown: markdownDescription,
		})
	}

	return tickets, nil
}
