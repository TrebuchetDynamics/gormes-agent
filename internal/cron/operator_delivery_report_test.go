package cron

import (
	"errors"
	"testing"
)

func TestOperatorRunReportIncludesDeliveryEvidence(t *testing.T) {
	job := NewJob("delivery-report", "@daily", "brief ops")
	job.Deliver = "telegram:-100123:42,google_chat:spaces/AAA:spaces/AAA/threads/thread-1,slack:C123:T456"
	run := Run{ID: 7, JobID: job.ID, Status: "success", Delivered: true, OutputPreview: "briefing ready"}
	plan := DeliveryPlan{Targets: []DeliveryTarget{
		{Platform: "telegram", ChatID: "-100123", ThreadID: "42", Explicit: true},
		{Platform: "google_chat", ChatID: "spaces/AAA", ThreadID: "spaces/AAA/threads/thread-1", Explicit: true},
		{Platform: "slack", ChatID: "C123", ThreadID: "T456", Explicit: true},
	}}
	outcome := DeliveryOutcome{Delivered: true, Evidence: []DeliveryEvidence{
		{Code: DeliveryEvidenceLiveAdapterUnavailable, Target: "google_chat:spaces/AAA:spaces/AAA/threads/thread-1", Detail: "live adapter unavailable"},
		{Code: DeliveryEvidenceStandaloneSenderUsed, Target: "google_chat:spaces/AAA:spaces/AAA/threads/thread-1", Detail: "standalone sender used"},
		{Code: DeliveryEvidenceLiveAdapterUnavailable, Target: "slack:C123:T456", Detail: "live adapter unavailable"},
		{Code: DeliveryEvidenceFallbackSinkUsed, Target: "slack:C123:T456", Detail: "existing cron delivery sink used"},
	}}

	report := BuildOperatorRunReport(OperatorRunReportInput{Job: job, Run: run, DeliveryPlan: plan, DeliveryOutcome: outcome})

	if report.Status != OperatorRunReportStatusSuccess {
		t.Fatalf("Status = %q, want success", report.Status)
	}
	assertDeliveryResult(t, report, "telegram:-100123:42", true, "live_adapter", "")
	assertDeliveryResult(t, report, "google_chat:spaces/AAA:spaces/AAA/threads/thread-1", true, "standalone_sender", "")
	assertDeliveryResult(t, report, "slack:C123:T456", true, "live_adapter", "fallback_sink")

	failed := BuildOperatorRunReport(OperatorRunReportInput{
		Job:          job,
		Run:          Run{ID: 8, JobID: job.ID, Status: "success", Delivered: false, OutputPreview: "briefing ready"},
		DeliveryPlan: DeliveryPlan{Targets: []DeliveryTarget{{Platform: "discord", ChatID: "ops", Explicit: true}}},
		DeliveryOutcome: DeliveryOutcome{Delivered: false, Evidence: []DeliveryEvidence{
			{Code: DeliveryEvidenceLiveAdapterUnavailable, Target: "discord:ops", Detail: "live adapter unavailable"},
			{Code: DeliveryEvidenceStandaloneSenderFailed, Target: "discord:ops", Detail: "standalone sender failed"},
		}, Err: errors.New("cron delivery fallback sink unavailable")},
	})
	if failed.Status != OperatorRunReportStatusDegraded || failed.DegradedReason != OperatorRunReportReasonDeliveryFailed {
		t.Fatalf("failed status/reason = %q/%q, want degraded/delivery_failed", failed.Status, failed.DegradedReason)
	}
	assertDeliveryResult(t, failed, "discord:ops", false, "live_adapter", "")
	if failed.RecommendedNextCommand != "gormes gateway status --json" {
		t.Fatalf("RecommendedNextCommand = %q, want gateway repair command", failed.RecommendedNextCommand)
	}
}

func assertDeliveryResult(t *testing.T, report OperatorRunReport, target string, delivered bool, path string, fallbackPath string) {
	t.Helper()
	for _, result := range report.DeliveryResults {
		if result.Target != target {
			continue
		}
		if result.Delivered != delivered || result.Path != path || result.FallbackPath != fallbackPath {
			t.Fatalf("delivery result for %s = %+v, want delivered=%v path=%q fallback=%q", target, result, delivered, path, fallbackPath)
		}
		return
	}
	t.Fatalf("delivery result for %s missing in %+v", target, report.DeliveryResults)
}
