package cmd

import (
	"fmt"
	"runtime/debug"
	"strings"

	"github.com/spf13/cobra"
)

// Version is set at build time via ldflags.
var Version = "dev"

// versionCmd is the version subcommand.
var versionCmd = &cobra.Command{
	Use:     "version",
	Aliases: []string{"v"},
	Short:   "Print the CLI version",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("cws %s\n", resolvedVersion())
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
	rootCmd.Version = resolvedVersion()
	rootCmd.SetVersionTemplate("cws {{.Version}}\n")
}

func resolvedVersion() string {
	if info, ok := debug.ReadBuildInfo(); ok {
		return resolveVersion(Version, info.Main.Version)
	}
	return resolveVersion(Version, "")
}

func resolveVersion(linked, module string) string {
	linked = normalizeVersion(linked)
	if linked != "" && linked != "dev" {
		return linked
	}
	module = normalizeVersion(module)
	if module != "" && module != "(devel)" {
		return module
	}
	return "dev"
}

func normalizeVersion(version string) string {
	return strings.TrimPrefix(strings.TrimSpace(version), "v")
}
