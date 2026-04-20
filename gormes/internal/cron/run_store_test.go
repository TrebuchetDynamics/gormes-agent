package cron

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/gormes/internal/memory"
)

func newTestRunStore(t *testing.T) (*RunStore, *memory.SqliteStore, func()) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "memory.db")
	ms, err := memory.OpenSqlite(path, 0, nil)
	if err != nil {
		t.Fatal(err)
	}
	rs := NewRunStore(ms.DB())
	return rs, ms, func() { _ = ms.Close(context.Background()) }
}

func TestRunStore_RecordRound(t *testing.T) {
	rs, _, cleanup := newTestRunStore(t)
	defer cleanup()

	run := Run{
		JobID:             "job-1",
		StartedAt:         1700000000,
		FinishedAt:        1700000005,
		PromptHash:        "deadbeef",
		Status:            "success",
		Delivered:         true,
		SuppressionReason: "",
		OutputPreview:     "report contents",
	}
	if err := rs.RecordRun(context.Background(), run); err != nil {
		t.Fatalf("RecordRun: %v", err)
	}
	got, err := rs.LatestRuns(context.Background(), "job-1", 5)
	if err != nil {
		t.Fatalf("LatestRuns: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("got %d rows, want 1", len(got))
	}
	if got[0].Status != "success" || got[0].OutputPreview != "report contents" || !got[0].Delivered {
		t.Errorf("got = %+v, want success/delivered/preview intact", got[0])
	}
}

func TestRunStore_RecordSuppressed(t *testing.T) {
	rs, _, cleanup := newTestRunStore(t)
	defer cleanup()
	run := Run{
		JobID:             "job-1",
		StartedAt:         1,
		FinishedAt:        2,
		PromptHash:        "h",
		Status:            "suppressed",
		Delivered:         false,
		SuppressionReason: "silent",
	}
	if err := rs.RecordRun(context.Background(), run); err != nil {
		t.Fatalf("RecordRun: %v", err)
	}
}

func TestRunStore_RecordTimeoutWithErrorMsg(t *testing.T) {
	rs, _, cleanup := newTestRunStore(t)
	defer cleanup()
	run := Run{
		JobID:      "job-1",
		StartedAt:  1,
		FinishedAt: 61,
		PromptHash: "h",
		Status:     "timeout",
		Delivered:  true,
		ErrorMsg:   "deadline exceeded after 60s",
	}
	if err := rs.RecordRun(context.Background(), run); err != nil {
		t.Fatalf("RecordRun: %v", err)
	}
}

func TestRunStore_LatestRunsOrdersByStartedDesc(t *testing.T) {
	rs, _, cleanup := newTestRunStore(t)
	defer cleanup()
	for _, s := range []int64{3, 1, 5, 2, 4} {
		_ = rs.RecordRun(context.Background(), Run{
			JobID: "j", StartedAt: s, PromptHash: "h", Status: "success", Delivered: true,
		})
	}
	got, _ := rs.LatestRuns(context.Background(), "j", 3)
	if len(got) != 3 {
		t.Fatalf("len = %d, want 3", len(got))
	}
	if got[0].StartedAt != 5 || got[1].StartedAt != 4 || got[2].StartedAt != 3 {
		t.Errorf("order = %v, want 5,4,3", []int64{got[0].StartedAt, got[1].StartedAt, got[2].StartedAt})
	}
}

func TestRunStore_RejectsInvalidStatus(t *testing.T) {
	rs, _, cleanup := newTestRunStore(t)
	defer cleanup()
	err := rs.RecordRun(context.Background(), Run{
		JobID: "j", StartedAt: 1, PromptHash: "h", Status: "bogus",
	})
	if err == nil {
		t.Error("RecordRun with status='bogus' should fail CHECK constraint")
	}
}
