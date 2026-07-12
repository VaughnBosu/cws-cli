package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/vaughnbosu/cws-cli/internal/config"
	"github.com/vaughnbosu/cws-cli/internal/manifest"
	"github.com/vaughnbosu/cws-cli/internal/output"
	cwszip "github.com/vaughnbosu/cws-cli/internal/zip"
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

	absSource, err := filepath.Abs(source)
	if err != nil {
		return fmt.Errorf("failed to resolve source path: %w", err)
	}
	info, err := os.Stat(absSource)
	if err != nil {
		return fmt.Errorf("source not found: %s", source)
	}
	if !info.IsDir() {
		return fmt.Errorf("pack requires a directory source, got a file: %s", source)
	}

	manifestPath := filepath.Join(absSource, "manifest.json")
	m, err := manifest.Parse(manifestPath)
	if err != nil {
		return err
	}

	data, err := cwszip.ZipDirectoryWithOptions(absSource, zipOptions(cfg.Package))
	if err != nil {
		return err
	}

	outPath, _ := cmd.Flags().GetString("output")
	if outPath == "" {
		name := sanitizeFilename(m.Name)
		if name == "" {
			name = "extension"
		}
		outPath = fmt.Sprintf("%s-%s.zip", name, m.Version)
	}

	if err := os.WriteFile(outPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write %s: %w", outPath, err)
	}

	output.Info("Packed %s (%d bytes)", outPath, len(data))
	if output.JSONMode() {
		return output.EmitJSON(map[string]any{
			"path":    outPath,
			"bytes":   len(data),
			"name":    m.Name,
			"version": m.Version,
		})
	}
	return nil
}

// sanitizeFilename converts an extension name to a safe file name fragment.
func sanitizeFilename(name string) string {
	var b []rune
	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '-', r == '_':
			b = append(b, r)
		case r == ' ':
			b = append(b, '-')
		}
	}
	return string(b)
}
