package gormescli

import (
	"github.com/spf13/cobra"

	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/gormescli/commandruntime"
)

// RowBackedCommandSpec describes a CLI surface that is intentionally present
// but still backed by a future progress row.
type RowBackedCommandSpec = commandruntime.RowBackedCommandSpec

// RowBackedCommandOptions carries binary-owned build values into importable
// row-backed command modules.
type RowBackedCommandOptions = commandruntime.RowBackedCommandOptions

// RowBackedReportJSON is the machine-readable report emitted by row-backed
// compatibility commands.
type RowBackedReportJSON = commandruntime.RowBackedReportJSON

// NewRowBackedCommand creates a command that is visible in help and emits a
// structured unavailable report with exit code 2 when invoked.
func NewRowBackedCommand(spec RowBackedCommandSpec, opts RowBackedCommandOptions, children ...*cobra.Command) *cobra.Command {
	return commandruntime.NewRowBackedCommand(spec, opts, children...)
}

// RunRowBackedCommand emits row-backed evidence and returns exit code 2.
func RunRowBackedCommand(cmd *cobra.Command, spec RowBackedCommandSpec, opts RowBackedCommandOptions) error {
	return commandruntime.RunRowBackedCommand(cmd, spec, opts)
}
