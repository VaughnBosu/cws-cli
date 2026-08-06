package cmd

import (
	"github.com/spf13/cobra"
	"github.com/vaughnbosu/cws-cli/internal/output"
	"github.com/vaughnbosu/cws-cli/pkg/config"
	"github.com/vaughnbosu/cws-cli/pkg/service"
)

var packCmd = &cobra.Command{
	Use:   "pack [source]",
	Short: "Zip an extension directory without uploading",
	Long: `Create the zip package that cws upload would send, and write it to disk.

Useful for inspecting exactly what gets uploaded, or for uploading manually.
The output defaults to <name>-<version>.zip in the current directory.`,
	Args: cobra.MaximumNArgs(1),
	RunE: runPack,
}

func init() {
	packCmd.Flags().StringP("output", "o", "", "Output zip path (default <name>-<version>.zip)")
	rootCmd.AddCommand(packCmd)
}

func runPack(cmd *cobra.Command, args []string) error {
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

	outPath, _ := cmd.Flags().GetString("output")
	result, err := service.Pack(source, outPath, cfg.Package)
	if err != nil {
		return err
	}

	output.Info("Packed %s (%d bytes)", result.Path, result.Bytes)
	if output.JSONMode() {
		return output.EmitJSON(result)
	}
	return nil
}
