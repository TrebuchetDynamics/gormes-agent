package gateway

import (
	"github.com/spf13/cobra"

	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/gormescli/modules/gateway/termuxsupport"
)

func NewBootInstallCommand() *cobra.Command {
	return termuxsupport.NewBootInstallCommand()
}

func NewBootUninstallCommand() *cobra.Command {
	return termuxsupport.NewBootUninstallCommand()
}
