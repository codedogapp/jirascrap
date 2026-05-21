package jira

import (
	"context"
	"encoding/json"
	"fmt"
	neturl "net/url"

	"github.com/codedogapp/jirascrap/internal/model"
)

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
