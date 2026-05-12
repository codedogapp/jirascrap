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
)

const (
	maxRetries     = 3
	initialBackoff = 500 * time.Millisecond
)

// doRequest executes an authenticated request with retry for transient failures.
// Retries on 429 (rate limit) and 5xx errors with exponential backoff.
func (c *Client) doRequest(ctx context.Context, method, url string, body any, acceptedStatus ...int) ([]byte, error) {
	jsonBytes, err := marshalBody(body)
	if err != nil {
		return nil, err
	}

	if len(acceptedStatus) == 0 {
		acceptedStatus = []int{http.StatusOK}
	}

	var lastErr error
	backoff := initialBackoff

	for attempt := range maxRetries {
		respBody, err := c.executeRequest(ctx, method, url, jsonBytes)
		if err != nil {
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				return nil, fmt.Errorf("request cancelled: %w", err)
			}
			lastErr = fmt.Errorf("network error: %w", err)
			backoff = c.retryOrBreak(ctx, attempt, backoff)
			continue
		}

		action, wait := classifyResponse(respBody, acceptedStatus)
		switch action {
		case responseOK:
			return respBody.body, nil
		case responseFail:
			return nil, fmt.Errorf("jira api error [%d]: %s", respBody.statusCode, string(respBody.body))
		case responseRetry:
			lastErr = fmt.Errorf("jira api error [%d]: %s", respBody.statusCode, string(respBody.body))
			if wait > 0 {
				backoff = wait
			}
			backoff = c.retryOrBreak(ctx, attempt, backoff)
		}
	}

	return nil, lastErr
}

type rawResponse struct {
	statusCode int
	body       []byte
	headers    http.Header
}

// executeRequest builds and sends a single HTTP request, returning the raw response.
func (c *Client) executeRequest(ctx context.Context, method, url string, jsonBytes []byte) (*rawResponse, error) {
	var reqBody io.Reader
	if jsonBytes != nil {
		reqBody = bytes.NewReader(jsonBytes)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")
	if jsonBytes != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.SetBasicAuth(c.email, c.token)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}

	body, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	return &rawResponse{statusCode: resp.StatusCode, body: body, headers: resp.Header}, nil
}

type responseAction int

const (
	responseOK    responseAction = iota
	responseFail                 // non-retryable error
	responseRetry                // retryable (429, 5xx)
)

// classifyResponse decides how to handle a response status code.
// Returns the action and an optional retry wait duration (from Retry-After header).
func classifyResponse(resp *rawResponse, accepted []int) (responseAction, time.Duration) {
	if resp.statusCode == http.StatusTooManyRequests {
		return responseRetry, parseRetryAfter(resp.headers)
	}
	if resp.statusCode >= 500 {
		return responseRetry, 0
	}
	for _, s := range accepted {
		if resp.statusCode == s {
			return responseOK, 0
		}
	}
	return responseFail, 0
}

// parseRetryAfter extracts the Retry-After header value as a duration.
// Returns 0 if absent or unparseable.
func parseRetryAfter(h http.Header) time.Duration {
	if v := h.Get("Retry-After"); v != "" {
		if secs, err := strconv.Atoi(v); err == nil && secs > 0 {
			return time.Duration(secs) * time.Second
		}
	}
	return 0
}

func marshalBody(body any) ([]byte, error) {
	if body == nil {
		return nil, nil
	}
	b, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}
	return b, nil
}

// retryOrBreak sleeps with backoff if retries remain, returns doubled backoff.
func (c *Client) retryOrBreak(ctx context.Context, attempt int, backoff time.Duration) time.Duration {
	if attempt < maxRetries-1 {
		_ = sleepWithContext(ctx, backoff)
	}
	return backoff * 2
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
