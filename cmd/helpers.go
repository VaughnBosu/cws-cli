package cmd

import (
	"github.com/spf13/cobra"
	"github.com/vaughnbosu/cws-cli/pkg/service"
)

func newAPIContext(cmd *cobra.Command) (*service.Context, error) {
	extensionIDFlag, _ := cmd.Flags().GetString("extension-id")
	extName, _ := cmd.Flags().GetString("ext")
	return service.NewContext(service.ContextOptions{
		ExtensionID: extensionIDFlag,
		Profile:     extName,
	})
}
