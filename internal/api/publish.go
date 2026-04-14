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
		return nil, fmt.Errorf("failed to parse publish response (HTTP %d): %s", statusCode, truncateBody(respBody, 200))
	}

	if statusCode < 200 || statusCode >= 300 {
		if parsed := ParseAPIErrorDetail(respBody); parsed != nil {
			return &resp, NewCWSErrorFromParsed("publish", statusCode, parsed, "")
		}

		cwsErr := &CWSError{
			Operation:  "publish",
			HTTPStatus: statusCode,
			Message:    truncateBody(respBody, 200),
		}
		if resp.StatusCode != "" {
			cwsErr.Code = resp.StatusCode
			cwsErr.Hint = HintForPublishStatus(resp.StatusCode)
		}
		if cwsErr.Hint == "" {
			cwsErr.Hint = ResolveHint(cwsErr.Code, statusCode, cwsErr.Message)
		}
		return &resp, cwsErr
	}

	return &resp, nil
}
