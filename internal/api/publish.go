package api

import (
	"context"
	"encoding/json"
	"fmt"
)

// Publish publishes the most recently uploaded version.
func (c *Client) Publish(ctx context.Context, extensionID string, staged bool) (*PublishResponse, error) {
	path := c.itemPath(extensionID, "publish")

	reqBody := &PublishRequest{}
	if staged {
		reqBody.PublishType = "STAGED_PUBLISH"
	}

	respBody, statusCode, err := c.doJSON(ctx, "POST", path, reqBody)
	if err != nil {
		return nil, err
	}

	var resp PublishResponse
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse publish response (HTTP %d): %s", statusCode, string(respBody))
	}

	if statusCode < 200 || statusCode >= 300 {
		errMsg := fmt.Sprintf("publish failed (HTTP %d)", statusCode)
		if resp.StatusCode != "" {
			errMsg += ": " + resp.StatusCode
		} else if detail := ParseAPIError(respBody); detail != "" {
			errMsg += ": " + detail
		}
		if len(resp.Status) > 0 {
			errMsg += " — " + resp.Status[0]
		}
		return &resp, fmt.Errorf("%s", errMsg)
	}

	return &resp, nil
}
