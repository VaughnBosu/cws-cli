package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/vaughnbosu/cws-cli/pkg/auth"
)

const defaultBaseURL = "https://chromewebstore.googleapis.com"

// jsonCallTimeout bounds metadata calls (publish, status, rollout, cancel).
// Uploads get uploadTimeout since the body transfer can be large and slow.
const (
	jsonCallTimeout = 2 * time.Minute
	uploadTimeout   = 60 * time.Minute
)

const maxRetries = 3

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
// Per-call deadlines are set in doRequest rather than on the http.Client,
// so a large upload body is not bounded by a metadata-call timeout.
func NewClient(authenticator auth.Authenticator, publisherID string) *Client {
	return &Client{
		httpClient:  &http.Client{},
		auth:        authenticator,
		publisherID: publisherID,
	}
}

// retryableStatus reports whether a response status is worth retrying.
func retryableStatus(code int) bool {
	return code == 429 || code == 500 || code == 502 || code == 503 || code == 504
}

// retryDelay returns how long to wait before the given attempt (0-based),
// honoring a Retry-After header when present.
func retryDelay(attempt int, resp *http.Response) time.Duration {
	if resp != nil {
		if ra := resp.Header.Get("Retry-After"); ra != "" {
			if secs, err := strconv.Atoi(ra); err == nil && secs > 0 && secs <= 60 {
				return time.Duration(secs) * time.Second
			}
		}
	}
	return time.Duration(1<<attempt) * time.Second // 1s, 2s, 4s
}

func (c *Client) doRequest(ctx context.Context, method, path string, body []byte, contentType string, timeout time.Duration) ([]byte, int, error) {
	token, err := c.auth.AccessToken(ctx)
	if err != nil {
		return nil, 0, err
	}

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	url := c.baseURL() + path

	var respBody []byte
	var statusCode int
	var lastErr error

	for attempt := 0; attempt < maxRetries; attempt++ {
		var reader io.Reader
		if body != nil {
			reader = bytes.NewReader(body)
		}
		req, err := http.NewRequestWithContext(ctx, method, url, reader)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to create request: %w", err)
		}
		req.Header.Set("Authorization", "Bearer "+token)
		if contentType != "" {
			req.Header.Set("Content-Type", contentType)
		}

		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("request failed: %w. Check your network connection and try again", err)
			if ctx.Err() != nil {
				return nil, 0, lastErr
			}
			select {
			case <-time.After(retryDelay(attempt, nil)):
				continue
			case <-ctx.Done():
				return nil, 0, lastErr
			}
		}

		respBody, err = io.ReadAll(resp.Body)
		resp.Body.Close()
		statusCode = resp.StatusCode
		if err != nil {
			return nil, statusCode, fmt.Errorf("failed to read response: %w", err)
		}

		if retryableStatus(statusCode) && attempt < maxRetries-1 {
			select {
			case <-time.After(retryDelay(attempt, resp)):
				continue
			case <-ctx.Done():
				return respBody, statusCode, nil
			}
		}

		return respBody, statusCode, nil
	}

	return nil, statusCode, lastErr
}

func (c *Client) doJSON(ctx context.Context, method, path string, reqBody any) ([]byte, int, error) {
	var body []byte
	var contentType string
	if reqBody != nil {
		data, err := json.Marshal(reqBody)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to marshal request body: %w", err)
		}
		body = data
		contentType = "application/json"
	}
	return c.doRequest(ctx, method, path, body, contentType, jsonCallTimeout)
}

// apiError builds a CWSError for a non-2xx response, preferring the structured
// Google error body when one is present.
func apiError(operation string, statusCode int, body []byte) *CWSError {
	if parsed := ParseAPIErrorDetail(body); parsed != nil {
		return NewCWSErrorFromParsed(operation, statusCode, parsed, "")
	}
	return NewOperationError(operation, statusCode, truncateBody(body, 200))
}

// ParsedAPIError holds structured information extracted from a Google API error response.
type ParsedAPIError struct {
	Message     string   // top-level error message
	StatusCode  int      // HTTP-like code from the error body
	Status      string   // status string (e.g., "INVALID_ARGUMENT")
	Reasons     []string // reason codes from field violations (e.g., "PKG_INVALID_VERSION_NUMBER")
	Description string   // most specific description found
	Violations  []FieldViolation
}

// ParseAPIErrorDetail extracts structured error information from a Google API error response.
// Returns nil if the body is not a recognized error format.
func ParseAPIErrorDetail(body []byte) *ParsedAPIError {
	var apiErr APIError
	if err := json.Unmarshal(body, &apiErr); err != nil || apiErr.Error == nil {
		return nil
	}

	parsed := &ParsedAPIError{
		Message:    apiErr.Error.Message,
		StatusCode: apiErr.Error.Code,
		Status:     apiErr.Error.Status,
	}

	for _, d := range apiErr.Error.Details {
		for _, v := range d.FieldViolations {
			if v.Reason != "" {
				parsed.Reasons = append(parsed.Reasons, v.Reason)
			}
			parsed.Violations = append(parsed.Violations, v)
			if v.Description != "" {
				if parsed.Description == "" {
					parsed.Description = v.Description
				}
			}
		}
	}

	if parsed.Description == "" {
		parsed.Description = parsed.Message
	}

	return parsed
}

// ParseAPIError attempts to extract a human-readable error from a Google API error response.
// Returns an empty string if the body is not a recognized error format.
func ParseAPIError(body []byte) string {
	parsed := ParseAPIErrorDetail(body)
	if parsed == nil {
		return ""
	}
	return parsed.Description
}

// truncateBody returns the response body as a string, truncated to maxLen characters.
func truncateBody(body []byte, maxLen int) string {
	s := string(body)
	if len(s) > maxLen {
		return s[:maxLen] + "..."
	}
	return s
}

func (c *Client) itemPath(extensionID, action string) string {
	return fmt.Sprintf("/v2/publishers/%s/items/%s:%s", c.publisherID, extensionID, action)
}

func (c *Client) uploadPath(extensionID string) string {
	return fmt.Sprintf("/upload/v2/publishers/%s/items/%s:upload", c.publisherID, extensionID)
}
