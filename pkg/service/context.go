package service

import (
	"github.com/vaughnbosu/cws-cli/pkg/api"
	"github.com/vaughnbosu/cws-cli/pkg/auth"
	"github.com/vaughnbosu/cws-cli/pkg/config"
)

// Context bundles loaded config, an authenticated API client, and the resolved
// extension identity for a single operation.
type Context struct {
	Config      *config.Config
	Client      *api.Client
	ExtensionID string
	Profile     string
}

// ContextOptions selects which extension profile and optional ID override to use.
type ContextOptions struct {
	ExtensionID string
	Profile     string
}

// NewContext loads config, validates auth, resolves the extension ID, and
// returns an authenticated API context.
func NewContext(opts ContextOptions) (*Context, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, err
	}
	if err := config.ValidateAuth(cfg); err != nil {
		return nil, err
	}

	profile := opts.Profile
	extID, err := config.ResolveExtensionID(opts.ExtensionID, profile, cfg)
	if err != nil {
		return nil, err
	}

	authenticator := auth.NewOAuthAuthenticator(cfg.Auth.ClientID, cfg.Auth.ClientSecret, cfg.Auth.RefreshToken)
	client := api.NewClient(authenticator, cfg.PublisherID)

	return &Context{
		Config:      cfg,
		Client:      client,
		ExtensionID: extID,
		Profile:     profile,
	}, nil
}
