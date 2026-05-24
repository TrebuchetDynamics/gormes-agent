package main

import (
	"github.com/spf13/cobra"

	providermodule "github.com/TrebuchetDynamics/gormes-agent/internal/cli/gormescli/modules/providers"
)

func newInsightsCommand() *cobra.Command {
	return providermodule.NewInsightsCommand(providerCommandOptions())
}
