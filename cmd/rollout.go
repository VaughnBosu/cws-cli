package cmd

import (
	"context"
	"fmt"
	"strconv"

	"github.com/spf13/cobra"
	"github.com/vaughnbosu/cws-cli/internal/api"
	"github.com/vaughnbosu/cws-cli/internal/auth"
	"github.com/vaughnbosu/cws-cli/internal/config"
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

	cfg, err := config.Load()
	if err != nil {
		return err
	}
	if err := config.ValidateAuth(cfg); err != nil {
		return err
	}

	extensionIDFlag, _ := cmd.Flags().GetString("extension-id")
	extensionID, err := config.ResolveExtensionID(extensionIDFlag, cfg)
	if err != nil {
		return err
	}

	authenticator := auth.NewOAuthAuthenticator(cfg.Auth.ClientID, cfg.Auth.ClientSecret, cfg.Auth.RefreshToken)
	client := api.NewClient(authenticator, cfg.PublisherID)
	ctx := context.Background()

	output.Info("Setting deploy percentage to %d%% for extension %s...", percentage, extensionID)

	resp, err := client.SetDeployPercentage(ctx, extensionID, percentage)
	if err != nil {
		return err
	}

	status, _, err := client.FetchStatus(ctx, extensionID)
	if err == nil {
		if current := publishedDeployPercentage(status); current >= 0 {
			output.Info("Deploy percentage set to %d%%.", current)
			return nil
		}
	}

	if resp.DeployPercentage > 0 {
		output.Info("Deploy percentage set to %d%%.", resp.DeployPercentage)
		return nil
	}

	output.Info("Deploy percentage update accepted. Run 'cws status' to confirm the live value.")
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
