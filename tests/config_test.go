package tests

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/vaughnbosu/cws-cli/pkg/config"
)

func defaultExtensions(ec config.ExtensionConfig) map[string]config.ExtensionConfig {
	return map[string]config.ExtensionConfig{config.DefaultExtension: ec}
}

// --- ResolveExtensionID tests ---

func TestResolveExtensionID_Flag(t *testing.T) {
	got, err := config.ResolveExtensionID("flag-id", "", &config.Config{
		Extensions: defaultExtensions(config.ExtensionConfig{ID: "config-id"}),
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
	got, err := config.ResolveExtensionID("", "", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "env-id" {
		t.Errorf("got %q, want %q", got, "env-id")
	}
}

func TestResolveExtensionID_Config(t *testing.T) {
	got, err := config.ResolveExtensionID("", "", &config.Config{
		Extensions: defaultExtensions(config.ExtensionConfig{ID: "config-id"}),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "config-id" {
		t.Errorf("got %q, want %q", got, "config-id")
	}
}

func TestResolveExtensionID_NamedProfile(t *testing.T) {
	cfg := &config.Config{
		Extensions: map[string]config.ExtensionConfig{
			"default": {ID: "default-id"},
			"beta":    {ID: "beta-id"},
		},
	}
	got, err := config.ResolveExtensionID("", "beta", cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "beta-id" {
		t.Errorf("got %q, want %q", got, "beta-id")
	}
}

func TestResolveExtensionID_NamedProfileMissing(t *testing.T) {
	_, err := config.ResolveExtensionID("", "nope", &config.Config{})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "nope") {
		t.Errorf("error = %q, want mention of missing profile name", err.Error())
	}
}

// Env var CWS_EXTENSION_ID must not hijack an explicitly named profile.
func TestResolveExtensionID_NamedProfileIgnoresEnv(t *testing.T) {
	t.Setenv("CWS_EXTENSION_ID", "env-id")
	cfg := &config.Config{
		Extensions: map[string]config.ExtensionConfig{"beta": {ID: "beta-id"}},
	}
	got, err := config.ResolveExtensionID("", "beta", cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "beta-id" {
		t.Errorf("got %q, want %q", got, "beta-id")
	}
}

func TestResolveExtensionID_None(t *testing.T) {
	_, err := config.ResolveExtensionID("", "", &config.Config{})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// --- ResolveSource tests ---

func TestResolveSource_Arg(t *testing.T) {
	got := config.ResolveSource("./dist", "", &config.Config{
		Extensions: defaultExtensions(config.ExtensionConfig{Source: "./build"}),
	})
	if got != "./dist" {
		t.Errorf("got %q, want %q", got, "./dist")
	}
}

func TestResolveSource_Config(t *testing.T) {
	got := config.ResolveSource("", "", &config.Config{
		Extensions: defaultExtensions(config.ExtensionConfig{Source: "./build"}),
	})
	if got != "./build" {
		t.Errorf("got %q, want %q", got, "./build")
	}
}

func TestResolveSource_Default(t *testing.T) {
	got := config.ResolveSource("", "", &config.Config{})
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
		Extensions: defaultExtensions(config.ExtensionConfig{ID: "ext-abc"}),
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
		Extensions: defaultExtensions(config.ExtensionConfig{ID: "ext-abc", Source: "./dist"}),
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

func TestWriteConfig_OverwritesExistingFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "cws.toml")
	if err := os.WriteFile(path, []byte("stale content"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := config.WriteConfig(path, &config.Config{
		Auth: config.AuthConfig{ClientID: "id", ClientSecret: "secret", RefreshToken: "token"},
	}); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "stale content") || !strings.Contains(string(data), `refresh_token = "token"`) {
		t.Fatalf("overwritten config = %q", data)
	}
}

func TestWriteConfig_TightensExistingFilePermissions(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows does not expose Unix permission bits")
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "cws.toml")
	if err := os.WriteFile(path, []byte("old"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(path, 0644); err != nil {
		t.Fatal(err)
	}

	err := config.WriteConfig(path, &config.Config{
		Auth: config.AuthConfig{ClientID: "id", ClientSecret: "secret", RefreshToken: "token"},
	})
	if err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := info.Mode().Perm(); got != 0600 {
		t.Fatalf("config permissions = %04o, want 0600", got)
	}
}

func TestWriteConfig_ReplacesSymlinkWithoutChangingTarget(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "target")
	path := filepath.Join(dir, "cws.toml")
	if err := os.WriteFile(target, []byte("leave me alone"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, path); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}

	if err := config.WriteConfig(path, &config.Config{
		Auth: config.AuthConfig{ClientID: "id", ClientSecret: "secret", RefreshToken: "token"},
	}); err != nil {
		t.Fatal(err)
	}
	targetData, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if string(targetData) != "leave me alone" {
		t.Fatalf("symlink target was overwritten: %q", targetData)
	}
	info, err := os.Lstat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		t.Fatal("config path is still a symlink")
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
	if cfg.Extension(config.DefaultExtension).ID != "ext-from-file" {
		t.Errorf("ExtensionID = %q, want %q", cfg.Extension(config.DefaultExtension).ID, "ext-from-file")
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
	if cfg.Extension(config.DefaultExtension).ID != "ext-from-env" {
		t.Errorf("ExtensionID = %q, want %q", cfg.Extension(config.DefaultExtension).ID, "ext-from-env")
	}
}

// Regression test: env vars must override local cws.toml values
// (documented priority: env vars > local cws.toml > global cws.toml).
func TestLoad_EnvVarOverridesLocalFile(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "cws.toml")
	os.WriteFile(configPath, []byte(`
publisher_id = "pub-from-file"

[auth]
client_id = "id-from-file"
client_secret = "secret-from-file"
refresh_token = "token-from-file"
`), 0600)

	origDir, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origDir)

	t.Setenv("CWS_CLIENT_ID", "id-from-env")
	t.Setenv("CWS_REFRESH_TOKEN", "token-from-env")
	t.Setenv("CWS_PUBLISHER_ID", "pub-from-env")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}
	if cfg.Auth.ClientID != "id-from-env" {
		t.Errorf("ClientID = %q, want env value to win over local file", cfg.Auth.ClientID)
	}
	if cfg.Auth.RefreshToken != "token-from-env" {
		t.Errorf("RefreshToken = %q, want env value to win over local file", cfg.Auth.RefreshToken)
	}
	if cfg.PublisherID != "pub-from-env" {
		t.Errorf("PublisherID = %q, want env value to win over local file", cfg.PublisherID)
	}
	// Unset-in-env values still come from the file.
	if cfg.Auth.ClientSecret != "secret-from-file" {
		t.Errorf("ClientSecret = %q, want file value when env is unset", cfg.Auth.ClientSecret)
	}
}

// Regression test: a malformed cws.toml must be reported, not silently ignored.
func TestLoad_MalformedLocalFile(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "cws.toml")
	os.WriteFile(configPath, []byte(`publisher_id = "unclosed`), 0600)

	origDir, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origDir)

	_, err := config.Load()
	if err == nil {
		t.Fatal("expected parse error for malformed cws.toml, got nil")
	}
	if !strings.Contains(err.Error(), "cws.toml") {
		t.Errorf("error = %q, want mention of cws.toml", err.Error())
	}
}

// Regression test: a file named exactly "cws" (e.g. a locally built binary)
// must not be picked up and parsed as the config file.
func TestLoad_IgnoresBareCwsFile(t *testing.T) {
	dir := t.TempDir()
	// Simulate `go build -o cws .` output: binary junk in a file named "cws".
	os.WriteFile(filepath.Join(dir, "cws"), []byte("\xcf\xfa\xed\xfenot toml"), 0755)

	origDir, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origDir)

	if _, err := config.Load(); err != nil {
		t.Fatalf("Load error with bare cws file present: %v", err)
	}
}
