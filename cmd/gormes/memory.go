package main

import (
	"database/sql"

	goncho "github.com/TrebuchetDynamics/goncho/service"
	"github.com/spf13/cobra"

	memorycmd "github.com/TrebuchetDynamics/gormes-agent/cmd/gormes/memory"
)

// newMemoryCommand returns a fresh memory command tree (parent + status
// subcommand). Constructor pattern avoids cross-test FlagSet/state
// contamination on a shared package-level var.
func newMemoryCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "memory",
		Short: "Inspect persisted memory and extractor state",
		Args:  cobra.NoArgs,
	}
	cmd.AddCommand(
		newMemoryStatusCommand(),
		newHermesUnavailableCommand(hermesUnavailableCommandSpec{
			Use:   "setup",
			Short: "Configure Hermes-compatible memory",
			Row:   hermesMemoryRow,
		}),
		newHermesUnavailableCommand(hermesUnavailableCommandSpec{
			Use:   "off",
			Short: "Disable Hermes-compatible memory",
			Row:   hermesMemoryRow,
		}),
		newHermesUnavailableCommand(hermesUnavailableCommandSpec{
			Use:         "reset",
			Short:       "Reset Hermes-compatible memory state",
			Row:         hermesMemoryRow,
			Destructive: true,
			FlagSet:     hermesUnavailableYesFlag,
		}),
	)
	return cmd
}

func newMemoryStatusCommand() *cobra.Command {
	var asJSON bool
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show extractor queue depth and dead letters",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return memorycmd.RunStatus(cmd.Context(), cmd.OutOrStdout(), asJSON, memoryCommandOptions())
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "emit a `{build, inventory, extractor, goncho_queue}` JSON document (suitable for SRE alerting on memory backlog)")
	return cmd
}

func memoryCommandOptions() memorycmd.Options {
	return memorycmd.Options{
		BuildProvenance: memoryBuildProvenance,
		OpenDB:          memoryOpenDB,
	}
}

func memoryBuildProvenance() memorycmd.BuildProvenance {
	build := newBuildProvenance()
	return memorycmd.BuildProvenance{Version: build.Version, GitCommit: build.GitCommit}
}

func memoryOpenDB(path string) (*sql.DB, error) {
	return sqlOpenGoncho(path)
}

func formatDreamQueueEvidence(status goncho.DreamQueueStatus) string {
	return memorycmd.FormatDreamQueueEvidence(status)
}
