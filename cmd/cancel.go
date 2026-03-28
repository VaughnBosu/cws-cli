package cmd

import (
	"context"

	"github.com/spf13/cobra"
	"github.com/VaughnBosu/cws-cli/internal/api"
	"github.com/VaughnBosu/cws-cli/internal/auth"
	"github.com/VaughnBosu/cws-cli/internal/config"
	"github.com/VaughnBosu/cws-cli/internal/output"
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

	output.Info("Cancelling submission for extension %s...", extensionID)

	_, err = client.CancelSubmission(ctx, extensionID)
	if err != nil {
		return err
	}

	output.Info("Submission cancelled successfully.")
	return nil
}
