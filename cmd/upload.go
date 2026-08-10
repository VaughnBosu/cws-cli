package cmd

import (
	"context"
	"errors"

	"github.com/spf13/cobra"
	"github.com/vaughnbosu/cws-cli/internal/output"
	"github.com/vaughnbosu/cws-cli/pkg/api"
	"github.com/vaughnbosu/cws-cli/pkg/config"
	"github.com/vaughnbosu/cws-cli/pkg/service"
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
		source = config.ResolveSource("", actx.Profile, actx.Config)
	}

	if !skipValidate {
		output.Info("Validating package before upload...")
	} else {
		output.Info("Uploading to extension %s...", actx.ExtensionID)
	}

	result, err := service.Upload(context.Background(), actx, service.UploadOptions{
		Source:       source,
		Wait:         wait,
		TimeoutSec:   timeout,
		Publish:      publish,
		SkipValidate: skipValidate,
	})
	if err != nil {
		var valErr *service.ValidationError
		if errors.As(err, &valErr) {
			for _, check := range valErr.Result.Checks {
				if !check.Passed {
					output.Info("  x %s", check.Message)
				}
			}
			output.Hint("To upload without validation, use --skip-validate")
		}
		return err
	}

	if !skipValidate {
		output.Info("Validation passed!")
		output.Info("")
	}

	output.Info("Upload state: %s", result.UploadState)

	if api.IsUploadInProgress(result.UploadState) {
		output.Info("Upload is still processing. Use 'cws status' to check progress.")
		if publish {
			output.Info("Automatic publish did not start. Run 'cws publish' after processing succeeds.")
		}
		return emitUploadJSON(result)
	}

	if result.CrxVersion != "" {
		output.Info("Upload successful! (version %s)", result.CrxVersion)
	} else {
		output.Info("Upload successful!")
	}

	if publish && result.PublishState != "" {
		output.Info("Publish state: %s", service.FormatState(result.PublishState))
	} else if publish {
		output.Info("Publish submitted successfully.")
	}
	printWarnings(result.PublishWarnings)

	return emitUploadJSON(result)
}

func emitUploadJSON(result *service.UploadResult) error {
	if !output.JSONMode() {
		return nil
	}
	return output.EmitJSON(result)
}
