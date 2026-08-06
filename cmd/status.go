package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/vaughnbosu/cws-cli/internal/output"
	"github.com/vaughnbosu/cws-cli/pkg/api"
	"github.com/vaughnbosu/cws-cli/pkg/service"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Check the current status of an extension",
	RunE:  runStatus,
}

func init() {
	rootCmd.AddCommand(statusCmd)
}

func runStatus(cmd *cobra.Command, args []string) error {
	actx, err := newAPIContext(cmd)
	if err != nil {
		return err
	}

	resp, rawJSON, err := service.GetStatus(context.Background(), actx)
	if err != nil {
		return err
	}

	if output.JSONMode() {
		fmt.Println(string(rawJSON))
		return nil
	}

	output.Info("Extension: %s", actx.ExtensionID)

	if resp.TakenDown {
		output.Warn("This extension has been TAKEN DOWN for a policy violation. Check the developer dashboard: https://chrome.google.com/webstore/devconsole")
	}
	if resp.Warned {
		output.Warn("This extension has a policy WARNING and may be taken down if not resolved. Check the developer dashboard: https://chrome.google.com/webstore/devconsole")
	}

	printRevision("Published", resp.PublishedItemRevisionStatus)
	printRevision("Submitted", resp.SubmittedItemRevisionStatus)

	if resp.LastAsyncUploadState != "" {
		output.Info("")
		output.Info("Upload:    %s", resp.LastAsyncUploadState)
		if api.IsUploadFailed(resp.LastAsyncUploadState) {
			output.Hint("The last upload failed. The v2 API does not return failure details; check the developer dashboard.")
		}
	}

	return nil
}

func printRevision(label string, rev *api.ItemRevisionStatus) {
	if rev == nil {
		return
	}
	output.Info("")
	output.Info("%s:", label)
	output.Info("  State:   %s", service.FormatState(rev.State))
	if rev.CrxVersion != "" {
		output.Info("  Version: %s", rev.CrxVersion)
	}
	for _, ch := range rev.DistributionChannels {
		if ch.CrxVersion != "" && ch.CrxVersion != rev.CrxVersion {
			output.Info("  Version: %s", ch.CrxVersion)
		}
		output.Info("  Deploy:  %d%%", ch.DeployPercentage)
	}
}
