package api

import (
	"context"
	"encoding/json"
	"fmt"
)

// CancelSubmission cancels a pending submission.
func (c *Client) CancelSubmission(ctx context.Context, extensionID string) (*CancelResponse, error) {
	path := c.itemPath(extensionID, "cancelSubmission")

	respBody, statusCode, err := c.doJSON(ctx, "POST", path, nil)
	if err != nil {
		return nil, err
	}

	var resp CancelResponse
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse cancel response (HTTP %d): %s", statusCode, string(respBody))
	}

	if statusCode < 200 || statusCode >= 300 {
		errMsg := fmt.Sprintf("cancel failed (HTTP %d)", statusCode)
		if detail := ParseAPIError(respBody); detail != "" {
			errMsg += ": " + detail
		} else {
			errMsg += ": no pending submission to cancel for this extension"
		}
		return &resp, fmt.Errorf("%s", errMsg)
	}

	return &resp, nil
}
