package gormescli

import (
	"github.com/spf13/cobra"

	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/gormescli/fidelityruntime"
)

type FidelityCommandOptions = fidelityruntime.FidelityCommandOptions

func NewFidelityCommand(options FidelityCommandOptions) *cobra.Command {
	return fidelityruntime.NewFidelityCommand(options)
}
