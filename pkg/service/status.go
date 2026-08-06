package service

import (
	"context"
	"encoding/json"

	"github.com/vaughnbosu/cws-cli/pkg/api"
)

// GetStatus fetches the current extension status and returns both the parsed
// response and the raw API JSON.
func GetStatus(ctx context.Context, actx *Context) (*api.StatusResponse, json.RawMessage, error) {
	return actx.Client.FetchStatus(ctx, actx.ExtensionID)
}
