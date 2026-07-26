// Package httpclient provides a minimal HTTP client for calling backend APIs
// from MCP tools. It supports configurable base URL, optional Bearer token,
// and customisable request timeouts via context.
package httpclient

import (
	"bytes"
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

// doRequest handles the common HTTP request logic: URL construction, token
// injection, execution, and body reading.
func (c *Client) doRequest(ctx context.Context, method, path string, body io.Reader) ([]byte, error) {
	url := c.baseURL + "/" + strings.TrimLeft(path, "/")

	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}

	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request to %s failed: %w", url, err)
	}
	defer resp.Body.Close()

	// Read full body for success responses (1MB cap for safety).
	// Error responses are discarded — the status code alone is returned to
	// avoid leaking sensitive data into error messages.
	if resp.StatusCode >= 400 {
		_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 1024))
		return nil, fmt.Errorf("upstream returned HTTP %d", resp.StatusCode)
	}

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	return respBody, nil
}

// Get performs a GET request to baseURL+path and returns the response body.
func (c *Client) Get(ctx context.Context, path string) ([]byte, error) {
	return c.doRequest(ctx, http.MethodGet, path, nil)
}

// Post performs a POST request to baseURL+path with the given JSON body.
func (c *Client) Post(ctx context.Context, path string, jsonBody []byte) ([]byte, error) {
	return c.doRequest(ctx, http.MethodPost, path, bytes.NewReader(jsonBody))
}

// Put performs a PUT request to baseURL+path with the given JSON body.
func (c *Client) Put(ctx context.Context, path string, jsonBody []byte) ([]byte, error) {
	return c.doRequest(ctx, http.MethodPut, path, bytes.NewReader(jsonBody))
}

// Delete performs a DELETE request to baseURL+path.
func (c *Client) Delete(ctx context.Context, path string) ([]byte, error) {
	return c.doRequest(ctx, http.MethodDelete, path, nil)
}

// BaseURL returns the configured base URL (without trailing slash).
func (c *Client) BaseURL() string {
	return c.baseURL
}

// HasToken returns true if an API token is configured.
func (c *Client) HasToken() bool {
	return c.token != ""
}
