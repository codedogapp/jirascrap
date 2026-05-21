package jira

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

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
