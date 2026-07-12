package cmd

import (
	"github.com/spf13/cobra"
	"github.com/vaughnbosu/cws-cli/internal/api"
	"github.com/vaughnbosu/cws-cli/internal/auth"
	"github.com/vaughnbosu/cws-cli/internal/config"
)

// apiContext bundles the loaded config, API client, and resolved extension
// identity that most commands need.
type apiContext struct {
	cfg         *config.Config
	client      *api.Client
	extensionID string
	extName     string
}

// newAPIContext loads config, validates auth, resolves the target extension,
// and builds an authenticated API client.
func newAPIContext(cmd *cobra.Command) (*apiContext, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, err
	}
	if err := config.ValidateAuth(cfg); err != nil {
		return nil, err
	}

	extensionIDFlag, _ := cmd.Flags().GetString("extension-id")
	extName, _ := cmd.Flags().GetString("ext")
	extensionID, err := config.ResolveExtensionID(extensionIDFlag, extName, cfg)
	if err != nil {
		return nil, err
	}

	authenticator := auth.NewOAuthAuthenticator(cfg.Auth.ClientID, cfg.Auth.ClientSecret, cfg.Auth.RefreshToken)
	client := api.NewClient(authenticator, cfg.PublisherID)

	return &apiContext{
		cfg:         cfg,
		client:      client,
		extensionID: extensionID,
		extName:     extName,
	}, nil
}
