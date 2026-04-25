package subagent

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"
)

func TestDurableLedgerPausesWaitingJobWithOperatorIntentEvidence(t *testing.T) {
	ledger, _, cleanup := newTestDurableLedger(t)
	defer cleanup()

	ctx := context.Background()
	if _, err := ledger.Submit(ctx, DurableJobSubmission{
		ID:       "cron:pause-waiting",
		Kind:     WorkKindCronJob,
		Progress: json.RawMessage(`{"phase":"queued"}`),
	}); err != nil {
		t.Fatalf("Submit: %v", err)
	}

	requestedAt := time.Now().UTC().Add(-time.Second).Truncate(0)
	paused, ok, err := ledger.Pause(ctx, "cron:pause-waiting", DurableLifecycleIntent{
		Trust:       TrustOperator,
		Actor:       "operator:ada",
		Reason:      "maintenance window",
		RequestedAt: requestedAt,
	})
	if err != nil {
		t.Fatalf("Pause: %v", err)
	}
	if !ok {
		t.Fatal("Pause ok = false, want true")
	}
	if paused.Status != DurableJobPaused {
		t.Fatalf("paused status = %q, want %q", paused.Status, DurableJobPaused)
	}
	if paused.PauseActor != "operator:ada" || paused.PauseReason != "maintenance window" {
		t.Fatalf("pause evidence actor/reason = %q/%q", paused.PauseActor, paused.PauseReason)
	}
	if !paused.PausedAt.Equal(requestedAt) {
		t.Fatalf("PausedAt = %v, want %v", paused.PausedAt, requestedAt)
	}
	assertJSONEqual(t, "progress", paused.Progress, `{"phase":"queued"}`)

	status, err := ledger.Status(ctx)
	if err != nil {
		t.Fatalf("Status: %v", err)
	}
	if status.Paused != 1 {
		t.Fatalf("Paused = %d, want 1", status.Paused)
	}
}

func TestDurableLedgerResumeRestoresPausedActiveJobToClaimableStateWithoutLosingAudit(t *testing.T) {
	ledger, _, cleanup := newTestDurableLedger(t)
	defer cleanup()

	ctx := context.Background()
	if _, err := ledger.Submit(ctx, DurableJobSubmission{
		ID:       "shell:pause-active",
		Kind:     WorkKindShellCommand,
		ParentID: "parent-42",
		Progress: json.RawMessage(`{"phase":"running","pct":25}`),
	}); err != nil {
		t.Fatalf("Submit: %v", err)
	}
	timeoutAt := time.Now().UTC().Add(20 * time.Minute).Truncate(0)
	lockUntil := time.Now().UTC().Add(5 * time.Minute).Truncate(0)
	claimed, ok, err := ledger.ClaimJob(ctx, "shell:pause-active", DurableClaim{
		WorkerID:  "worker-a",
		LockUntil: lockUntil,
		TimeoutAt: timeoutAt,
	})
	if err != nil {
		t.Fatalf("ClaimJob: %v", err)
	}
	if !ok {
		t.Fatal("ClaimJob ok = false, want true")
	}

	paused, ok, err := ledger.Pause(ctx, claimed.ID, DurableLifecycleIntent{
		Trust:       TrustSystem,
		Actor:       "supervisor-a",
		Reason:      "drain worker",
		RequestedAt: time.Now().UTC().Add(-time.Second).Truncate(0),
	})
	if err != nil {
		t.Fatalf("Pause active: %v", err)
	}
	if !ok || paused.Status != DurableJobPaused {
		t.Fatalf("Pause active = %+v ok=%v, want paused", paused, ok)
	}

	resumedAt := time.Now().UTC().Truncate(0)
	resumed, ok, err := ledger.Resume(ctx, paused.ID, DurableLifecycleIntent{
		Trust:       TrustOperator,
		Actor:       "operator:ada",
		Reason:      "maintenance done",
		RequestedAt: resumedAt,
	})
	if err != nil {
		t.Fatalf("Resume: %v", err)
	}
	if !ok {
		t.Fatal("Resume ok = false, want true")
	}
	if resumed.Status != DurableJobWaiting {
		t.Fatalf("resumed status = %q, want %q", resumed.Status, DurableJobWaiting)
	}
	if resumed.ResumeActor != "operator:ada" || resumed.ResumeReason != "maintenance done" {
		t.Fatalf("resume evidence actor/reason = %q/%q", resumed.ResumeActor, resumed.ResumeReason)
	}
	if !resumed.ResumeRequestedAt.Equal(resumedAt) {
		t.Fatalf("ResumeRequestedAt = %v, want %v", resumed.ResumeRequestedAt, resumedAt)
	}
	assertJSONEqual(t, "progress", resumed.Progress, `{"phase":"running","pct":25}`)
	if resumed.ParentID != "parent-42" {
		t.Fatalf("ParentID = %q, want parent-42", resumed.ParentID)
	}
	if !resumed.TimeoutAt.Equal(timeoutAt) {
		t.Fatalf("TimeoutAt = %v, want %v", resumed.TimeoutAt, timeoutAt)
	}
	if resumed.LockOwner != "worker-a" || !resumed.LockUntil.Equal(lockUntil) {
		t.Fatalf("lock audit = owner %q until %v, want worker-a until %v", resumed.LockOwner, resumed.LockUntil, lockUntil)
	}

	status, err := ledger.Status(ctx)
	if err != nil {
		t.Fatalf("Status: %v", err)
	}
	if status.Paused != 0 || status.ResumePending != 1 {
		t.Fatalf("status paused/resume-pending = %d/%d, want 0/1", status.Paused, status.ResumePending)
	}

	reclaimed, ok, err := ledger.ClaimJob(ctx, resumed.ID, DurableClaim{
		WorkerID:  "worker-b",
		LockUntil: time.Now().UTC().Add(time.Minute),
	})
	if err != nil {
		t.Fatalf("ClaimJob after resume: %v", err)
	}
	if !ok || reclaimed.Status != DurableJobActive || reclaimed.LockOwner != "worker-b" {
		t.Fatalf("ClaimJob after resume = %+v ok=%v, want active worker-b", reclaimed, ok)
	}
	status, err = ledger.Status(ctx)
	if err != nil {
		t.Fatalf("Status after claim: %v", err)
	}
	if status.ResumePending != 0 {
		t.Fatalf("ResumePending after claim = %d, want 0", status.ResumePending)
	}
}

func TestDurableLedgerDeniesChildAgentLifecycleControlForDeterministicJobs(t *testing.T) {
	ledger, _, cleanup := newTestDurableLedger(t)
	defer cleanup()

	ctx := context.Background()
	for _, tt := range []struct {
		id   string
		kind WorkKind
	}{
		{id: "shell:protected", kind: WorkKindShellCommand},
		{id: "cron:protected", kind: WorkKindCronJob},
	} {
		if _, err := ledger.Submit(ctx, DurableJobSubmission{ID: tt.id, Kind: tt.kind}); err != nil {
			t.Fatalf("Submit %s: %v", tt.id, err)
		}
		if _, ok, err := ledger.Pause(ctx, tt.id, DurableLifecycleIntent{
			Trust:  TrustChildAgent,
			Actor:  "child-agent:7",
			Reason: "try pause",
		}); !errors.Is(err, ErrDurableLifecycleDenied) || ok {
			t.Fatalf("Pause child-agent %s ok=%v err=%v, want ErrDurableLifecycleDenied and false", tt.id, ok, err)
		}
		if _, ok, err := ledger.Resume(ctx, tt.id, DurableLifecycleIntent{
			Trust:  TrustChildAgent,
			Actor:  "child-agent:7",
			Reason: "try resume",
		}); !errors.Is(err, ErrDurableLifecycleDenied) || ok {
			t.Fatalf("Resume child-agent %s ok=%v err=%v, want ErrDurableLifecycleDenied and false", tt.id, ok, err)
		}
		got, err := ledger.Get(ctx, tt.id)
		if err != nil {
			t.Fatalf("Get %s: %v", tt.id, err)
		}
		if got.Status != DurableJobWaiting {
			t.Fatalf("%s status = %q, want waiting after denied lifecycle controls", tt.id, got.Status)
		}
	}

	status, err := ledger.Status(ctx)
	if err != nil {
		t.Fatalf("Status: %v", err)
	}
	if status.LifecycleControlUnsupported != 4 {
		t.Fatalf("LifecycleControlUnsupported = %d, want 4", status.LifecycleControlUnsupported)
	}
}
