package cmd

import (
	"context"
	"strings"

	"github.com/spf13/cobra"
	"github.com/vaughnbosu/cws-cli/internal/api"
	"github.com/vaughnbosu/cws-cli/internal/auth"
	"github.com/vaughnbosu/cws-cli/internal/config"
	"github.com/vaughnbosu/cws-cli/internal/output"
)

var publishCmd = &cobra.Command{
	Use:   "publish",
	Short: "Publish the most recently uploaded version",
	Long: `Publish the most recently uploaded version of an extension.

Use --staged to submit for review without auto-publishing after approval.`,
	RunE: runPublish,
}

func init() {
	publishCmd.Flags().Bool("staged", false, "Use STAGED_PUBLISH: submit for review but don't auto-publish")
	rootCmd.AddCommand(publishCmd)
}

func runPublish(cmd *cobra.Command, args []string) error {
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

	staged, _ := cmd.Flags().GetBool("staged")

	authenticator := auth.NewOAuthAuthenticator(cfg.Auth.ClientID, cfg.Auth.ClientSecret, cfg.Auth.RefreshToken)
	client := api.NewClient(authenticator, cfg.PublisherID)
	ctx := context.Background()

	if staged {
		output.Info("Submitting extension %s for staged publish...", extensionID)
	} else {
		output.Info("Publishing extension %s...", extensionID)
	}

	resp, err := client.Publish(ctx, extensionID, staged)
	if err != nil {
		return err
	}

	if len(resp.Status) > 0 {
		output.Info("Status: %s", strings.Join(resp.Status, ", "))
	} else {
		output.Info("Publish submitted successfully.")
	}

	return nil
}
