package cron

import (
	"context"
	"strings"
	"testing"
)

func TestCronNoAgent_DeliversScriptStdoutWithoutKernelSubmit(t *testing.T) {
	fk := newFakeKernel("should not run", 0)
	e, deliveries, cleanup := newTestExecutorEnv(t, fk)
	defer cleanup()
	e.cfg.ScriptRunner = CronScriptRunnerFunc(func(context.Context, CronScriptRequest) CronScriptResult {
		return CronScriptResult{Success: true, Output: "RAM 92% on host"}
	})

	job := NewJob("watchdog", "every 5m", "")
	job.Script = "watchdog.sh"
	job.NoAgent = true
	if err := e.cfg.JobStore.Create(job); err != nil {
		t.Fatalf("Create job: %v", err)
	}

	e.Run(context.Background(), job)

	if len(fk.events) != 0 {
		t.Fatalf("kernel events = %d, want no agent submit for no_agent job", len(fk.events))
	}
	got := deliveries.Load().([]string)
	if len(got) != 1 || got[0] != "RAM 92% on host" {
		t.Fatalf("deliveries = %#v, want verbatim script stdout", got)
	}
	runs, err := e.cfg.RunStore.LatestRuns(context.Background(), job.ID, 5)
	if err != nil {
		t.Fatalf("LatestRuns: %v", err)
	}
	if len(runs) != 1 || runs[0].Status != "success" || !runs[0].Delivered {
		t.Fatalf("run = %+v, want success delivered", runs)
	}
}

func TestCronNoAgent_SilentOutputsDoNotDeliverOrSubmit(t *testing.T) {
	tests := []struct {
		name   string
		output string
		reason string
	}{
		{name: "empty stdout", output: " \n\t ", reason: "empty"},
		{name: "wakeAgent false", output: `{"wakeAgent": false}`, reason: "silent"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			fk := newFakeKernel("should not run", 0)
			e, deliveries, cleanup := newTestExecutorEnv(t, fk)
			defer cleanup()
			e.cfg.ScriptRunner = CronScriptRunnerFunc(func(context.Context, CronScriptRequest) CronScriptResult {
				return CronScriptResult{Success: true, Output: tc.output}
			})

			job := NewJob("quiet", "every 5m", "")
			job.Script = "quiet.sh"
			job.NoAgent = true
			if err := e.cfg.JobStore.Create(job); err != nil {
				t.Fatalf("Create job: %v", err)
			}

			e.Run(context.Background(), job)

			if len(fk.events) != 0 {
				t.Fatalf("kernel events = %d, want no agent submit for no_agent job", len(fk.events))
			}
			if got := deliveries.Load().([]string); len(got) != 0 {
				t.Fatalf("deliveries = %#v, want none", got)
			}
			runs, err := e.cfg.RunStore.LatestRuns(context.Background(), job.ID, 5)
			if err != nil {
				t.Fatalf("LatestRuns: %v", err)
			}
			if len(runs) != 1 || runs[0].Status != "suppressed" || runs[0].Delivered || runs[0].SuppressionReason != tc.reason {
				t.Fatalf("run = %+v, want suppressed %q without delivery", runs, tc.reason)
			}
		})
	}
}

func TestCronNoAgent_ScriptFailureAlertsWithoutKernelSubmit(t *testing.T) {
	tests := []struct {
		name       string
		output     string
		wantStatus string
		wantText   string
	}{
		{name: "exit failure", output: "Script exited with code 3\noops token=[REDACTED]", wantStatus: "error", wantText: "Cron watchdog"},
		{name: "timeout", output: "Script timed out after 30s: watchdog.sh", wantStatus: "timeout", wantText: "timed out"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			fk := newFakeKernel("should not run", 0)
			e, deliveries, cleanup := newTestExecutorEnv(t, fk)
			defer cleanup()
			e.cfg.ScriptRunner = CronScriptRunnerFunc(func(context.Context, CronScriptRequest) CronScriptResult {
				return CronScriptResult{Success: false, Output: tc.output}
			})

			job := NewJob("broken", "every 5m", "")
			job.Script = "broken.sh"
			job.NoAgent = true
			if err := e.cfg.JobStore.Create(job); err != nil {
				t.Fatalf("Create job: %v", err)
			}

			e.Run(context.Background(), job)

			if len(fk.events) != 0 {
				t.Fatalf("kernel events = %d, want no agent submit for no_agent job", len(fk.events))
			}
			got := deliveries.Load().([]string)
			if len(got) != 1 || !strings.Contains(got[0], tc.wantText) {
				t.Fatalf("deliveries = %#v, want alert containing %q", got, tc.wantText)
			}
			if strings.Contains(got[0], "token=plain-secret") {
				t.Fatalf("delivery leaked unredacted secret: %s", got[0])
			}
			runs, err := e.cfg.RunStore.LatestRuns(context.Background(), job.ID, 5)
			if err != nil {
				t.Fatalf("LatestRuns: %v", err)
			}
			if len(runs) != 1 || runs[0].Status != tc.wantStatus || !runs[0].Delivered {
				t.Fatalf("run = %+v, want %s delivered", runs, tc.wantStatus)
			}
		})
	}
}
