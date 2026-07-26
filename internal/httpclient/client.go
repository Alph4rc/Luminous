// Package httpclient provides a minimal HTTP client for calling backend APIs
// from MCP tools. It supports configurable base URL, optional Bearer token,
// and customisable request timeouts via context.
package httpclient

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Client is a minimal HTTP client configured for a specific backend API.
// It handles Bearer-token injection and truncates error response bodies to
// avoid leaking sensitive data into MCP tool results.
type Client struct {
	baseURL string
	token   string
	client  *http.Client
}

// NewClient creates a Client for the given base URL.
// token may be empty, in which case no Authorization header is sent.
// timeout controls the underlying http.Client.Timeout.
func NewClient(baseURL, token string, timeout time.Duration) *Client {
	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		token:   token,
		client: &http.Client{
			Timeout: timeout,
		},
	}
}

// Get performs a GET request to baseURL+path and returns the response body.
// If the token was set during construction, an Authorization: Bearer header
// is added. The caller MUST provide a context with the desired timeout.
//
// On HTTP errors (status >= 400) the error message includes the status code
// but does NOT include the response body to avoid leaking sensitive data.
func (c *Client) Get(ctx context.Context, path string) ([]byte, error) {
	url := c.baseURL + "/" + strings.TrimLeft(path, "/")

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}

	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request to %s failed: %w", url, err)
	}
	defer resp.Body.Close()

	// Read full body — used for both success and capped error snippets.
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1024))
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("upstream returned HTTP %d", resp.StatusCode)
	}

	return body, nil
}

// BaseURL returns the configured base URL (without trailing slash).
func (c *Client) BaseURL() string {
	return c.baseURL
}

// HasToken returns true if an API token is configured.
func (c *Client) HasToken() bool {
	return c.token != ""
}
