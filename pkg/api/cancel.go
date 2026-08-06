package api

import (
	"context"
)

// CancelSubmission cancels a pending submission. The v2 response body is empty,
// so success is indicated by the error being nil.
func (c *Client) CancelSubmission(ctx context.Context, extensionID string) error {
	path := c.itemPath(extensionID, "cancelSubmission")

	respBody, statusCode, err := c.doJSON(ctx, "POST", path, nil)
	if err != nil {
		return err
	}

	if statusCode < 200 || statusCode >= 300 {
		return apiError("cancel", statusCode, respBody)
	}

	return nil
}
