package service

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/vaughnbosu/cws-cli/pkg/api"
	"github.com/vaughnbosu/cws-cli/pkg/config"
	cwszip "github.com/vaughnbosu/cws-cli/pkg/zip"
)

// UploadOptions configures an upload operation.
type UploadOptions struct {
	Source       string
	Wait         bool
	TimeoutSec   int
	Publish      bool
	SkipValidate bool
	ZipData      []byte // optional pre-built zip from Validate
}

// UploadResult is the machine-readable output of an upload.
type UploadResult struct {
	ItemID          string        `json:"itemId,omitempty"`
	CrxVersion      string        `json:"crxVersion,omitempty"`
	UploadState     string        `json:"uploadState"`
	Published       bool          `json:"published,omitempty"`
	PublishState    string        `json:"publishState,omitempty"`
	PublishWarnings []api.Warning `json:"publishWarnings,omitempty"`
}

// Upload validates (unless skipped), zips if needed, uploads, and optionally publishes.
func Upload(ctx context.Context, actx *Context, opts UploadOptions) (*UploadResult, error) {
	var zipData []byte
	if !opts.SkipValidate {
		result, validatedZip, err := Validate(ctx, actx, ValidateOptions{Source: opts.Source, LocalOnly: false})
		if err != nil {
			return nil, err
		}
		if !result.Passed {
			return nil, ErrValidationFailed(result)
		}
		zipData = validatedZip
	}

	if zipData == nil {
		if opts.ZipData != nil {
			zipData = opts.ZipData
		} else {
			var err error
			zipData, err = PrepareZip(opts.Source, actx.Config.Package)
			if err != nil {
				return nil, err
			}
		}
	}

	resp, err := actx.Client.Upload(ctx, actx.ExtensionID, zipData)
	if err != nil {
		return nil, err
	}

	if opts.Wait && api.IsUploadInProgress(resp.UploadState) {
		timeout := opts.TimeoutSec
		if timeout <= 0 {
			timeout = 300
		}
		resp, err = WaitForUpload(ctx, actx.Client, actx.ExtensionID, timeout)
		if err != nil {
			return nil, err
		}
	}

	if api.IsUploadFailed(resp.UploadState) {
		return nil, &api.CWSError{
			Operation: "upload",
			Message:   "upload processing failed. The v2 API does not return failure details",
			Hint:      "Check the item in the developer dashboard for the specific error: https://chrome.google.com/webstore/devconsole",
		}
	}

	if api.IsUploadInProgress(resp.UploadState) {
		return &UploadResult{
			ItemID:      resp.ItemID,
			CrxVersion:  resp.CrxVersion,
			UploadState: resp.UploadState,
		}, nil
	}

	if !api.IsUploadSucceeded(resp.UploadState) {
		return nil, fmt.Errorf("upload finished in unexpected state %q", resp.UploadState)
	}

	out := &UploadResult{
		ItemID:      resp.ItemID,
		CrxVersion:  resp.CrxVersion,
		UploadState: resp.UploadState,
	}

	if opts.Publish {
		pubResp, err := actx.Client.Publish(ctx, actx.ExtensionID, api.PublishOptions{})
		if err != nil {
			return nil, fmt.Errorf("upload succeeded but publish failed: %w", err)
		}
		out.Published = true
		out.PublishState = pubResp.State
		if pubResp.WarningInfo != nil {
			out.PublishWarnings = pubResp.WarningInfo.Warnings
		}
	}

	return out, nil
}

// PrepareZip turns a source (directory, .zip, or .crx) into uploadable zip bytes.
func PrepareZip(source string, pkg config.PackageConfig) ([]byte, error) {
	absSource, err := filepath.Abs(source)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve source path: %w", err)
	}

	info, err := os.Stat(absSource)
	if err != nil {
		return nil, fmt.Errorf("source not found: %s", source)
	}

	if !info.IsDir() {
		ext := strings.ToLower(filepath.Ext(absSource))
		if ext != ".zip" && ext != ".crx" {
			return nil, fmt.Errorf("source file must be a .zip or .crx file, got: %s", ext)
		}

		data, err := os.ReadFile(absSource)
		if err != nil {
			return nil, fmt.Errorf("failed to read source file: %w", err)
		}

		if ext == ".crx" {
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

	if !cwszip.ContainsManifest(absSource) {
		return nil, fmt.Errorf("directory does not contain a manifest.json: %s", source)
	}

	data, err := cwszip.ZipDirectoryWithOptions(absSource, zipOptions(pkg))
	if err != nil {
		return nil, err
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

// WaitForUpload polls until upload processing completes or timeout is reached.
func WaitForUpload(ctx context.Context, client *api.Client, extensionID string, timeoutSec int) (*api.UploadResponse, error) {
	return waitForUpload(ctx, client, extensionID, time.Duration(timeoutSec)*time.Second, 5*time.Second)
}

func waitForUpload(ctx context.Context, client *api.Client, extensionID string, timeout, pollInterval time.Duration) (*api.UploadResponse, error) {
	waitCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()
	timeoutError := func() error {
		if timeout%time.Second == 0 {
			return fmt.Errorf("timed out waiting for upload processing after %d seconds", int(timeout/time.Second))
		}
		return fmt.Errorf("timed out waiting for upload processing after %s", timeout)
	}

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-waitCtx.Done():
			if ctx.Err() != nil {
				return nil, ctx.Err()
			}
			return nil, timeoutError()
		case <-ticker.C:
		}

		status, _, err := client.FetchStatus(waitCtx, extensionID)
		if err != nil {
			if ctx.Err() != nil {
				return nil, ctx.Err()
			}
			if waitCtx.Err() != nil {
				return nil, timeoutError()
			}
			return nil, err
		}

		if !api.IsUploadInProgress(status.LastAsyncUploadState) {
			return &api.UploadResponse{
				ItemID:      status.ItemID,
				CrxVersion:  status.SubmittedItemRevisionStatus.Version(),
				UploadState: status.LastAsyncUploadState,
			}, nil
		}
	}
}
