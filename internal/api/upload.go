package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
)

// Upload uploads a zip file to the Chrome Web Store.
func (c *Client) Upload(ctx context.Context, extensionID string, zipData []byte) (*UploadResponse, error) {
	path := c.uploadPath(extensionID)
	respBody, statusCode, err := c.doRequest(ctx, "POST", path, bytes.NewReader(zipData), "application/zip")
	if err != nil {
		return nil, err
	}

	var resp UploadResponse
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse upload response (HTTP %d): %s", statusCode, string(respBody))
	}

	if statusCode < 200 || statusCode >= 300 {
		errMsg := fmt.Sprintf("upload failed (HTTP %d)", statusCode)
		if len(resp.ItemError) > 0 {
			errMsg += ": " + resp.ItemError[0].ErrorDetail
		} else if detail := ParseAPIError(respBody); detail != "" {
			errMsg += ": " + detail
		} else {
			errMsg += ": " + string(respBody)
		}
		return &resp, fmt.Errorf("%s", errMsg)
	}

	return &resp, nil
}
