package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/vaughnbosu/cws-cli/internal/config"
	"github.com/vaughnbosu/cws-cli/internal/manifest"
	"github.com/vaughnbosu/cws-cli/internal/output"
	cwszip "github.com/vaughnbosu/cws-cli/internal/zip"
)

const maxPackageSize = 2 * 1024 * 1024 * 1024 // 2 GB (Chrome Web Store limit)

var validateCmd = &cobra.Command{
	Use:   "validate [source]",
	Short: "Run pre-flight checks before uploading",
	Long: `Validate an extension package before uploading to the Chrome Web Store.

Checks manifest.json structure, version format, icons, package size, and
optionally verifies the version is higher than the published and any
submitted (in-review or draft) version.

Use --local to skip remote checks (no credentials needed).`,
	Args: cobra.MaximumNArgs(1),
	RunE: runValidate,
}

func init() {
	validateCmd.Flags().Bool("local", false, "Only run local checks (skip API calls)")
	rootCmd.AddCommand(validateCmd)
}

// ValidationResult tracks the outcome of a single check.
type ValidationResult struct {
	Passed  bool   `json:"passed"`
	Message string `json:"message"`
}

func runValidate(cmd *cobra.Command, args []string) error {
	localOnly, _ := cmd.Flags().GetBool("local")

	cfg, err := config.Load()
	if err != nil && !localOnly {
		return err
	}

	extName, _ := cmd.Flags().GetString("ext")
	var source string
	if len(args) > 0 {
		source = args[0]
	} else {
		source = config.ResolveSource("", extName, cfg)
	}

	results, _ := runValidationChecks(cmd, cfg, source, localOnly)
	return printResults(results)
}

// runValidationChecks runs all validation checks and returns the results along
// with the zip built during validation (nil if none was built), so callers like
// cws upload can reuse it instead of zipping twice.
func runValidationChecks(cmd *cobra.Command, cfg *config.Config, source string, localOnly bool) ([]ValidationResult, []byte) {
	var results []ValidationResult
	var pkg config.PackageConfig
	if cfg != nil {
		pkg = cfg.Package
	}

	// Check 1: Source exists
	absSource, err := filepath.Abs(source)
	if err != nil {
		return append(results, fail("Source path: %s", err)), nil
	}

	info, err := os.Stat(absSource)
	if err != nil {
		return append(results, fail("Source not found: %s", source)), nil
	}

	// Check 2: Parse manifest.json
	var m *manifest.Manifest
	var zipData []byte

	if info.IsDir() {
		manifestPath := filepath.Join(absSource, "manifest.json")
		if _, err := os.Stat(manifestPath); err != nil {
			return append(results, fail("manifest.json not found in %s", source)), nil
		}
		results = append(results, pass("manifest.json found"))

		m, err = manifest.Parse(manifestPath)
		if err != nil {
			return append(results, fail("manifest.json: %s", err)), nil
		}
		results = append(results, pass("manifest.json is valid JSON"))

		// Zip for size check (reused by cws upload)
		zipData, err = cwszip.ZipDirectoryWithOptions(absSource, zipOptions(pkg))
		if err != nil {
			return append(results, fail("Failed to zip directory: %s", err)), nil
		}
	} else {
		ext := strings.ToLower(filepath.Ext(absSource))
		if ext != ".zip" && ext != ".crx" {
			return append(results, fail("Source must be a directory, .zip, or .crx file")), nil
		}

		zipData, err = os.ReadFile(absSource)
		if err != nil {
			return append(results, fail("Failed to read %s: %s", source, err)), nil
		}

		if ext == ".crx" {
			zipData, err = cwszip.ExtractZipFromCRX(zipData)
			if err != nil {
				return append(results, fail("CRX: %s", err)), nil
			}
			results = append(results, pass("CRX header unwrapped"))
		}

		m, err = manifest.ParseFromZip(zipData)
		if err != nil {
			return append(results, fail("manifest.json: %s", err)), nil
		}
		results = append(results, pass("manifest.json found"))
		results = append(results, pass("manifest.json is valid JSON"))
	}

	// Check 3: Required fields
	if m != nil {
		missing := manifest.ValidateRequired(m)
		if len(missing) > 0 {
			results = append(results, fail("Missing required fields: %s", strings.Join(missing, ", ")))
		} else {
			results = append(results, pass("Required fields present (name, version, manifest_version)"))
		}

		// Check 4: Version format
		if m.Version != "" {
			if err := manifest.ValidateVersion(m.Version); err != nil {
				results = append(results, fail("Invalid version %q: %s", m.Version, err))
			} else {
				results = append(results, pass("Version format valid: %s", m.Version))
			}
		}

		// Check 5: Icon files referenced by the manifest exist in the package
		if len(m.Icons) > 0 {
			missingIcons := missingIconFiles(m, info.IsDir(), absSource, zipData)
			if len(missingIcons) > 0 {
				results = append(results, fail("Missing icon files: %s", strings.Join(missingIcons, ", ")))
			} else {
				results = append(results, pass("Icon files present (%d)", len(m.Icons)))
			}
		}
	}

	// Check 6: Package size
	if zipData != nil {
		sizeMB := float64(len(zipData)) / (1024 * 1024)
		if len(zipData) > maxPackageSize {
			results = append(results, fail("Package too large: %.1f MB (max 2 GB)", sizeMB))
		} else if sizeMB >= 1.0 {
			results = append(results, pass("Package size OK (%.1f MB)", sizeMB))
		} else {
			sizeKB := float64(len(zipData)) / 1024
			results = append(results, pass("Package size OK (%.0f KB)", sizeKB))
		}
	}

	// Remote checks
	if localOnly || m == nil {
		return results, zipData
	}

	if cfg == nil {
		return append(results, fail("No configuration found (skipping remote checks)")), zipData
	}
	if err := config.ValidateAuth(cfg); err != nil {
		return append(results, fail("Auth not configured: %s (use --local to skip remote checks)", err)), zipData
	}

	actx, err := newAPIContext(cmd)
	if err != nil {
		return append(results, fail("Extension ID: %s", err)), zipData
	}
	ctx := context.Background()

	resp, _, err := actx.client.FetchStatus(ctx, actx.extensionID)
	if err != nil {
		return append(results, fail("Failed to fetch status: %s", err)), zipData
	}

	// Check 7: Version higher than published AND any submitted (draft/in-review) revision.
	// The upload endpoint rejects versions not higher than the last uploaded revision,
	// so comparing against the published version alone is not enough.
	if m.Version != "" {
		revisions := []struct {
			label   string
			version string
		}{
			{"published", resp.PublishedItemRevisionStatus.Version()},
			{"submitted", resp.SubmittedItemRevisionStatus.Version()},
		}
		checked := false
		for _, rev := range revisions {
			if rev.version == "" {
				continue
			}
			checked = true
			higher, err := manifest.CompareVersions(m.Version, rev.version)
			if err != nil {
				results = append(results, fail("Version comparison: %s", err))
			} else if !higher {
				results = append(results, fail("Version %s is not higher than %s %s", m.Version, rev.label, rev.version))
			} else {
				results = append(results, pass("Version %s > %s %s", m.Version, rev.label, rev.version))
			}
		}
		if !checked {
			results = append(results, pass("No published version found (first upload)"))
		}
	}

	// Check 8: No pending review submission
	if resp.SubmittedItemRevisionStatus != nil && resp.SubmittedItemRevisionStatus.State == "PENDING_REVIEW" {
		state := FormatState(resp.SubmittedItemRevisionStatus.State)
		results = append(results, fail("Pending submission exists (%s). Use 'cws cancel' first, or wait for review", state))
	} else {
		results = append(results, pass("No pending review submission"))
	}

	// Check 9: Policy flags
	if resp.TakenDown {
		results = append(results, fail("Extension has been TAKEN DOWN for a policy violation. Check the developer dashboard"))
	}
	if resp.Warned {
		results = append(results, fail("Extension has a policy WARNING and may be taken down if not resolved. Check the developer dashboard"))
	}

	return results, zipData
}

// missingIconFiles returns manifest icon paths that don't exist in the package.
func missingIconFiles(m *manifest.Manifest, isDir bool, absSource string, zipData []byte) []string {
	var missing []string
	for _, iconPath := range m.Icons {
		clean := strings.TrimPrefix(iconPath, "/")
		if isDir {
			if _, err := os.Stat(filepath.Join(absSource, filepath.FromSlash(clean))); err != nil {
				missing = append(missing, iconPath)
			}
		} else if zipData != nil {
			found, err := cwszip.ContainsFileInZip(zipData, clean)
			if err != nil || !found {
				missing = append(missing, iconPath)
			}
		}
	}
	return missing
}

func pass(format string, args ...any) ValidationResult {
	return ValidationResult{Passed: true, Message: fmt.Sprintf(format, args...)}
}

func fail(format string, args ...any) ValidationResult {
	return ValidationResult{Passed: false, Message: fmt.Sprintf(format, args...)}
}

func printResults(results []ValidationResult) error {
	failures := 0
	for _, r := range results {
		if r.Passed {
			output.Info("  + %s", r.Message)
		} else {
			output.Info("  x %s", r.Message)
			failures++
		}
	}

	if output.JSONMode() {
		_ = output.EmitJSON(map[string]any{
			"passed":   failures == 0,
			"failures": failures,
			"checks":   results,
		})
	} else {
		fmt.Println()
	}

	if failures > 0 {
		return fmt.Errorf("validation failed: %d issue(s) found", failures)
	}
	output.Info("Validation passed!")
	return nil
}
