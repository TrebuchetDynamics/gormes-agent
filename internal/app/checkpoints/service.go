package checkpoints

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	sessionsmodule "github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/gormescli/modules/sessions"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tools"
)

type BuildProvenance struct {
	Version   string `json:"version"`
	GitCommit string `json:"git_commit"`
}

var BuildProvenanceFunc = func() BuildProvenance { return BuildProvenance{} }

// FormatBytes mirrors Hermes' checkpoints _fmt_bytes helper.
func FormatBytes(n int64) string {
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

// Ago returns the age duration for checkpoint timestamps, with -1 for unknown timestamps.
func Ago(now, t time.Time) time.Duration {
	if t.IsZero() {
		return -1
	}
	return now.Sub(t)
}

// FormatAge mirrors Hermes' checkpoints _fmt_age helper: Xs/m/h/d ago, "now", or "—".
func FormatAge(d time.Duration) string {
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

func NewCommand() *cobra.Command {
	return sessionsmodule.NewCheckpointsCommandWithSeams(sessionsmodule.CheckpointsCommandSeams{
		RunStatus:      runCheckpointsStatus,
		RunPrune:       runCheckpointsPrune,
		RunClear:       runCheckpointsClear,
		RunClearLegacy: runCheckpointsClearLegacy,
	})
}

// checkpointsStatusJSON is the wire shape for `checkpoints status --json`.
// Operator scripts driving /rollback storage monitoring parse this to
// alert on growth, identify orphans before pruning, and feed dashboards
// without scraping column-formatted text. Build provenance leads —
// same convention as the rest of the `--json` arc.
type checkpointsStatusJSON struct {
	Build           BuildProvenance          `json:"build"`
	Root            string                   `json:"root"`
	TotalSizeBytes  int64                    `json:"total_size_bytes"`
	StoreSizeBytes  int64                    `json:"store_size_bytes"`
	LegacySizeBytes int64                    `json:"legacy_size_bytes"`
	ProjectCount    int                      `json:"project_count"`
	Projects        []checkpointsProjectJSON `json:"projects"`
	LegacyArchives  []checkpointsLegacyJSON  `json:"legacy_archives"`
}

type checkpointsProjectJSON struct {
	Name      string    `json:"name"`
	Workdir   string    `json:"workdir"`
	Commits   int       `json:"commits"`
	LastTouch time.Time `json:"last_touch"`
	Exists    bool      `json:"exists"`
}

type checkpointsLegacyJSON struct {
	Name      string `json:"name"`
	SizeBytes int64  `json:"size_bytes"`
}

func runCheckpointsStatus(cmd *cobra.Command, _ []string) error {
	root := tools.DefaultCheckpointRoot()
	out := cmd.OutOrStdout()
	result, err := tools.StoreStatus(root)
	if err != nil {
		return fmt.Errorf("checkpoints: %w", err)
	}

	asJSON, _ := cmd.Flags().GetBool("json")
	if asJSON {
		report := checkpointsStatusJSON{
			Build:           BuildProvenanceFunc(),
			Root:            result.Root,
			TotalSizeBytes:  result.TotalSizeBytes,
			StoreSizeBytes:  result.StoreSizeBytes,
			LegacySizeBytes: result.LegacySizeBytes,
			ProjectCount:    result.ProjectCount,
			Projects:        make([]checkpointsProjectJSON, len(result.Projects)),
			LegacyArchives:  make([]checkpointsLegacyJSON, len(result.LegacyArchives)),
		}
		for i, p := range result.Projects {
			report.Projects[i] = checkpointsProjectJSON{
				Name:      p.Name,
				Workdir:   p.Workdir,
				Commits:   p.Commits,
				LastTouch: p.LastTouch,
				Exists:    p.Exists,
			}
		}
		for i, a := range result.LegacyArchives {
			report.LegacyArchives[i] = checkpointsLegacyJSON{
				Name:      a.Name,
				SizeBytes: a.SizeBytes,
			}
		}
		body, marshalErr := json.MarshalIndent(report, "", "  ")
		if marshalErr != nil {
			return marshalErr
		}
		fmt.Fprintln(out, string(body))
		return nil
	}

	if result.ProjectCount == 0 && result.LegacySizeBytes == 0 {
		if _, err := os.Stat(root); os.IsNotExist(err) {
			fmt.Fprintf(out, "No checkpoint store at %s\n", root)
			return nil
		}
	}

	fmt.Fprintf(out, "Checkpoint base: %s\n", root)
	fmt.Fprintf(out, "Total size:      %s\n", FormatBytes(result.TotalSizeBytes))
	fmt.Fprintf(out, "  store/         %s\n", FormatBytes(result.StoreSizeBytes))
	fmt.Fprintf(out, "  legacy-*       %s\n", FormatBytes(result.LegacySizeBytes))
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
				wd, p.Commits, FormatAge(Ago(time.Now(), p.LastTouch)), state)
		}
	}

	if len(result.LegacyArchives) > 0 {
		fmt.Fprintln(out)
		fmt.Fprintf(out, "Legacy archives (%d):\n", len(result.LegacyArchives))
		for _, a := range result.LegacyArchives {
			fmt.Fprintf(out, "  %-40s  %10s\n", a.Name, FormatBytes(a.SizeBytes))
		}
		fmt.Fprintln(out)
		fmt.Fprintln(out, "Clear with: gormes checkpoints clear-legacy")
	}
	return nil
}

// checkpointsPruneJSON is the wire shape for `checkpoints prune --json`.
// Operator scripts gating destructive prunes (CI/CD jobs, scheduled GC,
// dashboard alerts) parse this to reason about what was — or, in
// dry-run mode, what WOULD be — pruned. `dry_run: true` makes the
// preview shape distinguishable from the apply shape. Build provenance
// leads, same convention as restore/update/doctor `--json`.
type checkpointsPruneJSON struct {
	Build          BuildProvenance `json:"build"`
	DryRun         bool            `json:"dry_run"`
	RetentionDays  int             `json:"retention_days"`
	MaxTotalSizeMB int             `json:"max_total_size_mb"`
	KeepOrphans    bool            `json:"keep_orphans"`
	Scanned        int             `json:"scanned"`
	DeletedOrphan  int             `json:"deleted_orphan"`
	DeletedStale   int             `json:"deleted_stale"`
	Errors         int             `json:"errors"`
	BytesFreed     int64           `json:"bytes_freed"`
}

func runCheckpointsPrune(cmd *cobra.Command, _ []string) error {
	out := cmd.OutOrStdout()
	retentionDays, _ := cmd.Flags().GetInt("retention-days")
	maxSizeMB, _ := cmd.Flags().GetInt("max-size-mb")
	keepOrphans, _ := cmd.Flags().GetBool("keep-orphans")
	dryRun, _ := cmd.Flags().GetBool("dry-run")
	asJSON, _ := cmd.Flags().GetBool("json")

	result := tools.PruneCheckpointsDryRun(tools.DefaultCheckpointRoot(), retentionDays, keepOrphans, maxSizeMB, dryRun, time.Now)

	if asJSON {
		body, marshalErr := json.MarshalIndent(checkpointsPruneJSON{
			Build:          BuildProvenanceFunc(),
			DryRun:         dryRun,
			RetentionDays:  retentionDays,
			MaxTotalSizeMB: maxSizeMB,
			KeepOrphans:    keepOrphans,
			Scanned:        result.Scanned,
			DeletedOrphan:  result.DeletedOrphan,
			DeletedStale:   result.DeletedStale,
			Errors:         result.Errors,
			BytesFreed:     result.BytesFreed,
		}, "", "  ")
		if marshalErr != nil {
			return marshalErr
		}
		fmt.Fprintln(out, string(body))
		return nil
	}

	if dryRun {
		fmt.Fprintln(out, "DRY RUN — previewing checkpoint store prune (no files deleted)…")
	} else {
		fmt.Fprintln(out, "Pruning checkpoint store…")
	}
	fmt.Fprintf(out, "  retention_days:    %d\n", retentionDays)
	fmt.Fprintf(out, "  delete_orphans:    %v\n", !keepOrphans)
	fmt.Fprintf(out, "  max_total_size_mb: %d\n", maxSizeMB)
	fmt.Fprintln(out)

	fmt.Fprintf(out, "Scanned:         %d\n", result.Scanned)
	fmt.Fprintf(out, "Deleted orphan:  %d\n", result.DeletedOrphan)
	fmt.Fprintf(out, "Deleted stale:   %d\n", result.DeletedStale)
	fmt.Fprintf(out, "Errors:          %d\n", result.Errors)
	fmt.Fprintf(out, "Bytes reclaimed: %s\n", FormatBytes(result.BytesFreed))
	if dryRun {
		fmt.Fprintln(out)
		fmt.Fprintln(out, "Re-run without --dry-run to apply.")
	}
	return nil
}

// checkpointsClearJSON is the wire shape for `checkpoints clear --json`.
// Operator scripts performing scheduled GC parse this to audit what
// was destroyed (root path, totals before delete, bytes reclaimed).
// Pre-state is captured BEFORE the delete so JSON callers learn what
// the operator just lost.
type checkpointsClearJSON struct {
	Build          BuildProvenance `json:"build"`
	Root           string          `json:"root"`
	Action         string          `json:"action"`
	Deleted        bool            `json:"deleted"`
	BytesFreed     int64           `json:"bytes_freed"`
	ProjectsBefore int             `json:"projects_before"`
	LegacyBefore   int             `json:"legacy_before"`
}

func runCheckpointsClear(cmd *cobra.Command, _ []string) error {
	out := cmd.OutOrStdout()
	force, _ := cmd.Flags().GetBool("force")
	asJSON, _ := cmd.Flags().GetBool("json")
	root := tools.DefaultCheckpointRoot()

	status, _ := tools.StoreStatus(root)
	if status.TotalSizeBytes == 0 {
		if !statCheckpointRoot(root) {
			if asJSON {
				body, marshalErr := json.MarshalIndent(checkpointsClearJSON{
					Build:  BuildProvenanceFunc(),
					Root:   root,
					Action: "noop",
				}, "", "  ")
				if marshalErr != nil {
					return marshalErr
				}
				fmt.Fprintln(out, string(body))
				return nil
			}
			fmt.Fprintln(out, "Nothing to clear — checkpoint base does not exist.")
			return nil
		}
	}

	if !asJSON {
		fmt.Fprintf(out, "This will delete the ENTIRE checkpoint base at %s\n", status.Root)
		fmt.Fprintf(out, "  size:        %s\n", FormatBytes(status.TotalSizeBytes))
		fmt.Fprintf(out, "  projects:    %d\n", status.ProjectCount)
		fmt.Fprintf(out, "  legacy dirs: %d\n", len(status.LegacyArchives))
		fmt.Fprintln(out)
		fmt.Fprintln(out, "All /rollback history for every working directory will be lost.")
	}

	if !force {
		if !confirm(out, cmd.InOrStdin(), "Proceed?") {
			fmt.Fprintln(out, "Aborted.")
			return nil
		}
	}

	result := tools.ClearAll(root)
	if asJSON {
		action := "cleared"
		if !result.Deleted {
			action = "failed"
		}
		body, marshalErr := json.MarshalIndent(checkpointsClearJSON{
			Build:          BuildProvenanceFunc(),
			Root:           status.Root,
			Action:         action,
			Deleted:        result.Deleted,
			BytesFreed:     result.BytesFreed,
			ProjectsBefore: status.ProjectCount,
			LegacyBefore:   len(status.LegacyArchives),
		}, "", "  ")
		if marshalErr != nil {
			return marshalErr
		}
		fmt.Fprintln(out, string(body))
		return nil
	}
	if result.Deleted {
		fmt.Fprintf(out, "Cleared. Reclaimed %s.\n", FormatBytes(result.BytesFreed))
		return nil
	}
	fmt.Fprintln(out, "Could not clear checkpoint base (see logs).")
	return nil
}

// checkpointsClearLegacyJSON is the wire shape for `checkpoints clear-legacy --json`.
// `archives_before` captures the names+sizes that were destroyed —
// computed BEFORE delete so JSON consumers can audit/log exactly which
// legacy directories are gone.
type checkpointsClearLegacyJSON struct {
	Build          BuildProvenance         `json:"build"`
	Root           string                  `json:"root"`
	Action         string                  `json:"action"`
	ArchivesBefore []checkpointsLegacyJSON `json:"archives_before"`
	Deleted        int                     `json:"deleted"`
	BytesFreed     int64                   `json:"bytes_freed"`
}

func runCheckpointsClearLegacy(cmd *cobra.Command, _ []string) error {
	out := cmd.OutOrStdout()
	force, _ := cmd.Flags().GetBool("force")
	asJSON, _ := cmd.Flags().GetBool("json")
	root := tools.DefaultCheckpointRoot()

	status, _ := tools.StoreStatus(root)
	legacy := status.LegacyArchives
	if len(legacy) == 0 {
		if asJSON {
			body, marshalErr := json.MarshalIndent(checkpointsClearLegacyJSON{
				Build:          BuildProvenanceFunc(),
				Root:           root,
				Action:         "noop",
				ArchivesBefore: []checkpointsLegacyJSON{},
			}, "", "  ")
			if marshalErr != nil {
				return marshalErr
			}
			fmt.Fprintln(out, string(body))
			return nil
		}
		fmt.Fprintln(out, "No legacy archives to clear.")
		return nil
	}

	if !asJSON {
		var total int64
		for _, a := range legacy {
			total += a.SizeBytes
		}
		fmt.Fprintf(out, "Found %d legacy archive(s), total %s:\n", len(legacy), FormatBytes(total))
		for _, a := range legacy {
			fmt.Fprintf(out, "  %-40s  %10s\n", a.Name, FormatBytes(a.SizeBytes))
		}
		fmt.Fprintln(out)
		fmt.Fprintln(out, "Legacy archives hold pre-v2 per-project shadow repos, moved aside")
		fmt.Fprintln(out, "during the single-store migration. Delete when you're confident")
		fmt.Fprintln(out, "you don't need the old /rollback history.")
	}

	if !force {
		if !confirm(out, cmd.InOrStdin(), "Delete all legacy archives?") {
			fmt.Fprintln(out, "Aborted.")
			return nil
		}
	}

	result := tools.ClearLegacy(root)
	deleted := countLegacyDeleted(status, root)
	if asJSON {
		archives := make([]checkpointsLegacyJSON, len(legacy))
		for i, a := range legacy {
			archives[i] = checkpointsLegacyJSON{
				Name:      a.Name,
				SizeBytes: a.SizeBytes,
			}
		}
		body, marshalErr := json.MarshalIndent(checkpointsClearLegacyJSON{
			Build:          BuildProvenanceFunc(),
			Root:           status.Root,
			Action:         "cleared",
			ArchivesBefore: archives,
			Deleted:        deleted,
			BytesFreed:     result.BytesFreed,
		}, "", "  ")
		if marshalErr != nil {
			return marshalErr
		}
		fmt.Fprintln(out, string(body))
		return nil
	}
	fmt.Fprintf(out, "Deleted %d archive(s), reclaimed %s.\n", deleted, FormatBytes(result.BytesFreed))
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
