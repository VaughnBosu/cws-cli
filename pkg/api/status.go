package api

import (
	"context"
	"encoding/json"
	"fmt"
)

// FetchStatus retrieves the current status of an extension.
// The raw response body is returned alongside the parsed struct for --json output.
func (c *Client) FetchStatus(ctx context.Context, extensionID string) (*StatusResponse, []byte, error) {
	path := c.itemPath(extensionID, "fetchStatus")

	respBody, statusCode, err := c.doJSON(ctx, "GET", path, nil)
	if err != nil {
		return nil, nil, err
	}

	if statusCode == 404 {
		return nil, nil, &CWSError{
			Operation:  "status check",
			HTTPStatus: 404,
			Message:    fmt.Sprintf("extension not found. Verify the extension ID: %s", extensionID),
			Hint:       "Double-check the extension ID in your cws.toml or --extension-id flag.",
		}
	}

	if statusCode < 200 || statusCode >= 300 {
		return nil, respBody, apiError("status check", statusCode, respBody)
	}

	var resp StatusResponse
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, respBody, fmt.Errorf("failed to parse status response (HTTP %d): %s", statusCode, truncateBody(respBody, 200))
	}

	return &resp, respBody, nil
}
