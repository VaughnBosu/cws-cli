package api

import (
	"context"
	"encoding/json"
	"fmt"
)

// Upload uploads a zip file to the Chrome Web Store.
func (c *Client) Upload(ctx context.Context, extensionID string, zipData []byte) (*UploadResponse, error) {
	path := c.uploadPath(extensionID)
	respBody, statusCode, err := c.doRequest(ctx, "POST", path, zipData, "application/zip", uploadTimeout)
	if err != nil {
		return nil, err
	}

	if statusCode < 200 || statusCode >= 300 {
		return nil, apiError("upload", statusCode, respBody)
	}

	var resp UploadResponse
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse upload response (HTTP %d): %s", statusCode, truncateBody(respBody, 200))
	}

	return &resp, nil
}
