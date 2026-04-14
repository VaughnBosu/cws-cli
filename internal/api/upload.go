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
		return nil, fmt.Errorf("failed to parse upload response (HTTP %d): %s", statusCode, truncateBody(respBody, 200))
	}

	if statusCode < 200 || statusCode >= 300 {
		if len(resp.ItemError) > 0 {
			return &resp, NewCWSError("upload", statusCode, resp.ItemError, "")
		}
		parsed := ParseAPIErrorDetail(respBody)
		if parsed != nil {
			return &resp, NewCWSErrorFromParsed("upload", statusCode, parsed, "")
		}
		return &resp, &CWSError{
			Operation:  "upload",
			HTTPStatus: statusCode,
			Message:    string(respBody),
			Hint:       HintForHTTPStatus(statusCode),
		}
	}

	return &resp, nil
}
