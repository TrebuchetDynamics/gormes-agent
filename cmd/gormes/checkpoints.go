package main

import (
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/TrebuchetDynamics/gormes-agent/internal/tools"
)

func newCheckpointsCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "checkpoints",
		Short: "Inspect and manage Gormes file-operation rollback state",
		Long: `View and control the filesystem checkpoint store at $XDG_DATA_HOME/gormes/checkpoints/.

Commands mirror Hermes' hermes checkpoints CLI. None require the agent to be
running — safe to call at any time.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCheckpointsStatus(cmd, args)
		},
	}

	cmd.AddCommand(
		newCheckpointsStatusCommand(),
		newCheckpointsListCommand(),
		newCheckpointsPruneCommand(),
		newCheckpointsClearCommand(),
		newCheckpointsClearLegacyCommand(),
	)
	return cmd
}

func newCheckpointsStatusCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show total size, project count, and per-project breakdown",
		RunE:  runCheckpointsStatus,
	}
	cmd.Flags().Int("limit", 20, "max projects to list")
	return cmd
}

func newCheckpointsListCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "Alias for 'status'",
		RunE:  runCheckpointsStatus,
	}
	cmd.Flags().Int("limit", 20, "max projects to list")
	return cmd
}

func runCheckpointsStatus(cmd *cobra.Command, _ []string) error {
	root := tools.DefaultCheckpointRoot()
	out := cmd.OutOrStdout()
	result, err := tools.StoreStatus(root)
	if err != nil {
		return fmt.Errorf("checkpoints: %w", err)
	}

	if result.ProjectCount == 0 && result.LegacySizeBytes == 0 {
		if _, err := os.Stat(root); os.IsNotExist(err) {
			fmt.Fprintf(out, "No checkpoint store at %s\n", root)
			return nil
		}
	}

	fmt.Fprintf(out, "Checkpoint base: %s\n", root)
	fmt.Fprintf(out, "Total size:      %s\n", formatBytes(result.TotalSizeBytes))
	fmt.Fprintf(out, "  store/         %s\n", formatBytes(result.StoreSizeBytes))
	fmt.Fprintf(out, "  legacy-*       %s\n", formatBytes(result.LegacySizeBytes))
	fmt.Fprintf(out, "Projects:        %d\n", result.ProjectCount)

	limit, _ := cmd.Flags().GetInt("limit")
	projects := result.Projects
	if len(projects) > limit {
		projects = projects[:limit]
	}
	if len(projects) > 0 {
		fmt.Fprintln(out)
		fmt.Fprintf(out, "  %-60s  %7s  %12s  %s\n", "WORKDIR", "COMMITS", "LAST TOUCH", "STATE")
		for _, p := range projects {
			wd := p.Workdir
			if wd == "" {
				wd = "(unknown)"
			}
			if len(wd) > 60 {
				wd = "…" + wd[len(wd)-59:]
			}
			state := "orphan"
			if p.Exists {
				state = "live"
			}
			fmt.Fprintf(out, "  %-60s  %7d  %12s  %s\n",
				wd, p.Commits, formatAge(ago(time.Now(), p.LastTouch)), state)
		}
	}

	if len(result.LegacyArchives) > 0 {
		fmt.Fprintln(out)
		fmt.Fprintf(out, "Legacy archives (%d):\n", len(result.LegacyArchives))
		for _, a := range result.LegacyArchives {
			fmt.Fprintf(out, "  %-40s  %10s\n", a.Name, formatBytes(a.SizeBytes))
		}
		fmt.Fprintln(out)
		fmt.Fprintln(out, "Clear with: gormes checkpoints clear-legacy")
	}
	return nil
}

func newCheckpointsPruneCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "prune",
		Short: "Delete orphan/stale checkpoints and GC the store",
		RunE:  runCheckpointsPrune,
	}
	cmd.Flags().Int("retention-days", 7, "drop projects whose last_touch is older than N days")
	cmd.Flags().Int("max-size-mb", 500, "after orphan/stale prune, drop oldest commits per project until total size <= this")
	cmd.Flags().Bool("keep-orphans", false, "skip deleting projects whose workdir no longer exists")
	return cmd
}

func runCheckpointsPrune(cmd *cobra.Command, _ []string) error {
	out := cmd.OutOrStdout()
	retentionDays, _ := cmd.Flags().GetInt("retention-days")
	maxSizeMB, _ := cmd.Flags().GetInt("max-size-mb")
	keepOrphans, _ := cmd.Flags().GetBool("keep-orphans")

	fmt.Fprintln(out, "Pruning checkpoint store…")
	fmt.Fprintf(out, "  retention_days:    %d\n", retentionDays)
	fmt.Fprintf(out, "  delete_orphans:    %v\n", !keepOrphans)
	fmt.Fprintf(out, "  max_total_size_mb: %d\n", maxSizeMB)
	fmt.Fprintln(out)

	result := tools.PruneCheckpoints(tools.DefaultCheckpointRoot(), retentionDays, keepOrphans, maxSizeMB, time.Now)
	fmt.Fprintf(out, "Scanned:         %d\n", result.Scanned)
	fmt.Fprintf(out, "Deleted orphan:  %d\n", result.DeletedOrphan)
	fmt.Fprintf(out, "Deleted stale:   %d\n", result.DeletedStale)
	fmt.Fprintf(out, "Errors:          %d\n", result.Errors)
	fmt.Fprintf(out, "Bytes reclaimed: %s\n", formatBytes(result.BytesFreed))
	return nil
}

func newCheckpointsClearCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "clear",
		Short: "Delete the entire checkpoint base (all /rollback history)",
		RunE:  runCheckpointsClear,
	}
	cmd.Flags().BoolP("force", "f", false, "skip confirmation prompt")
	return cmd
}

func runCheckpointsClear(cmd *cobra.Command, _ []string) error {
	out := cmd.OutOrStdout()
	force, _ := cmd.Flags().GetBool("force")
	root := tools.DefaultCheckpointRoot()

	status, _ := tools.StoreStatus(root)
	if status.TotalSizeBytes == 0 {
		if !statCheckpointRoot(root) {
			fmt.Fprintln(out, "Nothing to clear — checkpoint base does not exist.")
			return nil
		}
	}

	fmt.Fprintf(out, "This will delete the ENTIRE checkpoint base at %s\n", status.Root)
	fmt.Fprintf(out, "  size:        %s\n", formatBytes(status.TotalSizeBytes))
	fmt.Fprintf(out, "  projects:    %d\n", status.ProjectCount)
	fmt.Fprintf(out, "  legacy dirs: %d\n", len(status.LegacyArchives))
	fmt.Fprintln(out)
	fmt.Fprintln(out, "All /rollback history for every working directory will be lost.")

	if !force {
		if !confirm(out, cmd.InOrStdin(), "Proceed?") {
			fmt.Fprintln(out, "Aborted.")
			return nil
		}
	}

	result := tools.ClearAll(root)
	if result.Deleted {
		fmt.Fprintf(out, "Cleared. Reclaimed %s.\n", formatBytes(result.BytesFreed))
		return nil
	}
	fmt.Fprintln(out, "Could not clear checkpoint base (see logs).")
	return nil
}

func newCheckpointsClearLegacyCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "clear-legacy",
		Short: "Delete only the legacy-<ts>/ archives from v1 migration",
		RunE:  runCheckpointsClearLegacy,
	}
	cmd.Flags().BoolP("force", "f", false, "skip confirmation prompt")
	return cmd
}

func runCheckpointsClearLegacy(cmd *cobra.Command, _ []string) error {
	out := cmd.OutOrStdout()
	force, _ := cmd.Flags().GetBool("force")
	root := tools.DefaultCheckpointRoot()

	status, _ := tools.StoreStatus(root)
	legacy := status.LegacyArchives
	if len(legacy) == 0 {
		fmt.Fprintln(out, "No legacy archives to clear.")
		return nil
	}

	var total int64
	for _, a := range legacy {
		total += a.SizeBytes
	}
	fmt.Fprintf(out, "Found %d legacy archive(s), total %s:\n", len(legacy), formatBytes(total))
	for _, a := range legacy {
		fmt.Fprintf(out, "  %-40s  %10s\n", a.Name, formatBytes(a.SizeBytes))
	}
	fmt.Fprintln(out)
	fmt.Fprintln(out, "Legacy archives hold pre-v2 per-project shadow repos, moved aside")
	fmt.Fprintln(out, "during the single-store migration. Delete when you're confident")
	fmt.Fprintln(out, "you don't need the old /rollback history.")

	if !force {
		if !confirm(out, cmd.InOrStdin(), "Delete all legacy archives?") {
			fmt.Fprintln(out, "Aborted.")
			return nil
		}
	}

	result := tools.ClearLegacy(root)
	fmt.Fprintf(out, "Deleted %d archive(s), reclaimed %s.\n", countLegacyDeleted(status, root), formatBytes(result.BytesFreed))
	return nil
}

func countLegacyDeleted(before tools.StoreStatusResult, root string) int {
	after, _ := tools.StoreStatus(root)
	return len(before.LegacyArchives) - len(after.LegacyArchives)
}

func confirm(out io.Writer, in io.Reader, prompt string) bool {
	fmt.Fprintf(out, "%s [y/N]: ", prompt)
	var response string
	_, _ = fmt.Fscanln(in, &response)
	response = strings.TrimSpace(strings.ToLower(response))
	return response == "y" || response == "yes"
}

func statCheckpointRoot(root string) bool {
	_, err := os.Stat(root)
	return err == nil
}

// formatBytes mirrors Hermes' _fmt_bytes helper.
func formatBytes(n int64) string {
	units := []string{"B", "KB", "MB", "GB", "TB"}
	size := float64(n)
	if n < 0 {
		size = 0
	}
	for _, unit := range units {
		if size < 1024 || unit == units[len(units)-1] {
			if unit == "B" {
				return fmt.Sprintf("%d B", int64(size))
			}
			return fmt.Sprintf("%.1f %s", size, unit)
		}
		size /= 1024
	}
	return fmt.Sprintf("%.1f TB", size)
}

func ago(now, t time.Time) time.Duration {
	if t.IsZero() {
		return -1
	}
	return now.Sub(t)
}

// formatAge mirrors Hermes' _fmt_age helper: Xs/m/h/d ago, "now", or "—".
func formatAge(d time.Duration) string {
	if d < 0 {
		return "—"
	}
	if d < time.Second {
		return "now"
	}
	if d < time.Minute {
		return fmt.Sprintf("%ds ago", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	}
	if d < 24*time.Hour {
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	}
	return fmt.Sprintf("%dd ago", int(d.Hours()/24))
}
