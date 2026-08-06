package service

import (
	"context"

	"github.com/vaughnbosu/cws-cli/pkg/api"
)

// RolloutResult reports the outcome of a deploy percentage change.
type RolloutResult struct {
	Requested int  `json:"requested"`
	Confirmed bool `json:"confirmed"`
}

// SetRollout sets the deploy percentage for a published extension.
func SetRollout(ctx context.Context, actx *Context, percentage int) (*RolloutResult, error) {
	if err := actx.Client.SetDeployPercentage(ctx, actx.ExtensionID, percentage); err != nil {
		return nil, err
	}

	confirmed := false
	if status, _, err := actx.Client.FetchStatus(ctx, actx.ExtensionID); err == nil {
		confirmed = publishedDeployPercentage(status) == percentage
	}

	return &RolloutResult{
		Requested: percentage,
		Confirmed: confirmed,
	}, nil
}

func publishedDeployPercentage(status *api.StatusResponse) int {
	if status == nil || status.PublishedItemRevisionStatus == nil {
		return -1
	}
	channels := status.PublishedItemRevisionStatus.DistributionChannels
	if len(channels) == 0 {
		return -1
	}
	return channels[0].DeployPercentage
}
