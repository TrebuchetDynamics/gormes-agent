package subagent

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"
)

func TestDurableLedgerReplayCreatesLinkedWaitingJobWithoutMutatingTerminalSource(t *testing.T) {
	ledger, _, cleanup := newTestDurableLedger(t)
	defer cleanup()

	ctx := context.Background()
	for _, tt := range []struct {
		name       string
		sourceID   string
		replayID   string
		terminalFn func(id string)
	}{
		{
			name:     "completed",
			sourceID: "cron:daily:completed",
			replayID: "cron:daily:completed:replay-1",
			terminalFn: func(id string) {
				claimed, ok, err := ledger.ClaimJob(ctx, id, DurableClaim{
					WorkerID:  "worker-a",
					LockUntil: time.Now().UTC().Add(time.Minute),
				})
				if err != nil || !ok {
					t.Fatalf("ClaimJob %s ok=%v err=%v, want true nil", id, ok, err)
				}
				if _, ok, err := ledger.Complete(ctx, claimed.ID, "worker-a", json.RawMessage(`{"delivered":true}`)); err != nil || !ok {
					t.Fatalf("Complete %s ok=%v err=%v, want true nil", id, ok, err)
				}
			},
		},
		{
			name:     "failed",
			sourceID: "cron:daily:failed",
			replayID: "cron:daily:failed:replay-1",
			terminalFn: func(id string) {
				claimed, ok, err := ledger.ClaimJob(ctx, id, DurableClaim{
					WorkerID:  "worker-a",
					LockUntil: time.Now().UTC().Add(time.Minute),
				})
				if err != nil || !ok {
					t.Fatalf("ClaimJob %s ok=%v err=%v, want true nil", id, ok, err)
				}
				if _, ok, err := ledger.Fail(ctx, claimed.ID, "worker-a", "boom"); err != nil || !ok {
					t.Fatalf("Fail %s ok=%v err=%v, want true nil", id, ok, err)
				}
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := ledger.Submit(ctx, DurableJobSubmission{
				ID:       tt.sourceID,
				Kind:     WorkKindCronJob,
				Progress: json.RawMessage(`{"phase":"queued","attempt":1}`),
			}); err != nil {
				t.Fatalf("Submit source: %v", err)
			}
			tt.terminalFn(tt.sourceID)
			before, err := ledger.Get(ctx, tt.sourceID)
			if err != nil {
				t.Fatalf("Get before replay: %v", err)
			}

			replayed, ok, err := ledger.Replay(ctx, tt.sourceID, DurableReplayRequest{
				ID:            tt.replayID,
				DataOverrides: json.RawMessage(`{"window":"today","priority":"high"}`),
			})
			if err != nil {
				t.Fatalf("Replay: %v", err)
			}
			if !ok {
				t.Fatal("Replay ok = false, want true")
			}
			if replayed.ID != tt.replayID || replayed.ReplayOf != tt.sourceID {
				t.Fatalf("replay lineage = id %q replay_of %q, want %q/%q", replayed.ID, replayed.ReplayOf, tt.replayID, tt.sourceID)
			}
			if replayed.Kind != WorkKindCronJob || replayed.Status != DurableJobWaiting {
				t.Fatalf("replay kind/status = %q/%q, want cron_job/waiting", replayed.Kind, replayed.Status)
			}
			assertJSONEqual(t, "replay data overrides", replayed.DataOverrides, `{"window":"today","priority":"high"}`)
			assertJSONEqual(t, "replay progress", replayed.Progress, `{"phase":"queued","attempt":1}`)

			after, err := ledger.Get(ctx, tt.sourceID)
			if err != nil {
				t.Fatalf("Get after replay: %v", err)
			}
			if after.Status != before.Status || after.UpdatedAt != before.UpdatedAt || after.ReplayOf != "" {
				t.Fatalf("source mutated by replay: before=%+v after=%+v", before, after)
			}
			assertJSONEqual(t, "source result", after.Result, string(before.Result))
		})
	}
}

func TestDurableLedgerInboxMessagesAuditSenderAndClaimOnlyByOwnerOrTrustedActor(t *testing.T) {
	ledger, _, cleanup := newTestDurableLedger(t)
	defer cleanup()

	ctx := context.Background()
	if _, err := ledger.Submit(ctx, DurableJobSubmission{ID: "agent:target", Kind: WorkKindLLMSubagent}); err != nil {
		t.Fatalf("Submit target: %v", err)
	}
	sentAt := time.Now().UTC().Add(-time.Minute).Truncate(0)
	msg, err := ledger.SendInboxMessage(ctx, DurableInboxMessageSubmission{
		JobID:       "agent:target",
		Sender:      "operator:ada",
		SenderTrust: TrustOperator,
		Payload:     json.RawMessage(`{"directive":"focus on revenue","skip":"headcount"}`),
		SentAt:      sentAt,
	})
	if err != nil {
		t.Fatalf("SendInboxMessage: %v", err)
	}
	if msg.JobID != "agent:target" || msg.Sender != "operator:ada" || msg.SenderTrust != TrustOperator {
		t.Fatalf("message audit = %+v, want target/operator sender", msg)
	}
	if !msg.UnreadAt.Equal(sentAt) || !msg.ReadAt.IsZero() {
		t.Fatalf("message timestamps unread=%v read=%v, want unread sentAt and no read", msg.UnreadAt, msg.ReadAt)
	}
	assertJSONEqual(t, "inbox payload", msg.Payload, `{"directive":"focus on revenue","skip":"headcount"}`)
	status, err := ledger.Status(ctx)
	if err != nil {
		t.Fatalf("Status after send: %v", err)
	}
	if status.InboxUnread != 1 {
		t.Fatalf("InboxUnread = %d, want 1", status.InboxUnread)
	}

	if _, err := ledger.ClaimInboxMessages(ctx, "agent:target", DurableInboxClaim{
		Trust: TrustChildAgent,
		JobID: "agent:other",
		Actor: "agent:other",
	}); !errors.Is(err, ErrDurableInboxClaimDenied) {
		t.Fatalf("ClaimInboxMessages by other child err=%v, want ErrDurableInboxClaimDenied", err)
	}
	status, err = ledger.Status(ctx)
	if err != nil {
		t.Fatalf("Status after denied claim: %v", err)
	}
	if status.InboxUnread != 1 {
		t.Fatalf("InboxUnread after denied claim = %d, want 1", status.InboxUnread)
	}

	claimedAt := time.Now().UTC().Truncate(0)
	claimed, err := ledger.ClaimInboxMessages(ctx, "agent:target", DurableInboxClaim{
		Trust:     TrustChildAgent,
		JobID:     "agent:target",
		Actor:     "agent:target",
		ClaimedAt: claimedAt,
	})
	if err != nil {
		t.Fatalf("ClaimInboxMessages owner: %v", err)
	}
	if len(claimed) != 1 {
		t.Fatalf("claimed len = %d, want 1", len(claimed))
	}
	if claimed[0].ID != msg.ID || claimed[0].ReadBy != "agent:target" {
		t.Fatalf("claimed audit = %+v, want read by owner", claimed[0])
	}
	if !claimed[0].UnreadAt.Equal(sentAt) || !claimed[0].ReadAt.Equal(claimedAt) {
		t.Fatalf("claimed timestamps unread=%v read=%v, want %v/%v", claimed[0].UnreadAt, claimed[0].ReadAt, sentAt, claimedAt)
	}

	if _, err := ledger.SendInboxMessage(ctx, DurableInboxMessageSubmission{
		JobID:       "agent:target",
		Sender:      "system:supervisor",
		SenderTrust: TrustSystem,
		Payload:     json.RawMessage(`{"directive":"resume normal scan"}`),
	}); err != nil {
		t.Fatalf("SendInboxMessage second: %v", err)
	}
	claimed, err = ledger.ClaimInboxMessages(ctx, "agent:target", DurableInboxClaim{
		Trust: TrustOperator,
		Actor: "operator:ada",
	})
	if err != nil {
		t.Fatalf("ClaimInboxMessages operator: %v", err)
	}
	if len(claimed) != 1 || claimed[0].ReadBy != "operator:ada" {
		t.Fatalf("operator claimed = %+v, want one message read by operator:ada", claimed)
	}
	status, err = ledger.Status(ctx)
	if err != nil {
		t.Fatalf("Status after claims: %v", err)
	}
	if status.InboxUnread != 0 {
		t.Fatalf("InboxUnread after claims = %d, want 0", status.InboxUnread)
	}
	audit, err := ledger.InboxMessages(ctx, "agent:target")
	if err != nil {
		t.Fatalf("InboxMessages audit: %v", err)
	}
	if len(audit) != 2 {
		t.Fatalf("audit len = %d, want 2", len(audit))
	}
	for _, msg := range audit {
		if msg.ReadAt.IsZero() || msg.ReadBy == "" {
			t.Fatalf("audit message missing read receipt: %+v", msg)
		}
	}
}

func TestDurableLedgerProtectedSubmitDenialDoesNotBlockChildObservation(t *testing.T) {
	ledger, _, cleanup := newTestDurableLedger(t)
	defer cleanup()

	ctx := context.Background()
	if _, err := ledger.SubmitWithTrust(ctx, TrustOperator, DurableJobSubmission{
		ID:       "shell:observe",
		Kind:     WorkKindShellCommand,
		Progress: json.RawMessage(`{"step":"queued"}`),
	}); err != nil {
		t.Fatalf("operator SubmitWithTrust: %v", err)
	}

	policy := DefaultMinionRoutingPolicy()
	if !policy.CanObserve(TrustChildAgent, WorkKindShellCommand) {
		t.Fatal("child-agent CanObserve shell_command = false, want true")
	}
	got, err := ledger.Get(ctx, "shell:observe")
	if err != nil {
		t.Fatalf("Get observable shell job: %v", err)
	}
	if got.Kind != WorkKindShellCommand || got.Status != DurableJobWaiting {
		t.Fatalf("observable job kind/status = %q/%q, want shell_command/waiting", got.Kind, got.Status)
	}
	jobs, err := ledger.List(ctx, DurableJobListFilter{Kind: WorkKindShellCommand})
	if err != nil {
		t.Fatalf("List shell jobs: %v", err)
	}
	if len(jobs) != 1 || jobs[0].ID != "shell:observe" {
		t.Fatalf("List shell jobs = %+v, want only shell:observe", jobs)
	}
	progress, err := ledger.Progress(ctx, "shell:observe")
	if err != nil {
		t.Fatalf("Progress shell job: %v", err)
	}
	assertJSONEqual(t, "shell progress", progress, `{"step":"queued"}`)

	if _, err := ledger.SubmitWithTrust(ctx, TrustChildAgent, DurableJobSubmission{
		ID:   "shell:child-submit",
		Kind: WorkKindShellCommand,
	}); !errors.Is(err, ErrDurableRouteDenied) {
		t.Fatalf("child-agent SubmitWithTrust err=%v, want ErrDurableRouteDenied", err)
	}
	status, err := ledger.Status(ctx)
	if err != nil {
		t.Fatalf("Status after denied submit: %v", err)
	}
	if status.ProtectedSubmitDenied != 1 {
		t.Fatalf("ProtectedSubmitDenied = %d, want 1", status.ProtectedSubmitDenied)
	}
	jobs, err = ledger.List(ctx, DurableJobListFilter{Kind: WorkKindShellCommand})
	if err != nil {
		t.Fatalf("List after denied submit: %v", err)
	}
	if len(jobs) != 1 || jobs[0].ID != "shell:observe" {
		t.Fatalf("List after denied submit = %+v, want only original job", jobs)
	}
}
