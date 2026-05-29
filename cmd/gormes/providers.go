package main

import (
	"github.com/spf13/cobra"

	providermodule "github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/gormescli/modules/providers"
)

func newProvidersCommand() *cobra.Command {
	return providermodule.NewProvidersCommand(providerCommandOptions())
}
