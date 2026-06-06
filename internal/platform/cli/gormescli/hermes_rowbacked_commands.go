package gormescli

import (
	hermesrowbacked "github.com/TrebuchetDynamics/gormes-agent/internal/app/hermesrowbacked"
	"github.com/spf13/cobra"
)

const (
	HermesGatewayCronRow = hermesrowbacked.GatewayCronRow
	HermesDiagnosticsRow = hermesrowbacked.DiagnosticsRow
	HermesConfigRow      = hermesrowbacked.ConfigRow
	HermesToolRow        = hermesrowbacked.ToolRow
	HermesSkillsRow      = hermesrowbacked.SkillsRow
	HermesMemoryRow      = hermesrowbacked.MemoryRow
	HermesKanbanRow      = hermesrowbacked.KanbanRow
)

// NewDumpCommand returns the Hermes-compatible debug-dump placeholder surface.
func NewDumpCommand(opts RowBackedCommandOptions) *cobra.Command {
	return NewRowBackedCommand(newHermesRowBackedSpec(hermesrowbacked.DumpSpec()), opts)
}

// NewDebugCommand returns the Hermes-compatible debug bundle placeholder tree.
func NewDebugCommand(opts RowBackedCommandOptions) *cobra.Command {
	return newHermesRowBackedParent(
		"debug",
		"Manage Hermes-compatible debug share bundles",
		NewRowBackedCommand(newHermesRowBackedSpec(hermesrowbacked.DebugShareSpec()), opts),
		NewRowBackedCommand(newHermesRowBackedSpec(hermesrowbacked.DebugDeleteSpec()), opts),
	)
}

// NewBackupCommand returns the Hermes-compatible backup placeholder surface.
func NewBackupCommand(opts RowBackedCommandOptions) *cobra.Command {
	return NewRowBackedCommand(newHermesRowBackedSpec(hermesrowbacked.BackupSpec()), opts)
}

// NewImportCommand returns the Hermes-compatible import placeholder surface.
func NewImportCommand(opts RowBackedCommandOptions) *cobra.Command {
	return NewRowBackedCommand(newHermesRowBackedSpec(hermesrowbacked.ImportSpec()), opts)
}

func newHermesRowBackedSpec(spec hermesrowbacked.Spec) RowBackedCommandSpec {
	return RowBackedCommandSpec{
		Use:         spec.Use,
		Short:       spec.Short,
		Row:         spec.Row,
		Destructive: spec.Destructive,
	}
}

func newHermesRowBackedParent(use, short string, children ...*cobra.Command) *cobra.Command {
	cmd := &cobra.Command{
		Use:   use,
		Short: short,
	}
	cmd.AddCommand(children...)
	return cmd
}
