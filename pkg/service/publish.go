package service

import (
	"context"
	"fmt"

	"github.com/vaughnbosu/cws-cli/pkg/api"
)

// PublishOptions configures a publish operation.
type PublishOptions struct {
	Staged           bool
	SkipReview       bool
	BlockOnWarnings  bool
	DeployPercentage int
}

// Publish submits the most recently uploaded version for publishing.
func Publish(ctx context.Context, actx *Context, opts PublishOptions) (*api.PublishResponse, error) {
	if opts.DeployPercentage != 0 && (opts.DeployPercentage < 1 || opts.DeployPercentage > 99) {
		return nil, fmt.Errorf("deploy_percentage must be between 1 and 99 (omit for a full rollout)")
	}

	return actx.Client.Publish(ctx, actx.ExtensionID, api.PublishOptions{
		Staged:           opts.Staged,
		SkipReview:       opts.SkipReview,
		BlockOnWarnings:  opts.BlockOnWarnings,
		DeployPercentage: opts.DeployPercentage,
	})
}
