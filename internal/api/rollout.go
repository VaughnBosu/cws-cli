package api

import (
	"context"
	"encoding/json"
	"fmt"
)

// SetDeployPercentage sets the deploy percentage for a published extension.
func (c *Client) SetDeployPercentage(ctx context.Context, extensionID string, percentage int) (*DeployPercentageResponse, error) {
	path := c.itemPath(extensionID, "setPublishedDeployPercentage")

	reqBody := &DeployPercentageRequest{
		DeployPercentage: percentage,
	}

	respBody, statusCode, err := c.doJSON(ctx, "POST", path, reqBody)
	if err != nil {
		return nil, err
	}

	var resp DeployPercentageResponse
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse rollout response (HTTP %d): %s", statusCode, string(respBody))
	}

	if statusCode < 200 || statusCode >= 300 {
		errMsg := fmt.Sprintf("rollout failed (HTTP %d)", statusCode)
		if resp.Status != "" {
			errMsg += ": " + resp.Status
		} else if detail := ParseAPIError(respBody); detail != "" {
			errMsg += ": " + detail
		}
		return &resp, fmt.Errorf("%s", errMsg)
	}

	return &resp, nil
}
