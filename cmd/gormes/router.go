package main

import (
	providermodule "github.com/TrebuchetDynamics/gormes-agent/internal/cli/gormescli/modules/providers"
	"github.com/spf13/cobra"
)

func newRouterCommand() *cobra.Command {
	return providermodule.NewRouterCommand(providermodule.RouterCommandOptions{})
}
