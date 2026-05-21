package jira

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/codedogapp/jirascrap/internal/model"
)

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
