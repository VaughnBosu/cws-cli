package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/vaughnbosu/cws-cli/internal/api"
	"github.com/vaughnbosu/cws-cli/internal/config"
	"github.com/vaughnbosu/cws-cli/internal/output"
	cwszip "github.com/vaughnbosu/cws-cli/internal/zip"
)

var uploadCmd = &cobra.Command{
	Use:   "upload [source]",
	Short: "Upload a package to the Chrome Web Store",
	Long: `Zip (if needed) and upload a package to the Chrome Web Store.

Source can be a .zip file, .crx file, or a directory. If a directory
is given, it will be zipped automatically. CRX files are unwrapped to
their embedded zip before uploading. Defaults to the current directory
if not specified.`,
	Args: cobra.MaximumNArgs(1),
	RunE: runUpload,
}

func init() {
	uploadCmd.Flags().Bool("wait", true, "Wait for upload processing to complete")
	uploadCmd.Flags().Int("timeout", 300, "Max seconds to wait for upload processing")
	uploadCmd.Flags().Bool("publish", false, "Automatically publish after successful upload")
	uploadCmd.Flags().Bool("skip-validate", false, "Skip pre-upload validation checks")
	rootCmd.AddCommand(uploadCmd)
}

// uploadResult is the machine-readable output of cws upload.
type uploadResult struct {
	ItemID       string `json:"itemId,omitempty"`
	CrxVersion   string `json:"crxVersion,omitempty"`
	UploadState  string `json:"uploadState"`
	Published    bool   `json:"published,omitempty"`
	PublishState string `json:"publishState,omitempty"`
}

func runUpload(cmd *cobra.Command, args []string) error {
	actx, err := newAPIContext(cmd)
	if err != nil {
		return err
	}

	wait, _ := cmd.Flags().GetBool("wait")
	timeout, _ := cmd.Flags().GetInt("timeout")
	publish, _ := cmd.Flags().GetBool("publish")
	skipValidate, _ := cmd.Flags().GetBool("skip-validate")

	var source string
	if len(args) > 0 {
		source = args[0]
	} else {
		source = config.ResolveSource("", actx.extName, actx.cfg)
	}

	// Pre-upload validation (builds the zip once; reused for upload)
	var zipData []byte
	if !skipValidate {
		output.Info("Validating...")
		results, validatedZip := runValidationChecks(cmd, actx.cfg, source, false)
		failures := 0
		for _, r := range results {
			if !r.Passed {
				output.Info("  x %s", r.Message)
				failures++
			}
		}
		if failures > 0 {
			output.Hint("To upload without validation, use --skip-validate")
			return fmt.Errorf("cws validate failed: %d issue(s) found", failures)
		}
		output.Info("Validation passed!")
		output.Info("")
		zipData = validatedZip
	}

	if zipData == nil {
		zipData, err = prepareZip(source, actx.cfg.Package)
		if err != nil {
			return err
		}
	}

	ctx := context.Background()

	// Upload
	output.Info("Uploading to extension %s...", actx.extensionID)
	resp, err := actx.client.Upload(ctx, actx.extensionID, zipData)
	if err != nil {
		return err
	}

	output.Info("Upload state: %s", resp.UploadState)

	// Wait for processing
	if wait && api.IsUploadInProgress(resp.UploadState) {
		resp, err = waitForUpload(ctx, actx.client, actx.extensionID, timeout)
		if err != nil {
			return err
		}
	}

	if api.IsUploadFailed(resp.UploadState) {
		return &api.CWSError{
			Operation: "upload",
			Message:   "upload processing failed. The v2 API does not return failure details",
			Hint:      "Check the item in the developer dashboard for the specific error: https://chrome.google.com/webstore/devconsole",
		}
	}

	if api.IsUploadInProgress(resp.UploadState) {
		output.Info("Upload is still processing. Use 'cws status' to check progress.")
		return emitUploadJSON(resp, false, "")
	}

	if !api.IsUploadSucceeded(resp.UploadState) {
		return fmt.Errorf("upload finished in unexpected state %q. Run 'cws status' to check the item, or upload again", resp.UploadState)
	}

	if resp.CrxVersion != "" {
		output.Info("Upload successful! (version %s)", resp.CrxVersion)
	} else {
		output.Info("Upload successful!")
	}

	// Auto-publish
	publishState := ""
	if publish {
		output.Info("Publishing...")
		pubResp, err := actx.client.Publish(ctx, actx.extensionID, api.PublishOptions{})
		if err != nil {
			return fmt.Errorf("upload succeeded but publish failed: %w", err)
		}
		printPublishWarnings(pubResp)
		publishState = pubResp.State
		if pubResp.State != "" {
			output.Info("Publish state: %s", FormatState(pubResp.State))
		} else {
			output.Info("Publish submitted successfully.")
		}
	}

	return emitUploadJSON(resp, publish, publishState)
}

func emitUploadJSON(resp *api.UploadResponse, published bool, publishState string) error {
	if !output.JSONMode() {
		return nil
	}
	return output.EmitJSON(uploadResult{
		ItemID:       resp.ItemID,
		CrxVersion:   resp.CrxVersion,
		UploadState:  resp.UploadState,
		Published:    published,
		PublishState: publishState,
	})
}

// prepareZip turns a source (directory, .zip, or .crx) into uploadable zip bytes.
func prepareZip(source string, pkg config.PackageConfig) ([]byte, error) {
	absSource, err := filepath.Abs(source)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve source path: %w", err)
	}

	info, err := os.Stat(absSource)
	if err != nil {
		return nil, fmt.Errorf("source not found: %s", source)
	}

	if !info.IsDir() {
		// It's a file — read it directly (.zip or .crx)
		ext := strings.ToLower(filepath.Ext(absSource))
		if ext != ".zip" && ext != ".crx" {
			return nil, fmt.Errorf("source file must be a .zip or .crx file, got: %s", ext)
		}

		data, err := os.ReadFile(absSource)
		if err != nil {
			return nil, fmt.Errorf("failed to read source file: %w", err)
		}

		if ext == ".crx" {
			// The upload endpoint only accepts raw zips; unwrap the CRX header.
			data, err = cwszip.ExtractZipFromCRX(data)
			if err != nil {
				return nil, fmt.Errorf("failed to extract zip from CRX: %w", err)
			}
		}

		hasManifest, err := cwszip.ContainsManifestInZip(data)
		if err != nil {
			return nil, err
		}
		if !hasManifest {
			return nil, fmt.Errorf("package does not contain a manifest.json")
		}

		return data, nil
	}

	// Directory — validate and zip
	if !cwszip.ContainsManifest(absSource) {
		return nil, fmt.Errorf("directory does not contain a manifest.json: %s", source)
	}

	output.Info("Zipping directory %s...", source)
	data, err := cwszip.ZipDirectoryWithOptions(absSource, zipOptions(pkg))
	if err != nil {
		return nil, err
	}
	output.Info("Created zip (%d bytes)", len(data))

	return data, nil
}

// zipOptions converts config package settings to zip options.
func zipOptions(pkg config.PackageConfig) cwszip.Options {
	return cwszip.Options{
		Exclude: pkg.Exclude,
		Include: pkg.Include,
	}
}

func waitForUpload(ctx context.Context, client *api.Client, extensionID string, timeoutSec int) (*api.UploadResponse, error) {
	deadline := time.Now().Add(time.Duration(timeoutSec) * time.Second)
	pollInterval := 5 * time.Second

	output.Progress("Waiting for upload to process")

	for time.Now().Before(deadline) {
		time.Sleep(pollInterval)
		output.Progress(".")

		status, _, err := client.FetchStatus(ctx, extensionID)
		if err != nil {
			return nil, err
		}

		if !api.IsUploadInProgress(status.LastAsyncUploadState) {
			output.Progress("\n")
			return &api.UploadResponse{
				ItemID:      status.ItemID,
				CrxVersion:  status.SubmittedItemRevisionStatus.Version(),
				UploadState: status.LastAsyncUploadState,
			}, nil
		}
	}

	output.Progress("\n")
	return nil, fmt.Errorf("timed out waiting for upload processing after %d seconds", timeoutSec)
}
