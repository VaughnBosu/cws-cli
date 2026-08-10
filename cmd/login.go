package cmd

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/spf13/cobra"
	"github.com/vaughnbosu/cws-cli/internal/output"
	"github.com/vaughnbosu/cws-cli/internal/privatefile"
	"github.com/vaughnbosu/cws-cli/pkg/auth"
	"github.com/vaughnbosu/cws-cli/pkg/config"
)

var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Re-acquire a refresh token via browser sign-in",
	Long: `Sign in with your browser to obtain a fresh OAuth refresh token.

Uses the client ID and secret from your existing configuration (run
'cws init --global' first if you have none) and updates the config file in place.`,
	RunE: runLogin,
}

func init() {
	rootCmd.AddCommand(loginCmd)
}

func runLogin(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	if cfg.Auth.ClientID == "" || cfg.Auth.ClientSecret == "" {
		return fmt.Errorf("no OAuth client configured. Run 'cws init --global' to set up credentials first")
	}

	refreshToken, err := auth.AcquireRefreshToken(context.Background(), cfg.Auth.ClientID, cfg.Auth.ClientSecret)
	if err != nil {
		return fmt.Errorf("browser sign-in failed: %w", err)
	}

	if err := auth.ValidateCredentials(cfg.Auth.ClientID, cfg.Auth.ClientSecret, refreshToken); err != nil {
		return fmt.Errorf("acquired token failed validation: %w", err)
	}

	cfg.Auth.RefreshToken = refreshToken

	configPath, err := authConfigPath()
	if err != nil {
		return err
	}
	if err := SaveRefreshToken(configPath, refreshToken, cfg); err != nil {
		return err
	}

	output.Info("Signed in. Refresh token saved to %s", configPath)
	return nil
}

// SaveRefreshToken updates only the refresh_token entry in an existing config
// file, preserving every other section, key, and comment. Rewriting the whole
// file from the merged config would drop named extension profiles, [package]
// settings, and anything else Load() doesn't round-trip.
func SaveRefreshToken(path, token string, cfg *config.Config) error {
	data, err := os.ReadFile(path)
	if err != nil {
		// No existing file — write a fresh one from the loaded credentials.
		return config.WriteConfig(path, cfg)
	}

	line := fmt.Sprintf("refresh_token = %q", token)
	content := string(data)

	if tokenLineRe.MatchString(content) {
		content = tokenLineRe.ReplaceAllLiteralString(content, line)
	} else if strings.Contains(content, "[auth]") {
		content = strings.Replace(content, "[auth]", "[auth]\n"+line, 1)
	} else {
		if content != "" && !strings.HasSuffix(content, "\n") {
			content += "\n"
		}
		content += "\n[auth]\n" + line + "\n"
	}

	return privatefile.Write(path, []byte(content))
}

var tokenLineRe = regexp.MustCompile(`(?m)^[ \t]*refresh_token[ \t]*=.*$`)

// authConfigPath returns the config file that holds the [auth] section:
// the local ./cws.toml when it contains one, otherwise the global config.
func authConfigPath() (string, error) {
	if data, err := os.ReadFile("cws.toml"); err == nil {
		if strings.Contains(string(data), "[auth]") {
			return "cws.toml", nil
		}
	}
	return config.GlobalConfigPath()
}
