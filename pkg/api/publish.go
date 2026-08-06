package api

import (
	"context"
	"encoding/json"
	"fmt"
)

// PublishOptions controls publish behavior.
type PublishOptions struct {
	Staged           bool // submit for review without auto-publishing after approval
	SkipReview       bool // attempt to skip item review
	BlockOnWarnings  bool // fail the request if there are warnings
	DeployPercentage int  // initial rollout percentage (0 = full rollout)
}

// Publish publishes the most recently uploaded version.
func (c *Client) Publish(ctx context.Context, extensionID string, opts PublishOptions) (*PublishResponse, error) {
	path := c.itemPath(extensionID, "publish")

	reqBody := &PublishRequest{
		SkipReview:      opts.SkipReview,
		BlockOnWarnings: opts.BlockOnWarnings,
	}
	if opts.Staged {
		reqBody.PublishType = PublishTypeStaged
	}
	if opts.DeployPercentage > 0 && opts.DeployPercentage < 100 {
		reqBody.DeployInfos = []DeployInfo{{DeployPercentage: opts.DeployPercentage}}
	}

	respBody, statusCode, err := c.doJSON(ctx, "POST", path, reqBody)
	if err != nil {
		return nil, err
	}

	if statusCode < 200 || statusCode >= 300 {
		return nil, apiError("publish", statusCode, respBody)
	}

	var resp PublishResponse
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse publish response (HTTP %d): %s", statusCode, truncateBody(respBody, 200))
	}

	return &resp, nil
}
