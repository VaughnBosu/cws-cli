package cmd

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
	"syscall"

	"github.com/spf13/cobra"
	"github.com/vaughnbosu/cws-cli/internal/auth"
	"github.com/vaughnbosu/cws-cli/internal/config"
	"github.com/vaughnbosu/cws-cli/internal/output"
	"golang.org/x/term"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Interactive credential setup wizard",
	Long: `Set up Chrome Web Store API credentials interactively.

This wizard will guide you through configuring:
  - OAuth2 Client ID and Secret (from Google Cloud Console)
  - Refresh Token (via browser sign-in, or pasted manually)
  - Publisher ID (from Chrome Web Store Developer Dashboard)`,
	RunE: runInit,
}

func init() {
	initCmd.Flags().Bool("global", false, "Write config to ~/.config/cws/cws.toml instead of current directory")
	rootCmd.AddCommand(initCmd)
}

func runInit(cmd *cobra.Command, args []string) error {
	reader := bufio.NewReader(os.Stdin)
	global, _ := cmd.Flags().GetBool("global")

	fmt.Println("Chrome Web Store CLI — Credential Setup")
	fmt.Println("========================================")
	fmt.Println()

	// Client ID
	fmt.Println("Step 1: Client ID")
	fmt.Println("  Create OAuth2 credentials in the Google Cloud Console:")
	fmt.Println("  https://console.cloud.google.com/apis/credentials")
	fmt.Println("  Choose application type \"Desktop app\" and make sure the")
	fmt.Println("  Chrome Web Store API is enabled for your project.")
	fmt.Println()
	clientID, err := prompt(reader, "Client ID: ")
	if err != nil {
		return err
	}

	// Client Secret
	fmt.Println()
	fmt.Println("Step 2: Client Secret")
	clientSecret, err := promptSecret(reader, "Client Secret: ")
	if err != nil {
		return err
	}

	// Refresh Token
	fmt.Println()
	fmt.Println("Step 3: Refresh Token")
	fmt.Println("  Press Enter to sign in with your browser (recommended),")
	fmt.Println("  or paste a refresh token obtained elsewhere.")
	fmt.Println()
	refreshToken, err := promptSecret(reader, "Refresh Token (Enter for browser sign-in): ")
	if err != nil {
		return err
	}
	if refreshToken == "" {
		refreshToken, err = auth.AcquireRefreshToken(context.Background(), clientID, clientSecret)
		if err != nil {
			return fmt.Errorf("browser sign-in failed: %w", err)
		}
		output.Info("Refresh token acquired.")
	}

	// Publisher ID
	fmt.Println()
	fmt.Println("Step 4: Publisher ID")
	fmt.Println("  Find your Publisher ID in the Chrome Web Store Developer Dashboard:")
	fmt.Println("  Developer Dashboard → Account section")
	fmt.Println()
	publisherID, err := prompt(reader, "Publisher ID: ")
	if err != nil {
		return err
	}

	// Optional Extension ID
	fmt.Println()
	fmt.Println("Step 5: Default Extension ID (optional, press Enter to skip)")
	extensionID, err := prompt(reader, "Extension ID: ")
	if err != nil {
		return err
	}

	// Validate credentials
	fmt.Println()
	output.Progress("Validating credentials...")
	if err := auth.ValidateCredentials(clientID, clientSecret, refreshToken); err != nil {
		fmt.Println()
		return fmt.Errorf("credential validation failed: %w", err)
	}
	fmt.Println(" valid!")

	// Write config
	cfg := &config.Config{
		PublisherID: publisherID,
		Auth: config.AuthConfig{
			ClientID:     clientID,
			ClientSecret: clientSecret,
			RefreshToken: refreshToken,
		},
	}
	if extensionID != "" {
		cfg.Extensions = map[string]config.ExtensionConfig{
			config.DefaultExtension: {ID: extensionID},
		}
	}

	var configPath string
	if global {
		var err error
		configPath, err = config.GlobalConfigPath()
		if err != nil {
			return err
		}
	} else {
		configPath = "cws.toml"
	}

	if err := config.WriteConfig(configPath, cfg); err != nil {
		return err
	}

	fmt.Println()
	output.Info("Configuration saved to %s", configPath)
	if !global {
		ensureGitignored("cws.toml")
	}

	return nil
}

// ensureGitignored appends the entry to .gitignore when one exists in the
// current directory and the entry is missing. Local cws.toml holds secrets.
func ensureGitignored(entry string) {
	data, err := os.ReadFile(".gitignore")
	if err != nil {
		output.Info("Tip: Add %s to .gitignore — it contains secrets.", entry)
		return
	}
	for _, line := range strings.Split(string(data), "\n") {
		if strings.TrimSpace(line) == entry {
			return
		}
	}

	content := string(data)
	if content != "" && !strings.HasSuffix(content, "\n") {
		content += "\n"
	}
	content += entry + "\n"
	if err := os.WriteFile(".gitignore", []byte(content), 0644); err != nil {
		output.Info("Tip: Add %s to .gitignore — it contains secrets.", entry)
		return
	}
	output.Info("Added %s to .gitignore (it contains secrets).", entry)
}

func prompt(reader *bufio.Reader, label string) (string, error) {
	fmt.Print(label)
	input, err := reader.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("failed to read input: %w", err)
	}
	return strings.TrimSpace(input), nil
}

// promptSecret reads input without echoing when stdin is a terminal,
// falling back to plain input otherwise (e.g. piped stdin in CI).
func promptSecret(reader *bufio.Reader, label string) (string, error) {
	fmt.Print(label)
	fd := int(syscall.Stdin)
	if term.IsTerminal(fd) {
		data, err := term.ReadPassword(fd)
		fmt.Println()
		if err != nil {
			return "", fmt.Errorf("failed to read input: %w", err)
		}
		return strings.TrimSpace(string(data)), nil
	}
	return prompt(reader, "")
}
