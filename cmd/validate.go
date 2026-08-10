package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/vaughnbosu/cws-cli/internal/output"
	"github.com/vaughnbosu/cws-cli/pkg/config"
	"github.com/vaughnbosu/cws-cli/pkg/service"
)

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

func runValidate(cmd *cobra.Command, args []string) error {
	localOnly, _ := cmd.Flags().GetBool("local")

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	extName, _ := cmd.Flags().GetString("ext")
	var source string
	if len(args) > 0 {
		source = args[0]
	} else {
		source = config.ResolveSource("", extName, cfg)
	}

	var actx *service.Context
	if !localOnly {
		actx, err = newAPIContext(cmd)
		if err != nil {
			return err
		}
	}

	result, _, err := service.Validate(context.Background(), actx, service.ValidateOptions{
		Source:    source,
		LocalOnly: localOnly,
	})
	if err != nil {
		return err
	}

	return printValidateResults(result)
}

func printValidateResults(result *service.ValidateResult) error {
	failures := 0
	for _, check := range result.Checks {
		if check.Passed {
			output.Info("  + %s", check.Message)
		} else {
			output.Info("  x %s", check.Message)
			failures++
		}
	}

	if output.JSONMode() {
		_ = output.EmitJSON(result)
	} else {
		fmt.Println()
	}

	if failures > 0 {
		return fmt.Errorf("validation failed: %d issue(s) found", failures)
	}
	output.Info("Validation passed!")
	return nil
}
