package cmd

import (
	"context"

	"github.com/spf13/cobra"
	"github.com/vaughnbosu/cws-cli/internal/output"
)

var cancelCmd = &cobra.Command{
	Use:   "cancel",
	Short: "Cancel a pending submission",
	Long:  "Cancel a pending submission that is currently under review.",
	RunE:  runCancel,
}

func init() {
	rootCmd.AddCommand(cancelCmd)
}

func runCancel(cmd *cobra.Command, args []string) error {
	actx, err := newAPIContext(cmd)
	if err != nil {
		return err
	}
	ctx := context.Background()

	output.Info("Cancelling submission for extension %s...", actx.extensionID)

	if err := actx.client.CancelSubmission(ctx, actx.extensionID); err != nil {
		return err
	}

	output.Info("Submission cancelled successfully.")
	if output.JSONMode() {
		return output.EmitJSON(map[string]any{"cancelled": true})
	}
	return nil
}
