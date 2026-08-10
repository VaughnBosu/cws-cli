package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
	"github.com/vaughnbosu/cws-cli/internal/privatefile"
)

// Config holds all configuration for the CLI.
type Config struct {
	PublisherID string                     `mapstructure:"publisher_id" toml:"publisher_id"`
	Auth        AuthConfig                 `mapstructure:"auth" toml:"auth"`
	Extensions  map[string]ExtensionConfig `mapstructure:"extensions" toml:"extensions,omitempty"`
	Package     PackageConfig              `mapstructure:"package" toml:"package,omitempty"`
}

// AuthConfig holds OAuth2 credentials.
type AuthConfig struct {
	ClientID     string `mapstructure:"client_id" toml:"client_id"`
	ClientSecret string `mapstructure:"client_secret" toml:"client_secret"`
	RefreshToken string `mapstructure:"refresh_token" toml:"refresh_token"`
}

// ExtensionConfig holds configuration for a single extension.
type ExtensionConfig struct {
	ID     string `mapstructure:"id" toml:"id,omitempty"`
	Source string `mapstructure:"source" toml:"source,omitempty"`
}

// PackageConfig controls how directories are zipped.
type PackageConfig struct {
	// Exclude adds patterns (path components or extensions) to the default exclusion list.
	Exclude []string `mapstructure:"exclude" toml:"exclude,omitempty"`
	// Include keeps files that a default exclusion would otherwise drop (e.g. "package.json").
	Include []string `mapstructure:"include" toml:"include,omitempty"`
}

// DefaultExtension is the extension profile used when --ext is not given.
const DefaultExtension = "default"

// Extension returns the named extension config, or a zero value if absent.
func (c *Config) Extension(name string) ExtensionConfig {
	if c == nil || c.Extensions == nil {
		return ExtensionConfig{}
	}
	return c.Extensions[name]
}

// GlobalConfigDir returns the path to the global config directory.
func GlobalConfigDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".config", "cws")
}

// GlobalConfigPath returns the path to the global config file.
// Returns an error if the home directory cannot be determined.
func GlobalConfigPath() (string, error) {
	dir := GlobalConfigDir()
	if dir == "" {
		return "", fmt.Errorf("could not determine home directory for global config")
	}
	return filepath.Join(dir, "cws.toml"), nil
}

// Load reads configuration from config files and environment variables.
// Priority: env vars > local cws.toml > global cws.toml
func Load() (*Config, error) {
	v := viper.New()
	v.SetConfigType("toml")

	// Environment variable bindings
	v.SetEnvPrefix("CWS")
	v.AutomaticEnv()
	_ = v.BindEnv("auth.client_id", "CWS_CLIENT_ID")
	_ = v.BindEnv("auth.client_secret", "CWS_CLIENT_SECRET")
	_ = v.BindEnv("auth.refresh_token", "CWS_REFRESH_TOKEN")
	_ = v.BindEnv("publisher_id", "CWS_PUBLISHER_ID")
	_ = v.BindEnv("extensions.default.id", "CWS_EXTENSION_ID")

	// Config files are targeted by exact path. Viper's name-based search
	// (SetConfigName) would also match a bare file named "cws" — such as a
	// locally built binary — and try to parse it as TOML.

	// Read global config first (lowest priority file)
	if globalPath, err := GlobalConfigPath(); err == nil {
		if _, err := os.Stat(globalPath); err == nil {
			v.SetConfigFile(globalPath)
			if err := v.ReadInConfig(); err != nil {
				return nil, fmt.Errorf("failed to parse %s: %w", globalPath, err)
			}
		}
	}

	// Read local config (overrides global)
	if _, err := os.Stat("cws.toml"); err == nil {
		localV := viper.New()
		localV.SetConfigFile("cws.toml")
		localV.SetConfigType("toml")
		if err := localV.ReadInConfig(); err != nil {
			return nil, fmt.Errorf("failed to parse ./cws.toml: %w", err)
		}
		// MergeConfigMap merges at config-file precedence, so env vars
		// (bound above) still win over local file values.
		if err := v.MergeConfigMap(localV.AllSettings()); err != nil {
			return nil, fmt.Errorf("failed to merge local config: %w", err)
		}
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to parse configuration: %w", err)
	}

	return &cfg, nil
}

// ResolveExtensionID returns the extension ID from the flag override, env, or the named profile.
func ResolveExtensionID(flagValue, extName string, cfg *Config) (string, error) {
	if flagValue != "" {
		return flagValue, nil
	}
	if extName == "" {
		extName = DefaultExtension
	}
	if extName == DefaultExtension {
		if id := os.Getenv("CWS_EXTENSION_ID"); id != "" {
			return id, nil
		}
	}
	if id := cfg.Extension(extName).ID; id != "" {
		return id, nil
	}
	if extName != DefaultExtension {
		return "", fmt.Errorf("no extension named %q in cws.toml. Add an [extensions.%s] section with an id", extName, extName)
	}
	return "", fmt.Errorf("no extension ID specified. Use --extension-id flag, set CWS_EXTENSION_ID, or add [extensions.default] to cws.toml")
}

// ResolveSource returns the source path from the flag override or the named profile.
func ResolveSource(argValue, extName string, cfg *Config) string {
	if argValue != "" {
		return argValue
	}
	if extName == "" {
		extName = DefaultExtension
	}
	if src := cfg.Extension(extName).Source; src != "" {
		return src
	}
	return "."
}

// ValidateAuth checks that all required auth fields are present.
func ValidateAuth(cfg *Config) error {
	if cfg.Auth.ClientID == "" {
		return fmt.Errorf("no configuration found. Run 'cws init --global' to set up credentials, or set CWS_* environment variables")
	}
	if cfg.Auth.ClientSecret == "" {
		return fmt.Errorf("client secret not configured. Run 'cws init --global' to set up credentials, or set CWS_CLIENT_SECRET")
	}
	if cfg.Auth.RefreshToken == "" {
		return fmt.Errorf("refresh token not configured. Run 'cws init --global' to set up credentials, or set CWS_REFRESH_TOKEN")
	}
	if cfg.PublisherID == "" {
		return fmt.Errorf("publisher ID not configured. Run 'cws init --global' to set up credentials, or set CWS_PUBLISHER_ID")
	}
	return nil
}

// WriteConfig writes a config file to the specified path.
func WriteConfig(path string, cfg *Config) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	var content string
	def := cfg.Extension(DefaultExtension)

	// Check if this is a global config (has auth) or project config
	if cfg.Auth.ClientID != "" {
		content = fmt.Sprintf(`publisher_id = %q

[auth]
client_id = %q
client_secret = %q
refresh_token = %q
`,
			cfg.PublisherID,
			cfg.Auth.ClientID,
			cfg.Auth.ClientSecret,
			cfg.Auth.RefreshToken,
		)

		if def.ID != "" {
			content += fmt.Sprintf(`
[extensions.default]
id = %q
`, def.ID)
		}
	} else {
		content = fmt.Sprintf(`[extensions.default]
id = %q
`, def.ID)
		if def.Source != "" {
			content += fmt.Sprintf(`source = %q
`, def.Source)
		}
	}

	if err := privatefile.Write(path, []byte(content)); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}
	return nil
}
