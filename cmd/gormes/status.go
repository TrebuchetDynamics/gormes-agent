package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/automation/cron"
	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli"
	toolspkg "github.com/TrebuchetDynamics/gormes-agent/internal/tools"
	"github.com/spf13/cobra"
)

// statusReportJSON is the wire shape for `gormes status --json`.
// Build provenance leads, then the blockers array — same convention
// as update --json / doctor --json. The `system` block + `audit_path`
// mirror the system-events line the text surface prints
// via renderSystemStatusLine, so JSON consumers can ingest the same
// information without scraping prose. The `progress` block surfaces
// progress.Load failures as structured unavailable evidence (the
// text surface already renders this gracefully — JSON now matches).
type statusReportJSON struct {
	Build             buildProvenanceJSON           `json:"build"`
	Progress          *statusProgressJSON           `json:"progress,omitempty"`
	Blockers          []cli.StatusBlocker           `json:"blockers"`
	System            toolspkg.SystemEventsSnapshot `json:"system"`
	AuditPath         string                        `json:"audit_path"`
	OperatorRunReport *operatorRunReportJSON        `json:"operator_run_report,omitempty"`
}

type operatorRunReportJSON struct {
	Status                 string `json:"status"`
	JobID                  string `json:"job_id,omitempty"`
	RunID                  int64  `json:"run_id,omitempty"`
	Profile                string `json:"profile,omitempty"`
	StartedAtUnix          int64  `json:"started_at_unix,omitempty"`
	FinishedAtUnix         int64  `json:"finished_at_unix,omitempty"`
	DegradedReason         string `json:"degraded_reason,omitempty"`
	RecommendedNextCommand string `json:"recommended_next_command,omitempty"`
}

// statusProgressJSON carries the parity equivalent of the text
// surface's `blockers: unavailable status=progress_unavailable
// reason=...` line. Operators on a freshly-imaged host (no
// progress.json yet) get a structured degraded snapshot rather than
// a non-zero exit + raw filesystem error.
type statusProgressJSON struct {
	Status string `json:"status"`
	Reason string `json:"reason,omitempty"`
}

func newStatusCommand() *cobra.Command {
	var progressPath string
	var operatorReportDir string
	var asJSON bool
	cmd := &cobra.Command{
		Use:          "status",
		Short:        "Show Gormes runtime and progress blockers",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if asJSON {
				return runStatusJSON(cmd, progressPath, operatorReportDir)
			}
			return runStatusText(cmd, progressPath, operatorReportDir)
		},
	}
	cmd.Flags().StringVar(&progressPath, "progress", cli.DefaultStatusProgressPath, "progress.json path used for blocker status")
	cmd.Flags().StringVar(&operatorReportDir, "operator-report-dir", defaultOperatorReportDir(), "directory containing operator run reports")
	cmd.Flags().BoolVar(&asJSON, "json", false, "emit a machine-readable {blockers: [...]} JSON document (suitable for monitoring/automation)")
	return cmd
}

func defaultOperatorReportDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".gormes")
}

func findLatestOperatorReport(dir string) (cron.OperatorRunReport, error) {
	if dir == "" {
		return cron.OperatorRunReport{}, os.ErrNotExist
	}

	runsDir := filepath.Join(dir, "operator-runs")
	entries, err := os.ReadDir(runsDir)
	if err != nil {
		return cron.OperatorRunReport{}, err
	}

	var latestPath string
	var latestMod time.Time
	for _, jobEntry := range entries {
		if !jobEntry.IsDir() {
			continue
		}
		jobDir := filepath.Join(runsDir, jobEntry.Name())
		runFiles, err := os.ReadDir(jobDir)
		if err != nil {
			continue
		}
		for _, f := range runFiles {
			if !strings.HasSuffix(f.Name(), ".json") {
				continue
			}
			fullPath := filepath.Join(jobDir, f.Name())
			info, err := f.Info()
			if err != nil {
				continue
			}
			if info.ModTime().After(latestMod) {
				latestMod = info.ModTime()
				latestPath = fullPath
			}
		}
	}

	if latestPath == "" {
		return cron.OperatorRunReport{}, os.ErrNotExist
	}
	return cron.ReadOperatorRunReport(latestPath)
}

func runStatusJSON(cmd *cobra.Command, progressPath, operatorReportDir string) error {
	blockers, err := cli.CollectStatusBlockers(cli.StatusReportOptions{ProgressPath: progressPath})
	var progressUnavailable *statusProgressJSON
	if err != nil {
		progressUnavailable = &statusProgressJSON{
			Status: "progress_unavailable",
			Reason: err.Error(),
		}
		blockers = nil
	}
	if blockers == nil {
		blockers = []cli.StatusBlocker{}
	}

	system := collectSystemSnapshotForJSON(cmd)
	report := statusReportJSON{
		Build:     newBuildProvenance(),
		Progress:  progressUnavailable,
		Blockers:  blockers,
		System:    system,
		AuditPath: config.ToolAuditLogPath(),
	}

	if rpt, err := findLatestOperatorReport(operatorReportDir); err == nil {
		report.OperatorRunReport = &operatorRunReportJSON{
			Status:                 rpt.Status,
			JobID:                  rpt.JobID,
			RunID:                  rpt.RunID,
			Profile:                rpt.Profile,
			StartedAtUnix:          rpt.StartedAtUnix,
			FinishedAtUnix:         rpt.FinishedAtUnix,
			DegradedReason:         rpt.DegradedReason,
			RecommendedNextCommand: rpt.RecommendedNextCommand,
		}
	}

	body, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(cmd.OutOrStdout(), string(body))
	return err
}

func runStatusText(cmd *cobra.Command, progressPath, operatorReportDir string) error {
	report, err := cli.RenderStatusReport(cli.StatusReportOptions{ProgressPath: progressPath})
	if err != nil {
		return err
	}

	out := report + renderSystemStatusLine(cmd)

	if rpt, err := findLatestOperatorReport(operatorReportDir); err == nil {
		out += formatOperatorRunReportText(rpt)
	} else {
		out += "operator run report: operator_report_unavailable\n"
	}

	_, err = fmt.Fprint(cmd.OutOrStdout(), out)
	return err
}

func formatOperatorRunReportText(rpt cron.OperatorRunReport) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("latest run: %s (run %d) status=%s\n", rpt.JobID, rpt.RunID, rpt.Status))
	if rpt.Profile != "" {
		sb.WriteString(fmt.Sprintf("  profile: %s\n", rpt.Profile))
	}
	if rpt.StartedAtUnix > 0 {
		sb.WriteString(fmt.Sprintf("  started: %s\n", time.Unix(rpt.StartedAtUnix, 0).Format(time.RFC3339)))
	}
	if rpt.FinishedAtUnix > 0 {
		sb.WriteString(fmt.Sprintf("  finished: %s\n", time.Unix(rpt.FinishedAtUnix, 0).Format(time.RFC3339)))
	}
	if rpt.DegradedReason != "" {
		sb.WriteString(fmt.Sprintf("  degraded: %s\n", rpt.DegradedReason))
	}
	if rpt.RecommendedNextCommand != "" {
		sb.WriteString(fmt.Sprintf("  next: %s\n", rpt.RecommendedNextCommand))
	}
	return sb.String()
}

func renderSystemStatusLine(cmd *cobra.Command) string {
	snapshot, err := cliSystemEventsManager().Snapshot(cmd.Context())
	if err != nil {
		return fmt.Sprintf("system: %s reason=status_unavailable audit=%s\n", toolspkg.SystemEventCodeUnavailable, config.ToolAuditLogPath())
	}
	return toolspkg.FormatSystemStatus(snapshot, config.ToolAuditLogPath()) + "\n"
}

// collectSystemSnapshotForJSON returns a snapshot suitable for the
// JSON wire shape, normalizing nil slices to empty arrays so consumers
// can iterate without nil-checks (same convention as
// emitSessionListJSON returning `[]` for empty inventories).
func collectSystemSnapshotForJSON(cmd *cobra.Command) toolspkg.SystemEventsSnapshot {
	snapshot, err := cliSystemEventsManager().Snapshot(cmd.Context())
	if err != nil {
		return toolspkg.SystemEventsSnapshot{
			Events:   []toolspkg.SystemEvent{},
			Presence: []toolspkg.SystemPresenceEntry{},
		}
	}
	if snapshot.Events == nil {
		snapshot.Events = []toolspkg.SystemEvent{}
	}
	if snapshot.Presence == nil {
		snapshot.Presence = []toolspkg.SystemPresenceEntry{}
	}
	return snapshot
}
