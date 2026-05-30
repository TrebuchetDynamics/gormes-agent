package gormescli

import (
	"github.com/spf13/cobra"

	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/gormescli/rootruntime"
)

type RootOptions = rootruntime.RootOptions
type CommandFactories = rootruntime.CommandFactories

var RootCommandOrder = rootruntime.RootCommandOrder

func NewRootCommand(opts RootOptions, factories CommandFactories) *cobra.Command {
	return rootruntime.NewRootCommand(opts, factories)
}
