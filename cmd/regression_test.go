package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestLocalValidationReportsMalformedConfig(t *testing.T) {
	dir := t.TempDir()
	withWorkingDirectory(t, dir)
	t.Setenv("HOME", dir)
	if err := os.WriteFile("cws.toml", []byte(`publisher_id = "unclosed`), 0600); err != nil {
		t.Fatal(err)
	}

	command := &cobra.Command{}
	command.Flags().Bool("local", true, "")
	command.Flags().String("ext", "", "")
	err := runValidate(command, []string{"."})
	if err == nil || !strings.Contains(err.Error(), "failed to parse ./cws.toml") {
		t.Fatalf("runValidate error = %v, want malformed local config error", err)
	}
}

func TestEnsureGitignoredCreatesAndUpdatesFile(t *testing.T) {
	dir := t.TempDir()
	withWorkingDirectory(t, dir)

	ensureGitignored("cws.toml")
	data, err := os.ReadFile(".gitignore")
	if err != nil {
		t.Fatalf(".gitignore was not created: %v", err)
	}
	if string(data) != "cws.toml\n" {
		t.Fatalf("created .gitignore = %q", data)
	}

	if err := os.WriteFile(".gitignore", []byte("dist/"), 0644); err != nil {
		t.Fatal(err)
	}
	ensureGitignored("cws.toml")
	ensureGitignored("cws.toml")
	data, err = os.ReadFile(".gitignore")
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "dist/\ncws.toml\n" {
		t.Fatalf("updated .gitignore = %q", data)
	}
}

func TestResolveVersion(t *testing.T) {
	if got := resolveVersion("v2.0.0", "v1.9.0"); got != "2.0.0" {
		t.Fatalf("linked version = %q", got)
	}
	if got := resolveVersion("dev", "v1.9.0"); got != "1.9.0" {
		t.Fatalf("module version fallback = %q", got)
	}
	if got := resolveVersion("dev", "(devel)"); got != "dev" {
		t.Fatalf("development version = %q", got)
	}
}

func withWorkingDirectory(t *testing.T, dir string) {
	t.Helper()
	original, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(filepath.Clean(dir)); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(original); err != nil {
			t.Errorf("restore working directory: %v", err)
		}
	})
}
