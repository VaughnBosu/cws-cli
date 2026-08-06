package api

import (
	"context"
	"strings"
)

// SetDeployPercentage sets the deploy percentage for a published extension.
// The v2 response body is empty, so success is indicated by the error being nil.
func (c *Client) SetDeployPercentage(ctx context.Context, extensionID string, percentage int) error {
	path := c.itemPath(extensionID, "setPublishedDeployPercentage")

	reqBody := &DeployPercentageRequest{
		DeployPercentage: percentage,
	}

	respBody, statusCode, err := c.doJSON(ctx, "POST", path, reqBody)
	if err != nil {
		return err
	}

	if statusCode < 200 || statusCode >= 300 {
		cwsErr := apiError("rollout", statusCode, respBody)

		// Add rollout-specific hint for the common "does not meet requirements" error
		lower := strings.ToLower(cwsErr.Message)
		if strings.Contains(lower, "does not meet requirements") || strings.Contains(lower, "not eligible") {
			cwsErr.Hint = "Partial rollouts require your extension to have at least 10,000 weekly active users."
		}
		return cwsErr
	}

	return nil
}
