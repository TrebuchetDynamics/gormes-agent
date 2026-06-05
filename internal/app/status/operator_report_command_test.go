package status

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/spf13/cobra"

	"github.com/TrebuchetDynamics/gormes-agent/internal/automation/cron"
	toolspkg "github.com/TrebuchetDynamics/gormes-agent/internal/tools"
)

func newOperatorReportStatusCommand(t *testing.T) *cobra.Command {
	t.Helper()
	auditPath := filepath.Join(t.TempDir(), "audit.log")
	return NewCommand(Options{
		BuildProvenance: func() BuildProvenance {
			return BuildProvenance{Version: "test-version", GitCommit: "test-sha"}
		},
		SystemSnapshot: func(context.Context) (toolspkg.SystemEventsSnapshot, error) {
			return toolspkg.SystemEventsSnapshot{}, nil
		},
		AuditPath: func() string { return auditPath },
	})
}

func TestStatusRendersLatestOperatorRunReport(t *testing.T) {
	progressPath := writeOperatorReportProgressFixture(t)
	tmp := t.TempDir()
	reportDir := filepath.Join(tmp, "operator-runs", "test-job")
	if err := os.MkdirAll(reportDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	report := cron.OperatorRunReport{
		SchemaVersion:          cron.OperatorRunReportSchemaVersion,
		JobID:                  "test-job",
		RunID:                  42,
		Status:                 cron.OperatorRunReportStatusDegraded,
		DegradedReason:         cron.OperatorRunReportReasonProviderAuthUnready,
		RecommendedNextCommand: "gormes doctor --offline",
		StartedAtUnix:          time.Now().Add(-1 * time.Hour).Unix(),
		FinishedAtUnix:         time.Now().Unix(),
	}
	reportPath := filepath.Join(reportDir, "42.json")
	if err := cron.WriteOperatorRunReport(reportPath, report); err != nil {
		t.Fatalf("write report: %v", err)
	}

	cmd := newOperatorReportStatusCommand(t)
	cmd.SetArgs([]string{"--progress", progressPath, "--operator-report-dir", tmp})
	var out strings.Builder
	cmd.SetOut(&out)
	cmd.SetErr(&out)

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}

	output := out.String()
	if !strings.Contains(output, "test-job") {
		t.Errorf("status output missing job ID: %s", output)
	}
	if !strings.Contains(output, "degraded") {
		t.Errorf("status output missing degraded status: %s", output)
	}
	if !strings.Contains(output, "gormes doctor --offline") {
		t.Errorf("status output missing recommended command: %s", output)
	}
}

func TestStatusJSONIncludesOperatorRunReport(t *testing.T) {
	progressPath := writeOperatorReportProgressFixture(t)
	tmp := t.TempDir()
	reportDir := filepath.Join(tmp, "operator-runs", "json-job")
	if err := os.MkdirAll(reportDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	report := cron.OperatorRunReport{
		SchemaVersion:          cron.OperatorRunReportSchemaVersion,
		JobID:                  "json-job",
		RunID:                  99,
		Profile:                "main",
		Status:                 cron.OperatorRunReportStatusSuccess,
		StartedAtUnix:          time.Now().Add(-30 * time.Minute).Unix(),
		FinishedAtUnix:         time.Now().Unix(),
		RecommendedNextCommand: "gormes cron list",
	}
	reportPath := filepath.Join(reportDir, "99.json")
	if err := cron.WriteOperatorRunReport(reportPath, report); err != nil {
		t.Fatalf("write report: %v", err)
	}

	cmd := newOperatorReportStatusCommand(t)
	cmd.SetArgs([]string{"--json", "--progress", progressPath, "--operator-report-dir", tmp})
	var out strings.Builder
	cmd.SetOut(&out)
	cmd.SetErr(&out)

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}

	var data map[string]any
	if err := json.Unmarshal([]byte(out.String()), &data); err != nil {
		t.Fatalf("parse JSON: %v", err)
	}

	opRun, ok := data["operator_run_report"].(map[string]any)
	if !ok {
		t.Fatalf("missing operator_run_report in JSON output: %s", out.String())
	}
	if opRun["job_id"] != "json-job" {
		t.Errorf("job_id=%v, want json-job", opRun["job_id"])
	}
	if opRun["status"] != "success" {
		t.Errorf("status=%v, want success", opRun["status"])
	}
}

func TestStatusMissingOperatorRunReport(t *testing.T) {
	progressPath := writeOperatorReportProgressFixture(t)
	tmp := t.TempDir()

	cmd := newOperatorReportStatusCommand(t)
	cmd.SetArgs([]string{"--progress", progressPath, "--operator-report-dir", tmp})
	var out strings.Builder
	cmd.SetOut(&out)
	cmd.SetErr(&out)

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}

	output := out.String()
	if !strings.Contains(output, "operator_report_unavailable") {
		t.Errorf("status output should indicate unavailable report: %s", output)
	}
}

func TestStatusMalformedOperatorRunReport(t *testing.T) {
	progressPath := writeOperatorReportProgressFixture(t)
	tmp := t.TempDir()
	reportDir := filepath.Join(tmp, "operator-runs", "bad-job")
	if err := os.MkdirAll(reportDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	if err := os.WriteFile(filepath.Join(reportDir, "1.json"), []byte("{bad json"), 0o644); err != nil {
		t.Fatalf("write bad report: %v", err)
	}

	cmd := newOperatorReportStatusCommand(t)
	cmd.SetArgs([]string{"--progress", progressPath, "--operator-report-dir", tmp})
	var out strings.Builder
	cmd.SetOut(&out)
	cmd.SetErr(&out)

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}

	output := out.String()
	if !strings.Contains(output, "operator_report_unavailable") {
		t.Errorf("status output should indicate unavailable report for malformed data: %s", output)
	}
}

func writeOperatorReportProgressFixture(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "progress.json")
	body := `{
  "meta": {"version": "2.0", "links": {"github_readme": "", "landing_page": "", "docs_site": "", "source_code": ""}},
  "phases": {}
}`
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("write progress fixture: %v", err)
	}
	return path
}
