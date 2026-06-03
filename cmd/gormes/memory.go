package main

import (
	"database/sql"

	goncho "github.com/TrebuchetDynamics/goncho/service"
	"github.com/spf13/cobra"

	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/gormescli"
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
	return gormescli.NewMemoryStatusCommand(gormescli.MemoryOptions(memoryBuildProvenance, memoryOpenDB))
}

func memoryBuildProvenance() gormescli.BuildProvenance {
	build := newBuildProvenance()
	return gormescli.BuildProvenance{Version: build.Version, GitCommit: build.GitCommit}
}

func memoryOpenDB(path string) (*sql.DB, error) {
	return sqlOpenGoncho(path)
}

func formatDreamQueueEvidence(status goncho.DreamQueueStatus) string {
	return gormescli.FormatMemoryDreamQueueEvidence(status)
}
