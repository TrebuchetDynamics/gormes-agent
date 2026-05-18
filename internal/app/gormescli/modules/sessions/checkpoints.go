package sessions

import (
	"fmt"

	"github.com/spf13/cobra"
)

type CheckpointsCommandSeams struct {
	RunStatus      func(*cobra.Command, []string) error
	RunPrune       func(*cobra.Command, []string) error
	RunClear       func(*cobra.Command, []string) error
	RunClearLegacy func(*cobra.Command, []string) error
}

func NewCheckpointsCommandWithSeams(seams CheckpointsCommandSeams) *cobra.Command {
	seams = seams.withDefaults()
	cmd := &cobra.Command{
		Use:   "checkpoints",
		Short: "Inspect and manage Gormes file-operation rollback state",
		Long: `View and control the filesystem checkpoint store at $XDG_DATA_HOME/gormes/checkpoints/.

Commands mirror Hermes' hermes checkpoints CLI. None require the agent to be
running — safe to call at any time.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				if cmd.SuggestionsMinimumDistance <= 0 {
					cmd.SuggestionsMinimumDistance = 2
				}
				if suggestions := cmd.SuggestionsFor(args[0]); len(suggestions) > 0 {
					return fmt.Errorf("unknown command %q for %q; did you mean %q?", args[0], cmd.CommandPath(), suggestions[0])
				}
				return fmt.Errorf("unknown command %q for %q", args[0], cmd.CommandPath())
			}
			return seams.RunStatus(cmd, args)
		},
	}
	cmd.AddCommand(
		newCheckpointsStatusCommand(seams),
		newCheckpointsListCommand(seams),
		newCheckpointsPruneCommand(seams),
		newCheckpointsClearCommand(seams),
		newCheckpointsClearLegacyCommand(seams),
	)
	return cmd
}

func (s CheckpointsCommandSeams) withDefaults() CheckpointsCommandSeams {
	if s.RunStatus == nil {
		s.RunStatus = missingCheckpointsSeam("status")
	}
	if s.RunPrune == nil {
		s.RunPrune = missingCheckpointsSeam("prune")
	}
	if s.RunClear == nil {
		s.RunClear = missingCheckpointsSeam("clear")
	}
	if s.RunClearLegacy == nil {
		s.RunClearLegacy = missingCheckpointsSeam("clear-legacy")
	}
	return s
}

func missingCheckpointsSeam(name string) func(*cobra.Command, []string) error {
	return func(*cobra.Command, []string) error {
		return fmt.Errorf("checkpoints %s seam is not configured", name)
	}
}

func newCheckpointsStatusCommand(seams CheckpointsCommandSeams) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show total size, project count, and per-project breakdown",
		RunE:  seams.RunStatus,
	}
	cmd.Flags().Int("limit", 20, "max projects to list")
	cmd.Flags().Bool("json", false, "emit machine-readable JSON: `{build, root, total_size_bytes, …, projects: [...], legacy_archives: [...]}`")
	return cmd
}

func newCheckpointsListCommand(seams CheckpointsCommandSeams) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "Alias for 'status'",
		RunE:  seams.RunStatus,
	}
	cmd.Flags().Int("limit", 20, "max projects to list")
	cmd.Flags().Bool("json", false, "emit machine-readable JSON (see `status --json`)")
	return cmd
}

func newCheckpointsPruneCommand(seams CheckpointsCommandSeams) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "prune",
		Short: "Delete orphan/stale checkpoints and GC the store",
		RunE:  seams.RunPrune,
	}
	cmd.Flags().Int("retention-days", 7, "drop projects whose last_touch is older than N days")
	cmd.Flags().Int("max-size-mb", 500, "after orphan/stale prune, drop oldest commits per project until total size <= this")
	cmd.Flags().Bool("keep-orphans", false, "skip deleting projects whose workdir no longer exists")
	cmd.Flags().Bool("dry-run", false, "preview which orphan/stale shadows would be deleted without touching disk")
	cmd.Flags().Bool("json", false, "emit machine-readable JSON: `{build, dry_run, retention_days, max_total_size_mb, keep_orphans, scanned, deleted_orphan, deleted_stale, errors, bytes_freed}`")
	return cmd
}

func newCheckpointsClearCommand(seams CheckpointsCommandSeams) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "clear",
		Short: "Delete the entire checkpoint base (all /rollback history)",
		RunE:  seams.RunClear,
	}
	cmd.Flags().BoolP("force", "f", false, "skip confirmation prompt")
	cmd.Flags().Bool("json", false, "emit machine-readable JSON: `{build, root, action, deleted, bytes_freed, projects_before, legacy_before}`")
	return cmd
}

func newCheckpointsClearLegacyCommand(seams CheckpointsCommandSeams) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "clear-legacy",
		Short: "Delete only the legacy-<ts>/ archives from v1 migration",
		RunE:  seams.RunClearLegacy,
	}
	cmd.Flags().BoolP("force", "f", false, "skip confirmation prompt")
	cmd.Flags().Bool("json", false, "emit machine-readable JSON: `{build, root, action, archives_before: [...], deleted, bytes_freed}`")
	return cmd
}
