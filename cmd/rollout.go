package cmd

import (
	"context"
	"fmt"
	"strconv"

	"github.com/spf13/cobra"
	"github.com/vaughnbosu/cws-cli/internal/api"
	"github.com/vaughnbosu/cws-cli/internal/output"
)

var rolloutCmd = &cobra.Command{
	Use:   "rollout <percentage>",
	Short: "Set the deploy percentage for a published extension",
	Long: `Set the deploy percentage for a published extension.

Requires 10,000+ seven-day active users. Percentage can only increase, not decrease.`,
	Args: cobra.ExactArgs(1),
	RunE: runRollout,
}

func init() {
	rootCmd.AddCommand(rolloutCmd)
}

func runRollout(cmd *cobra.Command, args []string) error {
	percentage, err := strconv.Atoi(args[0])
	if err != nil || percentage < 1 || percentage > 100 {
		return fmt.Errorf("percentage must be a number between 1 and 100")
	}

	actx, err := newAPIContext(cmd)
	if err != nil {
		return err
	}
	ctx := context.Background()

	output.Info("Setting deploy percentage to %d%% for extension %s...", percentage, actx.extensionID)

	if err := actx.client.SetDeployPercentage(ctx, actx.extensionID, percentage); err != nil {
		return err
	}

	// Confirm against a fresh status read. The value may lag briefly after the
	// write, so only claim the live value when it matches the request.
	confirmed := -1
	if status, _, err := actx.client.FetchStatus(ctx, actx.extensionID); err == nil {
		confirmed = publishedDeployPercentage(status)
	}

	if confirmed == percentage {
		output.Info("Deploy percentage set to %d%%.", percentage)
	} else {
		output.Info("Deploy percentage update accepted. Run 'cws status' to confirm the live value.")
	}

	if output.JSONMode() {
		return output.EmitJSON(map[string]any{
			"requested": percentage,
			"confirmed": confirmed == percentage,
		})
	}
	return nil
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
