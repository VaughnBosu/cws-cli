package tests

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/VaughnBosu/cws-cli/internal/config"
)

// --- ResolveExtensionID tests ---

func TestResolveExtensionID_Flag(t *testing.T) {
	got, err := config.ResolveExtensionID("flag-id", &config.Config{
		Extensions: config.ExtensionsConfig{Default: config.ExtensionConfig{ID: "config-id"}},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "flag-id" {
		t.Errorf("got %q, want %q", got, "flag-id")
	}
}

func TestResolveExtensionID_EnvVar(t *testing.T) {
	t.Setenv("CWS_EXTENSION_ID", "env-id")
	got, err := config.ResolveExtensionID("", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "env-id" {
		t.Errorf("got %q, want %q", got, "env-id")
	}
}

func TestResolveExtensionID_Config(t *testing.T) {
	got, err := config.ResolveExtensionID("", &config.Config{
		Extensions: config.ExtensionsConfig{Default: config.ExtensionConfig{ID: "config-id"}},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "config-id" {
		t.Errorf("got %q, want %q", got, "config-id")
	}
}

func TestResolveExtensionID_None(t *testing.T) {
	_, err := config.ResolveExtensionID("", &config.Config{})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// --- ResolveSource tests ---

func TestResolveSource_Arg(t *testing.T) {
	got := config.ResolveSource("./dist", &config.Config{
		Extensions: config.ExtensionsConfig{Default: config.ExtensionConfig{Source: "./build"}},
	})
	if got != "./dist" {
		t.Errorf("got %q, want %q", got, "./dist")
	}
}

func TestResolveSource_Config(t *testing.T) {
	got := config.ResolveSource("", &config.Config{
		Extensions: config.ExtensionsConfig{Default: config.ExtensionConfig{Source: "./build"}},
	})
	if got != "./build" {
		t.Errorf("got %q, want %q", got, "./build")
	}
}

func TestResolveSource_Default(t *testing.T) {
	got := config.ResolveSource("", &config.Config{})
	if got != "." {
		t.Errorf("got %q, want %q", got, ".")
	}
}

// --- ValidateAuth tests ---

func TestValidateAuth_Valid(t *testing.T) {
	cfg := &config.Config{
		PublisherID: "pub-123",
		Auth: config.AuthConfig{
			ClientID:     "client-id",
			ClientSecret: "client-secret",
			RefreshToken: "refresh-token",
		},
	}
	if err := config.ValidateAuth(cfg); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestValidateAuth_MissingClientID(t *testing.T) {
	cfg := &config.Config{
		PublisherID: "pub-123",
		Auth: config.AuthConfig{
			ClientSecret: "secret",
			RefreshToken: "token",
		},
	}
	err := config.ValidateAuth(cfg)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "no configuration found") {
		t.Errorf("error = %q, want message about no configuration", err.Error())
	}
}

func TestValidateAuth_MissingSecret(t *testing.T) {
	cfg := &config.Config{
		PublisherID: "pub-123",
		Auth: config.AuthConfig{
			ClientID:     "id",
			RefreshToken: "token",
		},
	}
	err := config.ValidateAuth(cfg)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "client secret") {
		t.Errorf("error = %q, want message about client secret", err.Error())
	}
}

func TestValidateAuth_MissingToken(t *testing.T) {
	cfg := &config.Config{
		PublisherID: "pub-123",
		Auth: config.AuthConfig{
			ClientID:     "id",
			ClientSecret: "secret",
		},
	}
	err := config.ValidateAuth(cfg)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "refresh token") {
		t.Errorf("error = %q, want message about refresh token", err.Error())
	}
}

func TestValidateAuth_MissingPublisherID(t *testing.T) {
	cfg := &config.Config{
		Auth: config.AuthConfig{
			ClientID:     "id",
			ClientSecret: "secret",
			RefreshToken: "token",
		},
	}
	err := config.ValidateAuth(cfg)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "publisher ID") {
		t.Errorf("error = %q, want message about publisher ID", err.Error())
	}
}

// --- WriteConfig tests ---

func TestWriteConfig_Full(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "cws.toml")

	cfg := &config.Config{
		PublisherID: "pub-123",
		Auth: config.AuthConfig{
			ClientID:     "my-client-id",
			ClientSecret: "my-secret",
			RefreshToken: "my-token",
		},
		Extensions: config.ExtensionsConfig{Default: config.ExtensionConfig{ID: "ext-abc"}},
	}

	if err := config.WriteConfig(path, cfg); err != nil {
		t.Fatalf("WriteConfig error: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile error: %v", err)
	}

	content := string(data)
	for _, want := range []string{"pub-123", "my-client-id", "my-secret", "my-token", "ext-abc"} {
		if !strings.Contains(content, want) {
			t.Errorf("config file missing %q", want)
		}
	}
}

func TestWriteConfig_ProjectOnly(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "cws.toml")

	cfg := &config.Config{
		Extensions: config.ExtensionsConfig{Default: config.ExtensionConfig{ID: "ext-abc", Source: "./dist"}},
	}

	if err := config.WriteConfig(path, cfg); err != nil {
		t.Fatalf("WriteConfig error: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile error: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "ext-abc") {
		t.Error("config file missing extension ID")
	}
	if !strings.Contains(content, "./dist") {
		t.Error("config file missing source")
	}
	if strings.Contains(content, "[auth]") {
		t.Error("project-only config should not contain [auth] section")
	}
}

// --- Load tests ---

func TestLoad_LocalFile(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "cws.toml")
	os.WriteFile(configPath, []byte(`
publisher_id = "pub-from-file"

[auth]
client_id = "id-from-file"
client_secret = "secret-from-file"
refresh_token = "token-from-file"

[extensions.default]
id = "ext-from-file"
`), 0600)

	origDir, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origDir)

	t.Setenv("CWS_CLIENT_ID", "")
	t.Setenv("CWS_CLIENT_SECRET", "")
	t.Setenv("CWS_REFRESH_TOKEN", "")
	t.Setenv("CWS_PUBLISHER_ID", "")
	t.Setenv("CWS_EXTENSION_ID", "")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}
	if cfg.PublisherID != "pub-from-file" {
		t.Errorf("PublisherID = %q, want %q", cfg.PublisherID, "pub-from-file")
	}
	if cfg.Auth.ClientID != "id-from-file" {
		t.Errorf("ClientID = %q, want %q", cfg.Auth.ClientID, "id-from-file")
	}
	if cfg.Extensions.Default.ID != "ext-from-file" {
		t.Errorf("ExtensionID = %q, want %q", cfg.Extensions.Default.ID, "ext-from-file")
	}
}

func TestLoad_EnvVarWithNoFile(t *testing.T) {
	dir := t.TempDir()

	origDir, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origDir)

	t.Setenv("CWS_CLIENT_ID", "id-from-env")
	t.Setenv("CWS_CLIENT_SECRET", "secret-from-env")
	t.Setenv("CWS_REFRESH_TOKEN", "token-from-env")
	t.Setenv("CWS_PUBLISHER_ID", "pub-from-env")
	t.Setenv("CWS_EXTENSION_ID", "ext-from-env")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}
	if cfg.Auth.ClientID != "id-from-env" {
		t.Errorf("ClientID = %q, want %q", cfg.Auth.ClientID, "id-from-env")
	}
	if cfg.PublisherID != "pub-from-env" {
		t.Errorf("PublisherID = %q, want %q", cfg.PublisherID, "pub-from-env")
	}
	if cfg.Extensions.Default.ID != "ext-from-env" {
		t.Errorf("ExtensionID = %q, want %q", cfg.Extensions.Default.ID, "ext-from-env")
	}
}
