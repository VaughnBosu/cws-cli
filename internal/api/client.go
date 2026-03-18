package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/null3000/cws-cli/internal/auth"
)

const defaultBaseURL = "https://chromewebstore.googleapis.com"

// Client is the Chrome Web Store API V2 client.
type Client struct {
	httpClient  *http.Client
	auth        auth.Authenticator
	publisherID string
	BaseURL     string // override for testing; empty uses default
}

func (c *Client) baseURL() string {
	if c.BaseURL != "" {
		return c.BaseURL
	}
	return defaultBaseURL
}

// NewClient creates a new API client.
func NewClient(authenticator auth.Authenticator, publisherID string) *Client {
	return &Client{
		httpClient:  &http.Client{},
		auth:        authenticator,
		publisherID: publisherID,
	}
}

func (c *Client) doRequest(ctx context.Context, method, path string, body io.Reader, contentType string) ([]byte, int, error) {
	token, err := c.auth.AccessToken(ctx)
	if err != nil {
		return nil, 0, err
	}

	url := c.baseURL() + path
	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+token)
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("request failed: %w. Check your network connection and try again", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, fmt.Errorf("failed to read response: %w", err)
	}

	return respBody, resp.StatusCode, nil
}

func (c *Client) doJSON(ctx context.Context, method, path string, reqBody any) ([]byte, int, error) {
	var body io.Reader
	var contentType string
	if reqBody != nil {
		data, err := json.Marshal(reqBody)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to marshal request body: %w", err)
		}
		body = bytes.NewReader(data)
		contentType = "application/json"
	}
	return c.doRequest(ctx, method, path, body, contentType)
}

// ParseAPIError attempts to extract a human-readable error from a Google API error response.
// Returns an empty string if the body is not a recognized error format.
func ParseAPIError(body []byte) string {
	var apiErr APIError
	if err := json.Unmarshal(body, &apiErr); err != nil || apiErr.Error == nil {
		return ""
	}

	msg := apiErr.Error.Message
	for _, d := range apiErr.Error.Details {
		for _, v := range d.FieldViolations {
			if v.Description != "" {
				msg = v.Description
			}
		}
	}
	return msg
}

func (c *Client) itemPath(extensionID, action string) string {
	return fmt.Sprintf("/v2/publishers/%s/items/%s:%s", c.publisherID, extensionID, action)
}

func (c *Client) uploadPath(extensionID string) string {
	return fmt.Sprintf("/upload/v2/publishers/%s/items/%s:upload", c.publisherID, extensionID)
}
