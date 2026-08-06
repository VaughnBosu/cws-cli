package service

import "context"

// CancelResult confirms a submission was cancelled.
type CancelResult struct {
	Cancelled bool `json:"cancelled"`
}

// CancelSubmission cancels a pending submission under review.
func CancelSubmission(ctx context.Context, actx *Context) (*CancelResult, error) {
	if err := actx.Client.CancelSubmission(ctx, actx.ExtensionID); err != nil {
		return nil, err
	}
	return &CancelResult{Cancelled: true}, nil
}
