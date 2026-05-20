package cron

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestScheduledBriefingWritesOperatorRunReport(t *testing.T) {
	t.Run("provider backed success", func(t *testing.T) {
		fk := newFakeKernel("Morning briefing: all systems nominal.", 0)
		e, _, cleanup := newTestExecutorEnv(t, fk)
		defer cleanup()
		e.cfg.OperatorReportHome = t.TempDir()

		job := NewJob("morning-briefing", "0 8 * * *", "summarize fleet status")
		job.Name = "Morning briefing"
		job.Provider = "openai"
		job.Model = "gpt-4.1-mini"
		if err := e.cfg.JobStore.Create(job); err != nil {
			t.Fatalf("Create job: %v", err)
		}

		e.Run(context.Background(), job)

		report := latestOperatorReportForJob(t, e.cfg.OperatorReportHome, job.ID)
		if report.JobID != job.ID || report.JobName != job.Name {
			t.Fatalf("report identity = %+v, want job %q/%q", report, job.ID, job.Name)
		}
		if report.Status != OperatorRunReportStatusSuccess || report.DegradedReason != "" {
			t.Fatalf("report status/reason = %q/%q, want success", report.Status, report.DegradedReason)
		}
		if report.RunID == 0 || report.StartedAtUnix == 0 || report.FinishedAtUnix == 0 {
			t.Fatalf("report missing run timestamps/id: %+v", report)
		}
		if report.SessionID == "" || !strings.HasPrefix(report.SessionID, "cron:"+job.ID+":") {
			t.Fatalf("SessionID = %q, want cron job session id for %q", report.SessionID, job.ID)
		}
		if report.OutputSummary != "Morning briefing: all systems nominal." {
			t.Fatalf("OutputSummary = %q", report.OutputSummary)
		}
		if report.Provider != "openai" || report.Model != "gpt-4.1-mini" {
			t.Fatalf("provider/model = %q/%q", report.Provider, report.Model)
		}
	})

	t.Run("kernel submit error", func(t *testing.T) {
		e, _, cleanup := newTestExecutorEnv(t, &erroringKernel{err: errors.New("mailbox full")})
		defer cleanup()
		e.cfg.OperatorReportHome = t.TempDir()

		job := NewJob("submit-error", "@daily", "status")
		if err := e.cfg.JobStore.Create(job); err != nil {
			t.Fatalf("Create job: %v", err)
		}

		e.Run(context.Background(), job)

		report := latestOperatorReportForJob(t, e.cfg.OperatorReportHome, job.ID)
		if report.Status != OperatorRunReportStatusFailed || report.DegradedReason != OperatorRunReportReasonCronError {
			t.Fatalf("report status/reason = %q/%q, want failed cron_error", report.Status, report.DegradedReason)
		}
		if !strings.Contains(report.ErrorSummary, "mailbox full") {
			t.Fatalf("ErrorSummary = %q, want mailbox full", report.ErrorSummary)
		}
	})

	t.Run("delivery failure", func(t *testing.T) {
		fk := newFakeKernel("Briefing ready but delivery fails.", 0)
		e, _, cleanup := newTestExecutorEnv(t, fk)
		defer cleanup()
		e.cfg.OperatorReportHome = t.TempDir()
		e.cfg.Sink = FuncSink(func(context.Context, string) error { return errors.New("fallback sink unavailable") })

		job := NewJob("delivery-failure", "@daily", "status")
		if err := e.cfg.JobStore.Create(job); err != nil {
			t.Fatalf("Create job: %v", err)
		}

		e.Run(context.Background(), job)

		report := latestOperatorReportForJob(t, e.cfg.OperatorReportHome, job.ID)
		if report.Status != OperatorRunReportStatusDegraded || report.DegradedReason != OperatorRunReportReasonDeliveryFailed {
			t.Fatalf("report status/reason = %q/%q, want degraded delivery_failed", report.Status, report.DegradedReason)
		}
		if !strings.Contains(report.ErrorSummary, "fallback sink unavailable") {
			t.Fatalf("ErrorSummary = %q, want delivery failure evidence", report.ErrorSummary)
		}
	})

	t.Run("suppressed silent response", func(t *testing.T) {
		fk := newFakeKernel("[SILENT]", 0)
		e, _, cleanup := newTestExecutorEnv(t, fk)
		defer cleanup()
		e.cfg.OperatorReportHome = t.TempDir()

		job := NewJob("quiet-briefing", "@daily", "status")
		if err := e.cfg.JobStore.Create(job); err != nil {
			t.Fatalf("Create job: %v", err)
		}

		e.Run(context.Background(), job)

		report := latestOperatorReportForJob(t, e.cfg.OperatorReportHome, job.ID)
		if report.Status != OperatorRunReportStatusDegraded || report.DegradedReason != OperatorRunReportReasonSuppressed {
			t.Fatalf("report status/reason = %q/%q, want degraded suppressed", report.Status, report.DegradedReason)
		}
	})

	t.Run("timeout", func(t *testing.T) {
		fk := newFakeKernel("too late", time.Second)
		e, _, cleanup := newTestExecutorEnv(t, fk)
		defer cleanup()
		e.cfg.OperatorReportHome = t.TempDir()
		e.cfg.CallTimeout = 10 * time.Millisecond

		job := NewJob("slow-briefing", "@daily", "status")
		if err := e.cfg.JobStore.Create(job); err != nil {
			t.Fatalf("Create job: %v", err)
		}

		e.Run(context.Background(), job)

		report := latestOperatorReportForJob(t, e.cfg.OperatorReportHome, job.ID)
		if report.Status != OperatorRunReportStatusFailed || report.DegradedReason != OperatorRunReportReasonTimeout {
			t.Fatalf("report status/reason = %q/%q, want failed timeout", report.Status, report.DegradedReason)
		}
		if !strings.Contains(report.ErrorSummary, "context deadline exceeded") {
			t.Fatalf("ErrorSummary = %q, want deadline evidence", report.ErrorSummary)
		}
	})

	t.Run("no agent script only", func(t *testing.T) {
		fk := newFakeKernel("should not run", 0)
		e, _, cleanup := newTestExecutorEnv(t, fk)
		defer cleanup()
		e.cfg.OperatorReportHome = t.TempDir()
		e.cfg.ScriptRunner = CronScriptRunnerFunc(func(context.Context, CronScriptRequest) CronScriptResult {
			return CronScriptResult{Success: true, Output: "Disk 72%"}
		})

		job := NewJob("script-briefing", "@daily", "")
		job.NoAgent = true
		job.Script = "disk.sh"
		if err := e.cfg.JobStore.Create(job); err != nil {
			t.Fatalf("Create job: %v", err)
		}

		e.Run(context.Background(), job)

		if len(fk.events) != 0 {
			t.Fatalf("kernel events = %d, want no submit for no_agent", len(fk.events))
		}
		report := latestOperatorReportForJob(t, e.cfg.OperatorReportHome, job.ID)
		if report.Status != OperatorRunReportStatusSuccess || report.OutputSummary != "Disk 72%" {
			t.Fatalf("report = %+v, want success with script output", report)
		}
	})
}

func latestOperatorReportForJob(t *testing.T, home, jobID string) OperatorRunReport {
	t.Helper()
	entries, err := os.ReadDir(filepath.Join(home, "operator-runs", jobID))
	if err != nil {
		t.Fatalf("read operator report dir: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("operator report files = %d, want 1", len(entries))
	}
	report, err := ReadOperatorRunReport(filepath.Join(home, "operator-runs", jobID, entries[0].Name()))
	if err != nil {
		t.Fatalf("ReadOperatorRunReport: %v", err)
	}
	return report
}
