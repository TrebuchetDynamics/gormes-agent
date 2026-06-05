package gormescli

import (
	"github.com/spf13/cobra"

	memoryapp "github.com/TrebuchetDynamics/gormes-agent/internal/app/memory"
)

type MemoryBuildProvenance = memoryapp.BuildProvenance
type MemoryCommandStatusOptions = memoryapp.Options

type MemoryCommandRows struct {
	Setup RowBackedCommandSpec
	Off   RowBackedCommandSpec
	Reset RowBackedCommandSpec
}

type MemoryCommandOptions struct {
	Status             MemoryCommandStatusOptions
	Rows               MemoryCommandRows
	UnavailableCommand func(RowBackedCommandSpec) *cobra.Command
}

func NewMemoryCommand(opts MemoryCommandOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "memory",
		Short: "Inspect persisted memory and extractor state",
		Args:  cobra.NoArgs,
	}
	rows := memoryCommandRowsWithDefaults(opts.Rows)
	cmd.AddCommand(
		newMemoryStatusCommand(opts.Status),
		memoryUnavailableCommand(opts, rows.Setup),
		memoryUnavailableCommand(opts, rows.Off),
		memoryUnavailableCommand(opts, rows.Reset),
	)
	return cmd
}

func memoryCommandRowsWithDefaults(rows MemoryCommandRows) MemoryCommandRows {
	if rows.Setup.Use == "" {
		rows.Setup.Use = "setup"
	}
	if rows.Setup.Short == "" {
		rows.Setup.Short = "Configure Hermes-compatible memory"
	}
	if rows.Off.Use == "" {
		rows.Off.Use = "off"
	}
	if rows.Off.Short == "" {
		rows.Off.Short = "Disable Hermes-compatible memory"
	}
	if rows.Reset.Use == "" {
		rows.Reset.Use = "reset"
	}
	if rows.Reset.Short == "" {
		rows.Reset.Short = "Reset Hermes-compatible memory state"
	}
	rows.Reset.Destructive = true
	return rows
}

func newMemoryStatusCommand(opts MemoryCommandStatusOptions) *cobra.Command {
	var asJSON bool
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show extractor queue depth and dead letters",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return memoryapp.RunStatus(cmd.Context(), cmd.OutOrStdout(), asJSON, opts)
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "emit a `{build, inventory, extractor, goncho_queue}` JSON document (suitable for SRE alerting on memory backlog)")
	return cmd
}

func memoryUnavailableCommand(opts MemoryCommandOptions, spec RowBackedCommandSpec) *cobra.Command {
	if opts.UnavailableCommand != nil {
		return opts.UnavailableCommand(spec)
	}
	return NewRowBackedCommand(spec, RowBackedCommandOptions{})
}
