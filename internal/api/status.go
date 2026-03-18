package api

import (
	"context"
	"encoding/json"
	"fmt"
)

// FetchStatus retrieves the current status of an extension.
func (c *Client) FetchStatus(ctx context.Context, extensionID string) (*StatusResponse, []byte, error) {
	path := c.itemPath(extensionID, "fetchStatus")

	respBody, statusCode, err := c.doJSON(ctx, "GET", path, nil)
	if err != nil {
		return nil, nil, err
	}

	if statusCode == 404 {
		return nil, nil, fmt.Errorf("extension not found. Verify the extension ID: %s", extensionID)
	}

	var resp StatusResponse
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, nil, fmt.Errorf("failed to parse status response (HTTP %d): %s", statusCode, string(respBody))
	}

	if statusCode < 200 || statusCode >= 300 {
		errMsg := fmt.Sprintf("status check failed (HTTP %d)", statusCode)
		if len(resp.ItemError) > 0 {
			errMsg += ": " + resp.ItemError[0].ErrorDetail
		} else if detail := ParseAPIError(respBody); detail != "" {
			errMsg += ": " + detail
		} else {
			errMsg += ": " + string(respBody)
		}
		return &resp, respBody, fmt.Errorf("%s", errMsg)
	}

	return &resp, respBody, nil
}
