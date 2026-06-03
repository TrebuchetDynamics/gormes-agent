package fidelityruntime

import (
	"github.com/spf13/cobra"

	appfidelity "github.com/TrebuchetDynamics/gormes-agent/internal/app/fidelity"
)

type FidelityCommandOptions = appfidelity.FidelityCommandOptions

func NewFidelityCommand(options FidelityCommandOptions) *cobra.Command {
	return appfidelity.NewFidelityCommand(options)
}
