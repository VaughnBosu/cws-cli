package cmd

import (
	"context"

	"github.com/spf13/cobra"
	"github.com/vaughnbosu/cws-cli/internal/output"
	"github.com/vaughnbosu/cws-cli/pkg/service"
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

	output.Info("Cancelling submission for extension %s...", actx.ExtensionID)

	result, err := service.CancelSubmission(context.Background(), actx)
	if err != nil {
		return err
	}

	output.Info("Submission cancelled successfully.")
	if output.JSONMode() {
		return output.EmitJSON(result)
	}
	return nil
}
