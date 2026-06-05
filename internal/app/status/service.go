package status

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/TrebuchetDynamics/gormes-agent/internal/automation/cron"
	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli"
	toolspkg "github.com/TrebuchetDynamics/gormes-agent/internal/tools"
)

type BuildProvenance struct {
	Version   string `json:"version"`
	GitCommit string `json:"git_commit"`
}

type SnapshotFunc func(context.Context) (toolspkg.SystemEventsSnapshot, error)

type Options struct {
	BuildProvenance func() BuildProvenance
	SystemSnapshot  SnapshotFunc
	AuditPath       func() string
}

func (o Options) buildProvenance() BuildProvenance {
	if o.BuildProvenance != nil {
		return o.BuildProvenance()
	}
	return BuildProvenance{}
}

func (o Options) auditPath() string {
	if o.AuditPath != nil {
		return o.AuditPath()
	}
	return config.ToolAuditLogPath()
}

func NewCommand(options Options) *cobra.Command {
	var progressPath string
	var operatorReportDir string
	var asJSON bool
	cmd := &cobra.Command{
		Use:          "status",
		Short:        "Show Gormes runtime and progress blockers",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if asJSON {
				return RunJSON(cmd, progressPath, operatorReportDir, options)
			}
			return RunText(cmd, progressPath, operatorReportDir, options)
		},
	}
	cmd.Flags().StringVar(&progressPath, "progress", cli.DefaultStatusProgressPath, "progress.json path used for blocker status")
	cmd.Flags().StringVar(&operatorReportDir, "operator-report-dir", DefaultOperatorReportDir(), "directory containing operator run reports")
	cmd.Flags().BoolVar(&asJSON, "json", false, "emit a machine-readable {blockers: [...]} JSON document (suitable for monitoring/automation)")
	return cmd
}

type ReportJSON struct {
	Build             BuildProvenance               `json:"build"`
	Progress          *ProgressJSON                 `json:"progress,omitempty"`
	Blockers          []cli.StatusBlocker           `json:"blockers"`
	System            toolspkg.SystemEventsSnapshot `json:"system"`
	AuditPath         string                        `json:"audit_path"`
	OperatorRunReport *OperatorRunReportJSON        `json:"operator_run_report,omitempty"`
}

type OperatorRunReportJSON struct {
	Status                 string `json:"status"`
	JobID                  string `json:"job_id,omitempty"`
	RunID                  int64  `json:"run_id,omitempty"`
	Profile                string `json:"profile,omitempty"`
	StartedAtUnix          int64  `json:"started_at_unix,omitempty"`
	FinishedAtUnix         int64  `json:"finished_at_unix,omitempty"`
	DegradedReason         string `json:"degraded_reason,omitempty"`
	RecommendedNextCommand string `json:"recommended_next_command,omitempty"`
}

type ProgressJSON struct {
	Status string `json:"status"`
	Reason string `json:"reason,omitempty"`
}

func DefaultOperatorReportDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".gormes")
}

func FindLatestOperatorReport(dir string) (cron.OperatorRunReport, error) {
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

func RunJSON(cmd *cobra.Command, progressPath, operatorReportDir string, options Options) error {
	blockers, err := cli.CollectStatusBlockers(cli.StatusReportOptions{ProgressPath: progressPath})
	var progressUnavailable *ProgressJSON
	if err != nil {
		progressUnavailable = &ProgressJSON{Status: "progress_unavailable", Reason: err.Error()}
		blockers = nil
	}
	if blockers == nil {
		blockers = []cli.StatusBlocker{}
	}

	report := ReportJSON{
		Build:     options.buildProvenance(),
		Progress:  progressUnavailable,
		Blockers:  blockers,
		System:    CollectSystemSnapshotForJSON(cmd.Context(), options),
		AuditPath: options.auditPath(),
	}

	if rpt, err := FindLatestOperatorReport(operatorReportDir); err == nil {
		report.OperatorRunReport = &OperatorRunReportJSON{
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

func RunText(cmd *cobra.Command, progressPath, operatorReportDir string, options Options) error {
	report, err := cli.RenderStatusReport(cli.StatusReportOptions{ProgressPath: progressPath})
	if err != nil {
		return err
	}

	out := report + RenderSystemStatusLine(cmd.Context(), options)
	if rpt, err := FindLatestOperatorReport(operatorReportDir); err == nil {
		out += FormatOperatorRunReportText(rpt)
	} else {
		out += "operator run report: operator_report_unavailable\n"
	}
	_, err = fmt.Fprint(cmd.OutOrStdout(), out)
	return err
}

func FormatOperatorRunReportText(rpt cron.OperatorRunReport) string {
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

func RenderSystemStatusLine(ctx context.Context, options Options) string {
	snapshot, err := systemSnapshot(ctx, options)
	if err != nil {
		return fmt.Sprintf("system: %s reason=status_unavailable audit=%s\n", toolspkg.SystemEventCodeUnavailable, options.auditPath())
	}
	return toolspkg.FormatSystemStatus(snapshot, options.auditPath()) + "\n"
}

func CollectSystemSnapshotForJSON(ctx context.Context, options Options) toolspkg.SystemEventsSnapshot {
	snapshot, err := systemSnapshot(ctx, options)
	if err != nil {
		return toolspkg.SystemEventsSnapshot{Events: []toolspkg.SystemEvent{}, Presence: []toolspkg.SystemPresenceEntry{}}
	}
	if snapshot.Events == nil {
		snapshot.Events = []toolspkg.SystemEvent{}
	}
	if snapshot.Presence == nil {
		snapshot.Presence = []toolspkg.SystemPresenceEntry{}
	}
	return snapshot
}

func systemSnapshot(ctx context.Context, options Options) (toolspkg.SystemEventsSnapshot, error) {
	if options.SystemSnapshot == nil {
		return toolspkg.SystemEventsSnapshot{}, fmt.Errorf("system snapshot unavailable")
	}
	return options.SystemSnapshot(ctx)
}
