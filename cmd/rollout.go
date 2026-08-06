package cmd

import (
	"context"
	"fmt"
	"strconv"

	"github.com/spf13/cobra"
	"github.com/vaughnbosu/cws-cli/internal/output"
	"github.com/vaughnbosu/cws-cli/pkg/service"
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

	output.Info("Setting deploy percentage to %d%% for extension %s...", percentage, actx.ExtensionID)

	result, err := service.SetRollout(context.Background(), actx, percentage)
	if err != nil {
		return err
	}

	if result.Confirmed {
		output.Info("Deploy percentage set to %d%%.", percentage)
	} else {
		output.Info("Deploy percentage update accepted. Run 'cws status' to confirm the live value.")
	}

	if output.JSONMode() {
		return output.EmitJSON(result)
	}
	return nil
}
