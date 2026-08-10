package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/vaughnbosu/cws-cli/pkg/api"
	"github.com/vaughnbosu/cws-cli/pkg/config"
	cwszip "github.com/vaughnbosu/cws-cli/pkg/zip"
)

type staticAuthenticator struct{}

func (staticAuthenticator) AccessToken(context.Context) (string, error) {
	return "token", nil
}

func TestValidateChecksIconsInBuiltPackage(t *testing.T) {
	tests := []struct {
		name     string
		iconPath string
		writeAt  func(string) string
	}{
		{
			name:     "excluded directory",
			iconPath: "node_modules/icon.png",
			writeAt: func(dir string) string {
				return filepath.Join(dir, "node_modules", "icon.png")
			},
		},
		{
			name:     "parent traversal",
			iconPath: "../outside.png",
			writeAt: func(dir string) string {
				return filepath.Join(dir, "..", "outside.png")
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			dir := t.TempDir()
			manifest := fmt.Sprintf(`{"name":"Extension","version":"1.0","manifest_version":3,"icons":{"128":%q}}`, test.iconPath)
			writeTestFile(t, filepath.Join(dir, "manifest.json"), manifest)
			writeTestFile(t, test.writeAt(dir), "png")

			result, _, err := Validate(context.Background(), nil, ValidateOptions{
				Source:    dir,
				LocalOnly: true,
			})
			if err != nil {
				t.Fatal(err)
			}
			if result.Passed {
				t.Fatalf("validation passed even though %q is absent from the package", test.iconPath)
			}
			if !hasFailedCheck(result, "Missing icon files: "+test.iconPath) {
				t.Fatalf("missing icon failure not reported: %+v", result.Checks)
			}
		})
	}
}

func TestPackageRejectsExcludedManifest(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, filepath.Join(dir, "manifest.json"), `{"name":"Extension","version":"1.0","manifest_version":3}`)
	writeTestFile(t, filepath.Join(dir, "worker.js"), "")
	pkg := config.PackageConfig{Exclude: []string{"manifest.json"}}

	result, _, err := Validate(context.Background(), &Context{Config: &config.Config{Package: pkg}}, ValidateOptions{
		Source:    dir,
		LocalOnly: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Passed || !hasFailedCheckContaining(result, "Packaged manifest.json") {
		t.Fatalf("validation checks = %+v", result.Checks)
	}

	if _, err := PrepareZip(dir, pkg); err == nil || !strings.Contains(err.Error(), "package does not contain a manifest.json") {
		t.Fatalf("PrepareZip error = %v", err)
	}
	if _, err := Pack(dir, filepath.Join(dir, "extension.zip"), pkg); err == nil || !strings.Contains(err.Error(), "package does not contain a manifest.json") {
		t.Fatalf("Pack error = %v", err)
	}
}

func TestPackRepeatedlyExcludesPreviousArchive(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, filepath.Join(dir, "manifest.json"), `{"name":"Extension","version":"1.0","manifest_version":3}`)
	writeTestFile(t, filepath.Join(dir, "worker.js"), "console.log('ok')")
	output := filepath.Join(dir, "Extension-1.0.zip")

	if _, err := Pack(dir, output, config.PackageConfig{}); err != nil {
		t.Fatal(err)
	}
	if _, err := Pack(dir, output, config.PackageConfig{}); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	found, err := cwszip.ContainsFileInZip(data, filepath.Base(output))
	if err != nil {
		t.Fatal(err)
	}
	if found {
		t.Fatalf("repeated pack included stale output %q", filepath.Base(output))
	}
}

func TestWaitForUploadBoundsInflightRequest(t *testing.T) {
	requestStarted := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(requestStarted)
		<-r.Context().Done()
	}))
	defer server.Close()

	client := api.NewClient(staticAuthenticator{}, "publisher")
	client.BaseURL = server.URL
	started := time.Now()
	_, err := waitForUpload(context.Background(), client, "extension", 40*time.Millisecond, 5*time.Millisecond)
	if err == nil || !strings.Contains(err.Error(), "timed out waiting for upload processing") {
		t.Fatalf("waitForUpload error = %v, want timeout", err)
	}
	select {
	case <-requestStarted:
	default:
		t.Fatal("timeout elapsed before exercising an in-flight status request")
	}
	if elapsed := time.Since(started); elapsed > 250*time.Millisecond {
		t.Fatalf("waitForUpload exceeded its deadline: %s", elapsed)
	}
}

func TestUploadPreservesPublishWarnings(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case strings.HasSuffix(r.URL.Path, ":upload"):
			_ = json.NewEncoder(w).Encode(map[string]any{
				"itemId":      "extension",
				"uploadState": api.UploadStateSucceeded,
			})
		case strings.HasSuffix(r.URL.Path, ":publish"):
			_ = json.NewEncoder(w).Encode(map[string]any{
				"state": "PENDING_REVIEW",
				"warningInfo": map[string]any{
					"warnings": []map[string]string{{
						"reason":      "PERMISSION_WARNING",
						"description": "Broad host permissions.",
					}},
				},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client := api.NewClient(staticAuthenticator{}, "publisher")
	client.BaseURL = server.URL
	result, err := Upload(context.Background(), &Context{
		Config:      &config.Config{},
		Client:      client,
		ExtensionID: "extension",
	}, UploadOptions{
		SkipValidate: true,
		ZipData:      []byte("zip"),
		Publish:      true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.PublishWarnings) != 1 {
		t.Fatalf("publish warnings = %+v, want one", result.PublishWarnings)
	}
	if result.PublishWarnings[0].Reason != "PERMISSION_WARNING" {
		t.Fatalf("warning = %+v", result.PublishWarnings[0])
	}
}

func TestWaitForUploadHonorsParentCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := waitForUpload(ctx, nil, "extension", time.Second, time.Millisecond)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("waitForUpload error = %v, want context canceled", err)
	}
}

func writeTestFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

func hasFailedCheck(result *ValidateResult, message string) bool {
	for _, check := range result.Checks {
		if !check.Passed && check.Message == message {
			return true
		}
	}
	return false
}

func hasFailedCheckContaining(result *ValidateResult, text string) bool {
	for _, check := range result.Checks {
		if !check.Passed && strings.Contains(check.Message, text) {
			return true
		}
	}
	return false
}
