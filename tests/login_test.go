package tests

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/vaughnbosu/cws-cli/cmd"
	"github.com/vaughnbosu/cws-cli/pkg/config"
)

// Regression test: cws login must update only refresh_token, not rewrite the
// file from the merged config (which would drop named profiles, [package]
// sections, sources, and comments).
func TestSaveRefreshToken_PreservesOtherSections(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "cws.toml")

	original := `# my config
publisher_id = "pub123"

[auth]
client_id = "id"
client_secret = "secret"
refresh_token = "old-token"

[extensions.default]
id = "aaaa"
source = "./dist"

[extensions.beta]
id = "bbbb"

[package]
exclude = ["docs"]
`
	if err := os.WriteFile(path, []byte(original), 0600); err != nil {
		t.Fatal(err)
	}

	if err := cmd.SaveRefreshToken(path, "new-token", &config.Config{}); err != nil {
		t.Fatalf("SaveRefreshToken error: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	got := string(data)

	if strings.Contains(got, "old-token") {
		t.Error("old refresh token still present")
	}
	if !strings.Contains(got, `refresh_token = "new-token"`) {
		t.Errorf("new refresh token not written:\n%s", got)
	}
	for _, want := range []string{"# my config", "[extensions.beta]", `source = "./dist"`, "[package]", `exclude = ["docs"]`} {
		if !strings.Contains(got, want) {
			t.Errorf("login rewrite lost %q:\n%s", want, got)
		}
	}

	// The result must still be a loadable config with the new token.
	t.Setenv("CWS_CLIENT_ID", "")
	t.Setenv("CWS_CLIENT_SECRET", "")
	t.Setenv("CWS_REFRESH_TOKEN", "")
	t.Setenv("CWS_PUBLISHER_ID", "")
	t.Setenv("CWS_EXTENSION_ID", "")
	cwd, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(cwd) }()

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("reloading config after login rewrite: %v", err)
	}
	if cfg.Auth.RefreshToken != "new-token" {
		t.Errorf("reloaded refresh_token = %q, want new-token", cfg.Auth.RefreshToken)
	}
	if cfg.Extension("beta").ID != "bbbb" {
		t.Errorf("reloaded beta profile id = %q, want bbbb", cfg.Extension("beta").ID)
	}
}

// When the file has an [auth] section but no refresh_token line, the token is
// inserted into that section.
func TestSaveRefreshToken_InsertsIntoAuthSection(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "cws.toml")
	if err := os.WriteFile(path, []byte("[auth]\nclient_id = \"id\"\n"), 0600); err != nil {
		t.Fatal(err)
	}

	if err := cmd.SaveRefreshToken(path, "tok", &config.Config{}); err != nil {
		t.Fatalf("SaveRefreshToken error: %v", err)
	}
	data, _ := os.ReadFile(path)
	if !strings.Contains(string(data), "[auth]\nrefresh_token = \"tok\"") {
		t.Errorf("token not inserted into [auth] section:\n%s", data)
	}
}

func TestSaveRefreshToken_TightensExistingFilePermissions(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows does not expose Unix permission bits")
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "cws.toml")
	if err := os.WriteFile(path, []byte("[auth]\nrefresh_token = \"old\"\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(path, 0644); err != nil {
		t.Fatal(err)
	}

	if err := cmd.SaveRefreshToken(path, "new", &config.Config{}); err != nil {
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
