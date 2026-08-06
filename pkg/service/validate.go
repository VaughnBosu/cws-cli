package service

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/vaughnbosu/cws-cli/pkg/config"
	"github.com/vaughnbosu/cws-cli/pkg/manifest"
	cwszip "github.com/vaughnbosu/cws-cli/pkg/zip"
)

const maxPackageSize = 2 * 1024 * 1024 * 1024 // 2 GB

// ValidationResult tracks the outcome of a single check.
type ValidationResult struct {
	Passed  bool   `json:"passed"`
	Message string `json:"message"`
}

// ValidateResult is the aggregate outcome of all validation checks.
type ValidateResult struct {
	Passed   bool               `json:"passed"`
	Failures int                `json:"failures"`
	Checks   []ValidationResult `json:"checks"`
}

// ValidateOptions configures a validation run.
type ValidateOptions struct {
	Source    string
	LocalOnly bool
}

// Validate runs pre-flight checks on an extension package. When actx is nil and
// LocalOnly is false, remote checks are skipped with a failure entry. Returns
// zip bytes when a package was built (for upload reuse).
func Validate(ctx context.Context, actx *Context, opts ValidateOptions) (*ValidateResult, []byte, error) {
	var cfg *config.Config
	if actx != nil {
		cfg = actx.Config
	}

	var pkg config.PackageConfig
	if cfg != nil {
		pkg = cfg.Package
	}

	results, zipData := runValidationChecks(ctx, actx, cfg, opts.Source, opts.LocalOnly, pkg)
	failures := 0
	for _, r := range results {
		if !r.Passed {
			failures++
		}
	}
	return &ValidateResult{
		Passed:   failures == 0,
		Failures: failures,
		Checks:   results,
	}, zipData, nil
}

func runValidationChecks(ctx context.Context, actx *Context, cfg *config.Config, source string, localOnly bool, pkg config.PackageConfig) ([]ValidationResult, []byte) {
	var results []ValidationResult

	absSource, err := filepath.Abs(source)
	if err != nil {
		return append(results, validationFail("Source path: %s", err)), nil
	}

	info, err := os.Stat(absSource)
	if err != nil {
		return append(results, validationFail("Source not found: %s", source)), nil
	}

	var m *manifest.Manifest
	var zipData []byte

	if info.IsDir() {
		manifestPath := filepath.Join(absSource, "manifest.json")
		if _, err := os.Stat(manifestPath); err != nil {
			return append(results, validationFail("manifest.json not found in %s", source)), nil
		}
		results = append(results, validationPass("manifest.json found"))

		m, err = manifest.Parse(manifestPath)
		if err != nil {
			return append(results, validationFail("manifest.json: %s", err)), nil
		}
		results = append(results, validationPass("manifest.json is valid JSON"))

		zipData, err = cwszip.ZipDirectoryWithOptions(absSource, zipOptions(pkg))
		if err != nil {
			return append(results, validationFail("Failed to zip directory: %s", err)), nil
		}
	} else {
		ext := strings.ToLower(filepath.Ext(absSource))
		if ext != ".zip" && ext != ".crx" {
			return append(results, validationFail("Source must be a directory, .zip, or .crx file")), nil
		}

		zipData, err = os.ReadFile(absSource)
		if err != nil {
			return append(results, validationFail("Failed to read %s: %s", source, err)), nil
		}

		if ext == ".crx" {
			zipData, err = cwszip.ExtractZipFromCRX(zipData)
			if err != nil {
				return append(results, validationFail("CRX: %s", err)), nil
			}
			results = append(results, validationPass("CRX header unwrapped"))
		}

		m, err = manifest.ParseFromZip(zipData)
		if err != nil {
			return append(results, validationFail("manifest.json: %s", err)), nil
		}
		results = append(results, validationPass("manifest.json found"))
		results = append(results, validationPass("manifest.json is valid JSON"))
	}

	if m != nil {
		missing := manifest.ValidateRequired(m)
		if len(missing) > 0 {
			results = append(results, validationFail("Missing required fields: %s", strings.Join(missing, ", ")))
		} else {
			results = append(results, validationPass("Required fields present (name, version, manifest_version)"))
		}

		if m.Version != "" {
			if err := manifest.ValidateVersion(m.Version); err != nil {
				results = append(results, validationFail("Invalid version %q: %s", m.Version, err))
			} else {
				results = append(results, validationPass("Version format valid: %s", m.Version))
			}
		}

		if len(m.Icons) > 0 {
			missingIcons := missingIconFiles(m, info.IsDir(), absSource, zipData)
			if len(missingIcons) > 0 {
				results = append(results, validationFail("Missing icon files: %s", strings.Join(missingIcons, ", ")))
			} else {
				results = append(results, validationPass("Icon files present (%d)", len(m.Icons)))
			}
		}
	}

	if zipData != nil {
		sizeMB := float64(len(zipData)) / (1024 * 1024)
		if len(zipData) > maxPackageSize {
			results = append(results, validationFail("Package too large: %.1f MB (max 2 GB)", sizeMB))
		} else if sizeMB >= 1.0 {
			results = append(results, validationPass("Package size OK (%.1f MB)", sizeMB))
		} else {
			sizeKB := float64(len(zipData)) / 1024
			results = append(results, validationPass("Package size OK (%.0f KB)", sizeKB))
		}
	}

	if localOnly || m == nil {
		return results, zipData
	}

	if cfg == nil {
		return append(results, validationFail("No configuration found (skipping remote checks)")), zipData
	}
	if err := config.ValidateAuth(cfg); err != nil {
		return append(results, validationFail("Auth not configured: %s (use local_only to skip remote checks)", err)), zipData
	}
	if actx == nil {
		return append(results, validationFail("API context required for remote checks")), zipData
	}

	resp, _, err := actx.Client.FetchStatus(ctx, actx.ExtensionID)
	if err != nil {
		return append(results, validationFail("Failed to fetch status: %s", err)), zipData
	}

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
				results = append(results, validationFail("Version comparison: %s", err))
			} else if !higher {
				results = append(results, validationFail("Version %s is not higher than %s %s", m.Version, rev.label, rev.version))
			} else {
				results = append(results, validationPass("Version %s > %s %s", m.Version, rev.label, rev.version))
			}
		}
		if !checked {
			results = append(results, validationPass("No published version found (first upload)"))
		}
	}

	if resp.SubmittedItemRevisionStatus != nil && resp.SubmittedItemRevisionStatus.State == "PENDING_REVIEW" {
		state := FormatState(resp.SubmittedItemRevisionStatus.State)
		results = append(results, validationFail("Pending submission exists (%s). Cancel it first, or wait for review", state))
	} else {
		results = append(results, validationPass("No pending review submission"))
	}

	if resp.TakenDown {
		results = append(results, validationFail("Extension has been TAKEN DOWN for a policy violation. Check the developer dashboard"))
	}
	if resp.Warned {
		results = append(results, validationFail("Extension has a policy WARNING and may be taken down if not resolved. Check the developer dashboard"))
	}

	return results, zipData
}

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

func validationPass(format string, args ...any) ValidationResult {
	return ValidationResult{Passed: true, Message: fmt.Sprintf(format, args...)}
}

func validationFail(format string, args ...any) ValidationResult {
	return ValidationResult{Passed: false, Message: fmt.Sprintf(format, args...)}
}

func zipOptions(pkg config.PackageConfig) cwszip.Options {
	return cwszip.Options{
		Exclude: pkg.Exclude,
		Include: pkg.Include,
	}
}

// ValidationError indicates one or more checks failed.
type ValidationError struct {
	Result *ValidateResult
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("validation failed: %d issue(s) found", e.Result.Failures)
}

// ErrValidationFailed wraps a ValidateResult as an error.
func ErrValidationFailed(result *ValidateResult) error {
	return &ValidationError{Result: result}
}
