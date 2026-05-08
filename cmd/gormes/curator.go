package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/skills"
)

type curatorCommandDeps struct {
	skillsRoot func() string
	now        func() time.Time
	reviewer   skills.CuratorReviewer
}

type curatorBackupRow struct {
	ID        string
	Reason    string
	CreatedAt string
}

func newCuratorCommand() *cobra.Command {
	return newCuratorCommandWithDeps(curatorCommandDeps{})
}

func newCuratorCommandWithDeps(deps curatorCommandDeps) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "curator",
		Short: "Manage Hermes-compatible background skill curation",
	}
	cmd.AddCommand(
		newCuratorStatusCommand(deps),
		newCuratorRunCommand(deps),
		newCuratorPauseCommand(deps),
		newCuratorResumeCommand(deps),
		newCuratorPinCommand(deps),
		newCuratorUnpinCommand(deps),
		newCuratorBackupCommand(deps),
		newCuratorRollbackCommand(deps),
		newCuratorRestoreCommand(deps),
	)
	return cmd
}

func newCuratorStatusCommand(deps curatorCommandDeps) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show curator status and skill stats",
		RunE: func(cmd *cobra.Command, _ []string) error {
			root := resolveCuratorSkillsRoot(deps)
			curator := newCuratorForCommand(root, deps)
			state, err := curator.LoadState()
			if err != nil {
				return err
			}
			rows, err := skills.ListAgentCreatedSkillUsage(root)
			if err != nil {
				return err
			}
			asJSON, _ := cmd.Flags().GetBool("json")
			if asJSON {
				return writeCuratorStatusJSON(cmd.OutOrStdout(), state, rows)
			}
			status := "ENABLED"
			if state.Paused {
				status = "PAUSED"
			}
			out := cmd.OutOrStdout()
			if _, err := fmt.Fprintf(out, "curator: %s\n", status); err != nil {
				return err
			}
			if _, err := fmt.Fprintf(out, "  runs:           %d\n", state.RunCount); err != nil {
				return err
			}
			if _, err := fmt.Fprintf(out, "  last run:       %s\n", formatCuratorTimestamp(state.LastRunAt, deps)); err != nil {
				return err
			}
			summary := state.LastRunSummary
			if summary == "" {
				summary = "(none)"
			}
			if _, err := fmt.Fprintf(out, "  last summary:   %s\n", summary); err != nil {
				return err
			}
			if state.LastReportPath != "" {
				if _, err := fmt.Fprintf(out, "  last report:    %s\n", state.LastReportPath); err != nil {
					return err
				}
			}
			if _, err := fmt.Fprintf(out, "  interval:       every %s\n", curatorIntervalLabel(skills.DefaultCuratorIntervalHours)); err != nil {
				return err
			}
			if _, err := fmt.Fprintf(out, "  stale after:    %dd unused\n", skills.DefaultCuratorStaleDays); err != nil {
				return err
			}
			if _, err := fmt.Fprintf(out, "  archive after:  %dd unused\n", skills.DefaultCuratorArchiveDays); err != nil {
				return err
			}
			return writeCuratorStatusRows(cmd, rows)
		},
	}
	cmd.Flags().Bool("json", false, "emit machine-readable JSON: `{build, state, defaults, skills: {total, rows: [...]}}`")
	return cmd
}

// curatorStatusReportJSON is the wire shape for `curator status --json`.
// Fleet dashboards parse this to monitor curator across machines without
// scraping multi-section prose. `state` mirrors the CuratorState struct;
// `defaults` documents the build-baked thresholds; `skills.rows` lets
// dashboards sort by activity, identify stale skills, and feed alerts.
type curatorStatusReportJSON struct {
	Build    buildProvenanceJSON       `json:"build"`
	State    curatorStateJSON          `json:"state"`
	Defaults curatorDefaultsJSON       `json:"defaults"`
	Skills   curatorSkillsJSON         `json:"skills"`
}

type curatorStateJSON struct {
	Paused                 bool       `json:"paused"`
	RunCount               int        `json:"run_count"`
	LastRunAt              *time.Time `json:"last_run_at,omitempty"`
	LastRunDurationSeconds float64    `json:"last_run_duration_seconds,omitempty"`
	LastRunSummary         string     `json:"last_run_summary,omitempty"`
	LastReportPath         string     `json:"last_report_path,omitempty"`
}

type curatorDefaultsJSON struct {
	IntervalHours int `json:"interval_hours"`
	StaleDays     int `json:"stale_days"`
	ArchiveDays   int `json:"archive_days"`
}

type curatorSkillsJSON struct {
	Total int                 `json:"total"`
	Rows  []curatorSkillRowJSON `json:"rows"`
}

type curatorSkillRowJSON struct {
	Name           string    `json:"name"`
	Activity       int       `json:"activity"`
	UseCount       int       `json:"use_count"`
	LastUsedAt     time.Time `json:"last_used_at,omitempty"`
	LastActivityAt time.Time `json:"last_activity_at,omitempty"`
	SkillDir       string    `json:"skill_dir,omitempty"`
}

func writeCuratorStatusJSON(out interface{ Write(p []byte) (int, error) }, state skills.CuratorState, rows []skills.AgentCreatedSkillUsage) error {
	report := curatorStatusReportJSON{
		Build: newBuildProvenance(),
		State: curatorStateJSON{
			Paused:                 state.Paused,
			RunCount:               state.RunCount,
			LastRunAt:              state.LastRunAt,
			LastRunDurationSeconds: state.LastRunDurationSeconds,
			LastRunSummary:         state.LastRunSummary,
			LastReportPath:         state.LastReportPath,
		},
		Defaults: curatorDefaultsJSON{
			IntervalHours: skills.DefaultCuratorIntervalHours,
			StaleDays:     skills.DefaultCuratorStaleDays,
			ArchiveDays:   skills.DefaultCuratorArchiveDays,
		},
		Skills: curatorSkillsJSON{
			Total: len(rows),
			Rows:  make([]curatorSkillRowJSON, len(rows)),
		},
	}
	for i, r := range rows {
		report.Skills.Rows[i] = curatorSkillRowJSON{
			Name:           r.Name,
			Activity:       r.Record.UseCount,
			UseCount:       r.Record.UseCount,
			LastUsedAt:     r.Record.LastUsedAt,
			LastActivityAt: r.LastActivityAt,
			SkillDir:       r.SkillDir,
		}
	}
	body, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}
	fmt.Fprintln(out, string(body))
	return nil
}

func newCuratorRunCommand(deps curatorCommandDeps) *cobra.Command {
	var dryRun bool
	var synchronous bool
	cmd := &cobra.Command{
		Use:   "run",
		Short: "Trigger a curator review now",
		RunE: func(cmd *cobra.Command, _ []string) error {
			out := cmd.OutOrStdout()
			if dryRun {
				if _, err := fmt.Fprintln(out, "curator: running DRY-RUN (report only, no mutations)..."); err != nil {
					return err
				}
			} else {
				if _, err := fmt.Fprintln(out, "curator: running review pass..."); err != nil {
					return err
				}
			}
			root := resolveCuratorSkillsRoot(deps)
			report, err := newCuratorForCommand(root, deps).Run(cmd.Context(), skills.CuratorRunOptions{DryRun: dryRun})
			if err != nil {
				return err
			}
			if report.Summary != "" {
				if _, err := fmt.Fprintf(out, "%s\n", report.Summary); err != nil {
					return err
				}
			}
			if dryRun {
				if _, err := fmt.Fprintf(out, "auto (preview): %d candidate skill(s) - no transitions applied in dry-run\n", len(report.BeforeNames)); err != nil {
					return err
				}
			} else {
				if _, err := fmt.Fprintf(out, "auto: checked=%d stale=%d archived=%d reactivated=%d\n", report.AutoCounts.Checked, report.AutoCounts.MarkedStale, report.AutoCounts.Archived, report.AutoCounts.Reactivated); err != nil {
					return err
				}
			}
			if report.LastReportPath != "" {
				if _, err := fmt.Fprintf(out, "report: %s\n", report.LastReportPath); err != nil {
					return err
				}
			}
			if !synchronous {
				if _, err := fmt.Fprintln(out, "llm pass completed synchronously in this native runtime"); err != nil {
					return err
				}
			}
			if dryRun {
				_, err = fmt.Fprintln(out, "dry-run: no changes applied. Read the report and run `gormes curator run` to apply.")
				return err
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&synchronous, "sync", false, "Wait for the review pass to finish")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Report only; do not mutate state, archives, or skill content")
	return cmd
}

func newCuratorPauseCommand(deps curatorCommandDeps) *cobra.Command {
	return &cobra.Command{
		Use:   "pause",
		Short: "Pause curator runs until resumed",
		RunE: func(cmd *cobra.Command, _ []string) error {
			root := resolveCuratorSkillsRoot(deps)
			curator := newCuratorForCommand(root, deps)
			state, err := curator.LoadState()
			if err != nil {
				return err
			}
			state.Paused = true
			if err := curator.SaveState(state); err != nil {
				return err
			}
			_, err = fmt.Fprintln(cmd.OutOrStdout(), "curator: paused")
			return err
		},
	}
}

func newCuratorResumeCommand(deps curatorCommandDeps) *cobra.Command {
	return &cobra.Command{
		Use:   "resume",
		Short: "Resume curator runs",
		RunE: func(cmd *cobra.Command, _ []string) error {
			root := resolveCuratorSkillsRoot(deps)
			curator := newCuratorForCommand(root, deps)
			state, err := curator.LoadState()
			if err != nil {
				return err
			}
			state.Paused = false
			if err := curator.SaveState(state); err != nil {
				return err
			}
			_, err = fmt.Fprintln(cmd.OutOrStdout(), "curator: resumed")
			return err
		},
	}
}

func newCuratorPinCommand(deps curatorCommandDeps) *cobra.Command {
	return &cobra.Command{
		Use:   "pin <skill>",
		Short: "Pin an agent-created skill so curator never auto-transitions it",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCuratorPin(cmd, resolveCuratorSkillsRoot(deps), args[0], true)
		},
	}
}

func newCuratorUnpinCommand(deps curatorCommandDeps) *cobra.Command {
	return &cobra.Command{
		Use:   "unpin <skill>",
		Short: "Unpin an agent-created skill",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCuratorPin(cmd, resolveCuratorSkillsRoot(deps), args[0], false)
		},
	}
}

func newCuratorBackupCommand(deps curatorCommandDeps) *cobra.Command {
	var reason string
	var asJSON bool
	cmd := &cobra.Command{
		Use:   "backup",
		Short: "Take a manual curator snapshot",
		RunE: func(cmd *cobra.Command, _ []string) error {
			root := resolveCuratorSkillsRoot(deps)
			if strings.TrimSpace(reason) == "" {
				reason = "manual"
			}
			backup, err := skills.CreateCuratorBackup(root, curatorNow(deps), reason, nil)
			if err != nil {
				return err
			}
			if asJSON {
				body, marshalErr := json.MarshalIndent(curatorBackupReportJSON{
					Build:        newBuildProvenance(),
					ID:           backup.ID,
					ArchivePath:  backup.ArchivePath,
					ManifestPath: backup.ManifestPath,
					Reason:       reason,
				}, "", "  ")
				if marshalErr != nil {
					return marshalErr
				}
				_, err = fmt.Fprintln(cmd.OutOrStdout(), string(body))
				return err
			}
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "curator: snapshot created at %s\n", filepath.Dir(backup.ArchivePath))
			return err
		},
	}
	cmd.Flags().StringVar(&reason, "reason", "", "Free-text label stored in manifest.json")
	cmd.Flags().BoolVar(&asJSON, "json", false, "emit machine-readable JSON: `{build, id, archive_path, manifest_path, reason}`")
	return cmd
}

// curatorBackupReportJSON is the wire shape for `curator backup --json`.
// Fleet automation taking pre-deploy snapshots parses this to record
// the snapshot id (for later rollback) and archive path. The reason
// echo lets dashboards correlate snapshots with the deploy that
// triggered them.
type curatorBackupReportJSON struct {
	Build        buildProvenanceJSON `json:"build"`
	ID           string              `json:"id"`
	ArchivePath  string              `json:"archive_path"`
	ManifestPath string              `json:"manifest_path"`
	Reason       string              `json:"reason"`
}

func newCuratorRollbackCommand(deps curatorCommandDeps) *cobra.Command {
	var list bool
	var backupID string
	var yes bool
	var asJSON bool
	cmd := &cobra.Command{
		Use:   "rollback",
		Short: "Restore skills from a curator snapshot",
		RunE: func(cmd *cobra.Command, _ []string) error {
			root := resolveCuratorSkillsRoot(deps)
			rows, err := listCuratorBackups(root)
			if err != nil {
				return err
			}
			if list {
				if asJSON {
					return writeCuratorBackupListJSON(cmd.OutOrStdout(), rows)
				}
				return writeCuratorBackupList(cmd, rows)
			}
			target := resolveCuratorBackupID(rows, backupID)
			if target == "" {
				if len(rows) == 0 {
					_, _ = fmt.Fprintln(cmd.OutOrStdout(), "curator: no snapshots exist yet. Take one with `gormes curator backup` or wait for the next curator run.")
				} else {
					_, _ = fmt.Fprintf(cmd.OutOrStdout(), "curator: no snapshot matching %q\n", backupID)
				}
				return newExitCodeError(1, fmt.Errorf("curator_rollback_snapshot_missing"))
			}
			if !asJSON {
				if _, err := fmt.Fprintf(cmd.OutOrStdout(), "Rollback target: %s\n", target); err != nil {
					return err
				}
			}
			if !yes {
				if _, err := fmt.Fprintln(cmd.OutOrStdout(), "This will replace the current Gormes skills tree after taking a safety snapshot."); err != nil {
					return err
				}
				if _, err := fmt.Fprint(cmd.OutOrStdout(), "Proceed? [y/N] "); err != nil {
					return err
				}
				answer, _ := bufio.NewReader(cmd.InOrStdin()).ReadString('\n')
				answer = strings.ToLower(strings.TrimSpace(answer))
				if answer != "y" && answer != "yes" {
					_, _ = fmt.Fprintln(cmd.OutOrStdout(), "cancelled")
					return newExitCodeError(1, fmt.Errorf("curator_rollback_cancelled"))
				}
			}
			rollback, err := skills.RollbackCuratorBackup(root, target, curatorNow(deps))
			if err != nil {
				return err
			}
			if asJSON {
				body, marshalErr := json.MarshalIndent(curatorRollbackReportJSON{
					Build:               newBuildProvenance(),
					Action:              "rolled_back",
					RestoredBackupID:    rollback.RestoredBackupID,
					PreRollbackBackupID: rollback.PreRollbackBackupID,
				}, "", "  ")
				if marshalErr != nil {
					return marshalErr
				}
				_, err = fmt.Fprintln(cmd.OutOrStdout(), string(body))
				return err
			}
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "curator: rollback restored %s (safety snapshot %s)\n", rollback.RestoredBackupID, rollback.PreRollbackBackupID)
			return err
		},
	}
	cmd.Flags().BoolVar(&list, "list", false, "List available snapshots and exit")
	cmd.Flags().StringVar(&backupID, "id", "", "Snapshot id to restore; defaults to newest")
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "Skip confirmation prompt")
	cmd.Flags().BoolVar(&asJSON, "json", false, "emit machine-readable JSON: `--list` returns `{build, backups: [{id, reason, created_at}]}`; rollback returns `{build, action, restored_backup_id, pre_rollback_backup_id}`")
	return cmd
}

// curatorRollbackReportJSON is the wire shape for `curator rollback --json`
// (apply mode). Operator scripts capture `pre_rollback_backup_id` to undo
// the rollback if needed — that's the safety snapshot the rollback
// implicitly takes before mutating the live tree.
type curatorRollbackReportJSON struct {
	Build               buildProvenanceJSON `json:"build"`
	Action              string              `json:"action"`
	RestoredBackupID    string              `json:"restored_backup_id"`
	PreRollbackBackupID string              `json:"pre_rollback_backup_id"`
}

// curatorBackupListJSON is the wire shape for `curator rollback --list --json`.
type curatorBackupListJSON struct {
	Build   buildProvenanceJSON          `json:"build"`
	Backups []curatorBackupListEntryJSON `json:"backups"`
}

type curatorBackupListEntryJSON struct {
	ID        string `json:"id"`
	Reason    string `json:"reason"`
	CreatedAt string `json:"created_at"`
}

func writeCuratorBackupListJSON(out interface{ Write(p []byte) (int, error) }, rows []curatorBackupRow) error {
	report := curatorBackupListJSON{
		Build:   newBuildProvenance(),
		Backups: make([]curatorBackupListEntryJSON, len(rows)),
	}
	for i, r := range rows {
		report.Backups[i] = curatorBackupListEntryJSON{
			ID:        r.ID,
			Reason:    r.Reason,
			CreatedAt: r.CreatedAt,
		}
	}
	body, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}
	fmt.Fprintln(out, string(body))
	return nil
}

func newCuratorRestoreCommand(deps curatorCommandDeps) *cobra.Command {
	return &cobra.Command{
		Use:   "restore <skill>",
		Short: "Restore one archived skill",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			root := resolveCuratorSkillsRoot(deps)
			if err := restoreCuratorArchivedSkill(root, args[0]); err != nil {
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "curator: %v\n", err)
				return newExitCodeError(1, err)
			}
			_, err := fmt.Fprintf(cmd.OutOrStdout(), "curator: restored '%s'\n", args[0])
			return err
		},
	}
}

func resolveCuratorSkillsRoot(deps curatorCommandDeps) string {
	if deps.skillsRoot != nil {
		if root := strings.TrimSpace(deps.skillsRoot()); root != "" {
			return root
		}
	}
	cfg, err := config.Load(nil)
	if err == nil {
		return cfg.SkillsRoot()
	}
	return filepath.Join(config.GormesHome(), "skills")
}

func newCuratorForCommand(root string, deps curatorCommandDeps) *skills.Curator {
	return skills.NewCurator(skills.CuratorConfig{
		Root:     root,
		Now:      deps.now,
		Reviewer: deps.reviewer,
	})
}

func curatorNow(deps curatorCommandDeps) time.Time {
	if deps.now != nil {
		return deps.now().UTC()
	}
	return time.Now().UTC()
}

func formatCuratorTimestamp(ts *time.Time, deps curatorCommandDeps) string {
	if ts == nil || ts.IsZero() {
		return "never"
	}
	now := curatorNow(deps)
	delta := now.Sub(ts.UTC())
	if delta < 0 {
		return ts.UTC().Format(time.RFC3339)
	}
	switch {
	case delta < time.Minute:
		return fmt.Sprintf("%ds ago", int(delta.Seconds()))
	case delta < time.Hour:
		return fmt.Sprintf("%dm ago", int(delta.Minutes()))
	case delta < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(delta.Hours()))
	default:
		return fmt.Sprintf("%dd ago", int(delta.Hours()/24))
	}
}

func curatorIntervalLabel(hours int) string {
	if hours >= 24 && hours%24 == 0 {
		return fmt.Sprintf("%dd", hours/24)
	}
	return fmt.Sprintf("%dh", hours)
}

func writeCuratorStatusRows(cmd *cobra.Command, rows []skills.AgentCreatedSkillUsage) error {
	out := cmd.OutOrStdout()
	if len(rows) == 0 {
		_, err := fmt.Fprintln(out, "\nno agent-created skills")
		return err
	}
	sort.SliceStable(rows, func(i, j int) bool { return rows[i].Name < rows[j].Name })
	byState := map[string][]skills.AgentCreatedSkillUsage{
		skills.SkillStateActive:   {},
		skills.SkillStateStale:    {},
		skills.SkillStateArchived: {},
	}
	var pinned []string
	for _, row := range rows {
		state := row.Record.State
		if state == "" {
			state = skills.SkillStateActive
		}
		byState[state] = append(byState[state], row)
		if row.Record.Pinned {
			pinned = append(pinned, row.Name)
		}
	}
	if _, err := fmt.Fprintf(out, "\nagent-created skills: %d total\n", len(rows)); err != nil {
		return err
	}
	for _, state := range []string{skills.SkillStateActive, skills.SkillStateStale, skills.SkillStateArchived} {
		if _, err := fmt.Fprintf(out, "  %-10s %d\n", state, len(byState[state])); err != nil {
			return err
		}
	}
	if len(pinned) > 0 {
		if _, err := fmt.Fprintf(out, "\npinned (%d): %s\n", len(pinned), strings.Join(pinned, ", ")); err != nil {
			return err
		}
	}
	active := append([]skills.AgentCreatedSkillUsage(nil), byState[skills.SkillStateActive]...)
	sort.SliceStable(active, func(i, j int) bool {
		return active[i].LastActivityAt.Before(active[j].LastActivityAt)
	})
	if len(active) > 0 {
		if _, err := fmt.Fprintln(out, "\nleast recently active (top 5):"); err != nil {
			return err
		}
		if err := writeCuratorActivityRows(out, active[:min(len(active), 5)]); err != nil {
			return err
		}
	}
	mostActive := append([]skills.AgentCreatedSkillUsage(nil), active...)
	sort.SliceStable(mostActive, func(i, j int) bool {
		ai, aj := curatorActivityCount(mostActive[i].Record), curatorActivityCount(mostActive[j].Record)
		if ai != aj {
			return ai > aj
		}
		return mostActive[i].LastActivityAt.After(mostActive[j].LastActivityAt)
	})
	if len(mostActive) > 0 && curatorActivityCount(mostActive[0].Record) > 0 {
		if _, err := fmt.Fprintln(out, "\nmost active (top 5):"); err != nil {
			return err
		}
		if err := writeCuratorActivityRows(out, mostActive[:min(len(mostActive), 5)]); err != nil {
			return err
		}
	}
	leastActive := append([]skills.AgentCreatedSkillUsage(nil), active...)
	sort.SliceStable(leastActive, func(i, j int) bool {
		ai, aj := curatorActivityCount(leastActive[i].Record), curatorActivityCount(leastActive[j].Record)
		if ai != aj {
			return ai < aj
		}
		return leastActive[i].LastActivityAt.Before(leastActive[j].LastActivityAt)
	})
	if len(leastActive) > 0 {
		if _, err := fmt.Fprintln(out, "\nleast active (top 5):"); err != nil {
			return err
		}
		if err := writeCuratorActivityRows(out, leastActive[:min(len(leastActive), 5)]); err != nil {
			return err
		}
	}
	return nil
}

func writeCuratorActivityRows(out anyWriter, rows []skills.AgentCreatedSkillUsage) error {
	for _, row := range rows {
		if _, err := fmt.Fprintf(out, "  %-40s  activity=%3d  use=%3d  view=%3d  patches=%3d  last_activity=%s\n",
			row.Name,
			curatorActivityCount(row.Record),
			row.Record.UseCount,
			row.Record.ViewCount,
			row.Record.PatchCount,
			formatCuratorActivityTime(row.LastActivityAt),
		); err != nil {
			return err
		}
	}
	return nil
}

type anyWriter interface {
	Write([]byte) (int, error)
}

func curatorActivityCount(rec skills.SkillUsageRecord) int {
	return rec.UseCount + rec.ViewCount + rec.PatchCount
}

func formatCuratorActivityTime(t time.Time) string {
	if t.IsZero() {
		return "never"
	}
	return t.UTC().Format(time.RFC3339)
}

func runCuratorPin(cmd *cobra.Command, root, name string, pinned bool) error {
	if !curatorAgentCreatedActive(root, name) {
		if pinned {
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "curator: '%s' is bundled or hub-installed - cannot pin (only agent-created skills participate in curation)\n", name)
			return newExitCodeError(1, fmt.Errorf("curator_pin_refused"))
		}
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "curator: '%s' is bundled or hub-installed - there's nothing to unpin (curator only tracks agent-created skills)\n", name)
		return newExitCodeError(1, fmt.Errorf("curator_unpin_refused"))
	}
	if err := skills.SetPinned(root, name, pinned); err != nil {
		return err
	}
	if pinned {
		_, err := fmt.Fprintf(cmd.OutOrStdout(), "curator: pinned '%s' (will bypass auto-transitions)\n", name)
		return err
	}
	_, err := fmt.Fprintf(cmd.OutOrStdout(), "curator: unpinned '%s'\n", name)
	return err
}

func curatorAgentCreatedActive(root, name string) bool {
	rows, err := skills.ListAgentCreatedSkillUsage(root)
	if err != nil {
		return false
	}
	for _, row := range rows {
		if row.Name == name {
			return true
		}
	}
	return false
}

func listCuratorBackups(root string) ([]curatorBackupRow, error) {
	dir := filepath.Join(root, ".curator_backups")
	entries, err := os.ReadDir(dir)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	rows := make([]curatorBackupRow, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() || strings.HasPrefix(entry.Name(), ".") {
			continue
		}
		row := curatorBackupRow{ID: entry.Name()}
		raw, err := os.ReadFile(filepath.Join(dir, entry.Name(), "manifest.json"))
		if err == nil {
			var manifest struct {
				Reason    string `json:"reason"`
				CreatedAt string `json:"created_at"`
			}
			if json.Unmarshal(raw, &manifest) == nil {
				row.Reason = manifest.Reason
				row.CreatedAt = manifest.CreatedAt
			}
		}
		rows = append(rows, row)
	}
	sort.SliceStable(rows, func(i, j int) bool { return rows[i].ID > rows[j].ID })
	return rows, nil
}

func writeCuratorBackupList(cmd *cobra.Command, rows []curatorBackupRow) error {
	if len(rows) == 0 {
		_, err := fmt.Fprintln(cmd.OutOrStdout(), "curator: no snapshots")
		return err
	}
	for _, row := range rows {
		reason := row.Reason
		if reason == "" {
			reason = "unknown"
		}
		created := row.CreatedAt
		if created == "" {
			created = "unknown"
		}
		if _, err := fmt.Fprintf(cmd.OutOrStdout(), "%s  %s  %s\n", row.ID, created, reason); err != nil {
			return err
		}
	}
	return nil
}

func resolveCuratorBackupID(rows []curatorBackupRow, query string) string {
	query = strings.TrimSpace(query)
	if query == "" {
		if len(rows) == 0 {
			return ""
		}
		return rows[0].ID
	}
	for _, row := range rows {
		if row.ID == query || strings.HasPrefix(row.ID, query) {
			return row.ID
		}
	}
	return ""
}

func restoreCuratorArchivedSkill(root, name string) error {
	if name == "" || filepath.Base(name) != name || strings.Contains(name, "..") {
		return fmt.Errorf("unsafe skill name %q", name)
	}
	src := filepath.Join(root, "active", ".archive", name)
	dst := filepath.Join(root, "active", name)
	if _, err := os.Stat(src); err != nil {
		return fmt.Errorf("archived skill %q not found", name)
	}
	if _, err := os.Stat(dst); err == nil {
		return fmt.Errorf("active skill %q already exists", name)
	}
	if err := os.Rename(src, dst); err != nil {
		return err
	}
	return skills.SetSkillState(root, name, skills.SkillStateActive)
}
