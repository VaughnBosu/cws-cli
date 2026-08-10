package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/vaughnbosu/cws-cli/internal/output"
	"github.com/vaughnbosu/cws-cli/pkg/api"
	"github.com/vaughnbosu/cws-cli/pkg/service"
)

var publishCmd = &cobra.Command{
	Use:   "publish",
	Short: "Publish the most recently uploaded version",
	Long: `Publish the most recently uploaded version of an extension.

Use --staged to submit for review without auto-publishing after approval.
Use --deploy-percentage to start a partial rollout on publish (requires
10,000+ seven-day active users).`,
	RunE: runPublish,
}

func init() {
	publishCmd.Flags().Bool("staged", false, "Use STAGED_PUBLISH: submit for review but don't auto-publish")
	publishCmd.Flags().Bool("skip-review", false, "Attempt to skip item review (only some changes are eligible)")
	publishCmd.Flags().Bool("block-on-warnings", false, "Fail the publish if any validation warnings are found")
	publishCmd.Flags().Int("deploy-percentage", 0, "Initial rollout percentage (1-99; omit for full rollout)")
	rootCmd.AddCommand(publishCmd)
}

func runPublish(cmd *cobra.Command, args []string) error {
	staged, _ := cmd.Flags().GetBool("staged")
	skipReview, _ := cmd.Flags().GetBool("skip-review")
	blockOnWarnings, _ := cmd.Flags().GetBool("block-on-warnings")
	deployPercentage, _ := cmd.Flags().GetInt("deploy-percentage")
	if cmd.Flags().Changed("deploy-percentage") && (deployPercentage < 1 || deployPercentage > 99) {
		return fmt.Errorf("--deploy-percentage must be between 1 and 99 (omit the flag for a full rollout)")
	}

	actx, err := newAPIContext(cmd)
	if err != nil {
		return err
	}

	if staged {
		output.Info("Submitting extension %s for staged publish...", actx.ExtensionID)
	} else {
		output.Info("Publishing extension %s...", actx.ExtensionID)
	}

	resp, err := service.Publish(context.Background(), actx, service.PublishOptions{
		Staged:           staged,
		SkipReview:       skipReview,
		BlockOnWarnings:  blockOnWarnings,
		DeployPercentage: deployPercentage,
	})
	if err != nil {
		return err
	}

	printPublishWarnings(resp)

	if resp.State != "" {
		output.Info("State: %s", service.FormatState(resp.State))
	} else {
		output.Info("Publish submitted successfully.")
	}

	if output.JSONMode() {
		return output.EmitJSON(resp)
	}
	return nil
}

func printPublishWarnings(resp *api.PublishResponse) {
	if resp == nil || resp.WarningInfo == nil {
		return
	}
	printWarnings(resp.WarningInfo.Warnings)
}

func printWarnings(warnings []api.Warning) {
	for _, w := range warnings {
		if w.Reason != "" {
			output.Warn("[%s] %s", w.Reason, w.Description)
		} else {
			output.Warn("%s", w.Description)
		}
	}
}
