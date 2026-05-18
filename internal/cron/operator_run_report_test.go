package cron

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	agentruntime "github.com/TrebuchetDynamics/gormes-agent/internal/runtime"
)

func TestOperatorRunReportBuildsSuccessAndDegradedArtifacts(t *testing.T) {
	home := t.TempDir()
	workspace := filepath.Join(home, "workspace", "ops")
	job := Job{
		ID:       "job-alpha",
		Name:     "Morning briefing",
		Schedule: "@daily",
		Deliver:  "local",
		Provider: "openai",
		Model:    "gpt-4.1-mini",
		Workdir:  workspace,
	}

	t.Run("success", func(t *testing.T) {
		run := Run{
			ID:            42,
			JobID:         job.ID,
			StartedAt:     1700000000,
			FinishedAt:    1700000015,
			PromptHash:    "deadbeef",
			Status:        "success",
			Delivered:     true,
			OutputPreview: "briefing ready",
		}
		report := BuildOperatorRunReport(OperatorRunReportInput{
			Job:             job,
			Run:             run,
			Profile:         "ops",
			Workspace:       workspace,
			HomeDir:         home,
			RuntimeEvidence: agentruntime.StatusEvidence(agentruntime.Binding{Provider: job.Provider, Model: job.Model, EndpointSource: agentruntime.EndpointSourceNativeProvider}),
			DeliveryPlan: DeliveryPlan{Targets: []DeliveryTarget{
				{Platform: "local", Local: true},
			}},
			DeliveryOutcome: DeliveryOutcome{Delivered: true},
			SessionID:       "cron:job-alpha:1700000000",
			TranscriptRefs:  []string{"session:sess-briefing", filepath.Join(home, "transcripts", "sess-briefing.json")},
			ReleaseEvidence: []ReleaseEvidence{{Code: ReleaseEvidenceSessionDBClosed, Label: "session-db"}},
		})

		if report.SchemaVersion != OperatorRunReportSchemaVersion {
			t.Fatalf("SchemaVersion = %q, want %q", report.SchemaVersion, OperatorRunReportSchemaVersion)
		}
		if report.JobID != job.ID || report.RunID != run.ID || report.Profile != "ops" {
			t.Fatalf("identity = %+v, want job/run/profile populated", report)
		}
		if report.Provider != job.Provider || report.Model != job.Model {
			t.Fatalf("provider/model = %q/%q, want %q/%q", report.Provider, report.Model, job.Provider, job.Model)
		}
		if report.Status != OperatorRunReportStatusSuccess || report.DegradedReason != "" {
			t.Fatalf("status = %q degraded_reason=%q, want clean success", report.Status, report.DegradedReason)
		}
		if report.StartedAtUnix != run.StartedAt || report.FinishedAtUnix != run.FinishedAt {
			t.Fatalf("timestamps = %d/%d, want %d/%d", report.StartedAtUnix, report.FinishedAtUnix, run.StartedAt, run.FinishedAt)
		}
		if got := report.DeliveryTargets[0].Target; got != "local" {
			t.Fatalf("delivery target = %q, want local", got)
		}
		if report.SessionID != "cron:job-alpha:1700000000" || len(report.TranscriptRefs) != 2 {
			t.Fatalf("session/transcript refs = %q/%v", report.SessionID, report.TranscriptRefs)
		}
		if report.RecommendedNextCommand != "gormes cron status job-alpha" {
			t.Fatalf("RecommendedNextCommand = %q", report.RecommendedNextCommand)
		}

		path := OperatorRunReportPath(home, report)
		if want := filepath.Join(home, "operator-runs", job.ID, "42.json"); path != want {
			t.Fatalf("OperatorRunReportPath = %q, want %q", path, want)
		}
		if err := WriteOperatorRunReport(path, report); err != nil {
			t.Fatalf("WriteOperatorRunReport: %v", err)
		}
		read, err := ReadOperatorRunReport(path)
		if err != nil {
			t.Fatalf("ReadOperatorRunReport: %v", err)
		}
		if !reflect.DeepEqual(read, report) {
			t.Fatalf("round trip mismatch\nread=%+v\nwant=%+v", read, report)
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read raw report: %v", err)
		}
		if !json.Valid(raw) {
			t.Fatalf("report is not valid JSON:\n%s", raw)
		}
		if strings.Contains(string(raw), home) {
			t.Fatalf("report leaked home path %q:\n%s", home, raw)
		}
	})

	t.Run("degraded", func(t *testing.T) {
		secret := "sk-operatorreportsecret123456"
		cases := []struct {
			name        string
			run         Run
			runtime     map[string]any
			plan        DeliveryPlan
			outcome     DeliveryOutcome
			wantStatus  string
			wantReason  string
			wantCommand string
		}{
			{
				name: "provider auth missing",
				run:  Run{ID: 51, JobID: job.ID, StartedAt: 1700000100, FinishedAt: 1700000102, Status: "error", ErrorMsg: "provider failed with OPENAI_API_KEY=" + secret + " at " + filepath.Join(home, "config.toml")},
				runtime: agentruntime.StatusEvidence(agentruntime.Binding{
					Provider:        "openai",
					Model:           "gpt-4.1-mini",
					EndpointSource:  agentruntime.EndpointSourceUnconfigured,
					DegradedReasons: []string{agentruntime.DegradedReasonProviderConfigMissing, agentruntime.DegradedReasonNativeRuntimeUnavailable},
				}),
				wantStatus:  OperatorRunReportStatusFailed,
				wantReason:  OperatorRunReportReasonProviderAuthUnready,
				wantCommand: "gormes doctor --offline",
			},
			{
				name:        "timeout",
				run:         Run{ID: 52, JobID: job.ID, StartedAt: 1700000200, FinishedAt: 1700000260, Status: "timeout", ErrorMsg: "deadline exceeded after 60s"},
				wantStatus:  OperatorRunReportStatusFailed,
				wantReason:  OperatorRunReportReasonTimeout,
				wantCommand: "gormes cron status job-alpha",
			},
			{
				name:        "suppressed",
				run:         Run{ID: 53, JobID: job.ID, StartedAt: 1700000300, FinishedAt: 1700000301, Status: "suppressed", SuppressionReason: "silent"},
				wantStatus:  OperatorRunReportStatusDegraded,
				wantReason:  OperatorRunReportReasonSuppressed,
				wantCommand: "gormes cron status job-alpha",
			},
			{
				name: "delivery failed",
				run:  Run{ID: 54, JobID: job.ID, StartedAt: 1700000400, FinishedAt: 1700000405, Status: "success", Delivered: false, OutputPreview: "briefing ready"},
				plan: DeliveryPlan{Targets: []DeliveryTarget{
					{Platform: "slack", ChatID: "C123", ThreadID: "T456", Explicit: true},
				}},
				outcome: DeliveryOutcome{
					Delivered: false,
					Evidence: []DeliveryEvidence{{
						Code:   DeliveryEvidenceLiveAdapterUnavailable,
						Target: "slack:C123:T456",
						Detail: "live adapter returned Bearer " + secret + " from " + filepath.Join(home, "gateway.log"),
					}},
					Err: errors.New("cron delivery fallback sink unavailable"),
				},
				wantStatus:  OperatorRunReportStatusDegraded,
				wantReason:  OperatorRunReportReasonDeliveryFailed,
				wantCommand: "gormes gateway status --json",
			},
		}

		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				report := BuildOperatorRunReport(OperatorRunReportInput{
					Job:             job,
					Run:             tc.run,
					Profile:         "ops",
					Workspace:       workspace,
					HomeDir:         home,
					RuntimeEvidence: tc.runtime,
					DeliveryPlan:    tc.plan,
					DeliveryOutcome: tc.outcome,
					SessionID:       "cron:job-alpha:1700000000",
					TranscriptRefs:  []string{filepath.Join(home, "transcripts", "degraded.json")},
				})

				if report.Status != tc.wantStatus || report.DegradedReason != tc.wantReason {
					t.Fatalf("status/reason = %q/%q, want %q/%q", report.Status, report.DegradedReason, tc.wantStatus, tc.wantReason)
				}
				if report.RecommendedNextCommand != tc.wantCommand {
					t.Fatalf("RecommendedNextCommand = %q, want %q", report.RecommendedNextCommand, tc.wantCommand)
				}
				raw, err := json.Marshal(report)
				if err != nil {
					t.Fatalf("marshal report: %v", err)
				}
				for _, forbidden := range []string{secret, home} {
					if strings.Contains(string(raw), forbidden) {
						t.Fatalf("report leaked %q:\n%s", forbidden, raw)
					}
				}
			})
		}
	})
}
