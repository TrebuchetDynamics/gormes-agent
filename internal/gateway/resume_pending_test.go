package gateway

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/hermes"
	"github.com/TrebuchetDynamics/gormes-agent/internal/kernel"
	"github.com/TrebuchetDynamics/gormes-agent/internal/session"
)

func TestManager_DrainTimeoutMarksOnlyStillRunningTurnResumePending(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 4, 25, 16, 20, 0, 0, time.UTC)
	tg := newFakeChannel("telegram")
	dc := newFakeChannel("discord")
	frames := make(chan kernel.RenderFrame, 8)
	fk := &fakeKernel{}
	smap := session.NewMemMap()
	store := NewRuntimeStatusStore(t.TempDir() + "/gateway_state.json")
	store.now = func() time.Time { return now }

	if err := smap.Put(ctx, "telegram:42", "sess-running"); err != nil {
		t.Fatalf("Put running session: %v", err)
	}
	if err := smap.Put(ctx, "discord:99", "sess-queued"); err != nil {
		t.Fatalf("Put queued session: %v", err)
	}

	m := NewManagerWithSubmitter(ManagerConfig{
		AllowedChats: map[string]string{
			"telegram": "42",
			"discord":  "99",
		},
		CoalesceMs:    10,
		SessionMap:    smap,
		RuntimeStatus: store,
		Now:           func() time.Time { return now },
	}, fk, slog.Default())
	m.setRenderChan(frames)
	_ = m.Register(tg)
	_ = m.Register(dc)

	runManagerForResumeTest(t, m)

	tg.pushInbound(InboundEvent{
		Platform: "telegram", ChatID: "42", UserID: "u-42", MsgID: "m1",
		Kind: EventSubmit, Text: "long work",
	})
	waitFor(t, 200*time.Millisecond, func() bool {
		return len(fk.submitsSnapshot()) == 1
	})

	dc.pushInbound(InboundEvent{
		Platform: "discord", ChatID: "99", UserID: "u-99", MsgID: "m2",
		Kind: EventSubmit, Text: "queued behind long work",
	})
	time.Sleep(30 * time.Millisecond)
	if got := fk.submitsSnapshot(); len(got) != 1 {
		t.Fatalf("queued follow-up submitted before drain timeout: %#v", got)
	}

	shutdownCtx, cancelShutdown := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancelShutdown()
	err := m.ShutdownWithDrainReason(shutdownCtx, DrainReasonShutdownTimeout)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("ShutdownWithDrainReason error = %v, want deadline exceeded", err)
	}

	running, ok, err := smap.GetMetadata(ctx, "sess-running")
	if err != nil {
		t.Fatalf("GetMetadata running: %v", err)
	}
	if !ok {
		t.Fatal("running session metadata missing")
	}
	if !running.ResumePending {
		t.Fatalf("running session ResumePending = false, want true: %+v", running)
	}
	if running.ResumeReason != session.ResumeReasonShutdownTimeout {
		t.Fatalf("running ResumeReason = %q, want %q", running.ResumeReason, session.ResumeReasonShutdownTimeout)
	}
	if running.ResumeMarkedAt != now.Unix() {
		t.Fatalf("running ResumeMarkedAt = %d, want %d", running.ResumeMarkedAt, now.Unix())
	}
	if running.Source != "telegram" || running.ChatID != "42" || running.UserID != "u-42" {
		t.Fatalf("running metadata source = %+v, want telegram/42/u-42", running)
	}

	queued, ok, err := smap.GetMetadata(ctx, "sess-queued")
	if err != nil {
		t.Fatalf("GetMetadata queued: %v", err)
	}
	if ok && queued.ResumePending {
		t.Fatalf("queued follow-up was marked resume pending: %+v", queued)
	}

	status, err := store.ReadRuntimeStatus(ctx)
	if err != nil {
		t.Fatalf("ReadRuntimeStatus: %v", err)
	}
	if len(status.DrainTimeouts) != 1 {
		t.Fatalf("DrainTimeouts len = %d, want 1: %+v", len(status.DrainTimeouts), status.DrainTimeouts)
	}
	if status.DrainTimeouts[0].Reason != string(DrainReasonShutdownTimeout) {
		t.Fatalf("DrainTimeout reason = %q, want %q", status.DrainTimeouts[0].Reason, DrainReasonShutdownTimeout)
	}
	if len(status.ResumePending) != 1 || status.ResumePending[0].SessionID != "sess-running" {
		t.Fatalf("ResumePending status = %+v, want sess-running evidence", status.ResumePending)
	}
}

func TestManager_ResumePendingNextSubmitPrependsOneReasonNoteAndClearsAfterAccepted(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 4, 25, 16, 25, 0, 0, time.UTC)
	tg := newFakeChannel("telegram")
	frames := make(chan kernel.RenderFrame, 8)
	fk := &fakeKernel{}
	smap := session.NewMemMap()

	if err := smap.Put(ctx, "telegram:42", "sess-resume"); err != nil {
		t.Fatalf("Put session: %v", err)
	}
	if err := smap.PutMetadata(ctx, session.Metadata{
		SessionID:      "sess-resume",
		Source:         "telegram",
		ChatID:         "42",
		UserID:         "u-42",
		ResumePending:  true,
		ResumeReason:   session.ResumeReasonRestartTimeout,
		ResumeMarkedAt: now.Add(-time.Minute).Unix(),
	}); err != nil {
		t.Fatalf("PutMetadata resume pending: %v", err)
	}

	m := NewManagerWithSubmitter(ManagerConfig{
		AllowedChats: map[string]string{"telegram": "42"},
		CoalesceMs:   10,
		SessionMap:   smap,
		Now:          func() time.Time { return now },
	}, fk, slog.Default())
	m.setRenderChan(frames)
	_ = m.Register(tg)

	runManagerForResumeTest(t, m)

	tg.pushInbound(InboundEvent{
		Platform: "telegram", ChatID: "42", UserID: "u-42", MsgID: "m1",
		Kind: EventSubmit, Text: "what happened?",
	})
	waitFor(t, 200*time.Millisecond, func() bool {
		return len(fk.submitsSnapshot()) == 1
	})

	first := fk.submitsSnapshot()[0]
	if first.SessionID != "sess-resume" {
		t.Fatalf("first SessionID = %q, want resumable sess-resume", first.SessionID)
	}
	if strings.Count(first.Text, "[System note:") != 1 {
		t.Fatalf("first text should contain exactly one system note:\n%s", first.Text)
	}
	if !strings.Contains(first.Text, "gateway restart") || !strings.Contains(first.Text, "what happened?") {
		t.Fatalf("first text missing restart-aware note or user text:\n%s", first.Text)
	}

	meta, ok, err := smap.GetMetadata(ctx, "sess-resume")
	if err != nil {
		t.Fatalf("GetMetadata after accepted submit: %v", err)
	}
	if !ok {
		t.Fatal("metadata missing after accepted submit")
	}
	if meta.ResumePending || meta.ResumeReason != "" || meta.ResumeMarkedAt != 0 {
		t.Fatalf("resume_pending not cleared after accepted submit: %+v", meta)
	}

	tg.pushInbound(InboundEvent{
		Platform: "telegram", ChatID: "42", UserID: "u-42", MsgID: "m2",
		Kind: EventSubmit, Text: "next in line",
	})
	time.Sleep(30 * time.Millisecond)
	if got := fk.submitsSnapshot(); len(got) != 1 {
		t.Fatalf("follow-up submitted before terminal frame: %#v", got)
	}

	frames <- kernel.RenderFrame{
		Phase: kernel.PhaseIdle,
		History: []hermes.Message{
			{Role: "user", Content: "what happened?"},
			{Role: "assistant", Content: "resume accepted"},
		},
	}
	waitFor(t, 200*time.Millisecond, func() bool {
		return len(fk.submitsSnapshot()) == 2
	})

	second := fk.submitsSnapshot()[1]
	if second.Text != "next in line" {
		t.Fatalf("second text = %q, want FIFO follow-up without resume note", second.Text)
	}
}

func TestManager_ResumePendingLosesToHardNonResumableEvidence(t *testing.T) {
	for _, reason := range []string{
		session.NonResumableSuspended,
		session.NonResumableCancelled,
		session.NonResumableStuckLoop,
	} {
		t.Run(reason, func(t *testing.T) {
			ctx := context.Background()
			now := time.Date(2026, 4, 25, 16, 30, 0, 0, time.UTC)
			tg := newFakeChannel("telegram")
			fk := &fakeKernel{}
			smap := session.NewMemMap()
			store := NewRuntimeStatusStore(t.TempDir() + "/gateway_state.json")
			store.now = func() time.Time { return now }

			if err := smap.Put(ctx, "telegram:42", "sess-hard"); err != nil {
				t.Fatalf("Put session: %v", err)
			}
			if err := smap.PutMetadata(ctx, session.Metadata{
				SessionID:          "sess-hard",
				Source:             "telegram",
				ChatID:             "42",
				ResumePending:      true,
				ResumeReason:       session.ResumeReasonRestartTimeout,
				ResumeMarkedAt:     now.Add(-time.Minute).Unix(),
				NonResumableReason: reason,
				NonResumableAt:     now.Unix(),
			}); err != nil {
				t.Fatalf("PutMetadata hard state: %v", err)
			}

			m := NewManagerWithSubmitter(ManagerConfig{
				AllowedChats:  map[string]string{"telegram": "42"},
				SessionMap:    smap,
				RuntimeStatus: store,
				Now:           func() time.Time { return now },
			}, fk, slog.Default())
			_ = m.Register(tg)

			runManagerForResumeTest(t, m)

			tg.pushInbound(InboundEvent{
				Platform: "telegram", ChatID: "42", MsgID: "m1",
				Kind: EventSubmit, Text: "start fresh",
			})
			waitFor(t, 200*time.Millisecond, func() bool {
				return len(fk.submitsSnapshot()) == 1
			})

			got := fk.submitsSnapshot()[0]
			if got.SessionID != "telegram:42" {
				t.Fatalf("SessionID = %q, want fresh chat key after %s", got.SessionID, reason)
			}
			if got.Text != "start fresh" || strings.Contains(got.Text, "[System note:") {
				t.Fatalf("hard non-resumable state injected resume note:\n%s", got.Text)
			}
			for _, want := range []string{
				"**Non-Resumable Session ID:** `sess-hard`",
				"**Non-Resumable Reason:** `" + reason + "`",
			} {
				if !strings.Contains(got.SessionContext, want) {
					t.Fatalf("SessionContext missing %q in:\n%s", want, got.SessionContext)
				}
			}

			status, err := store.ReadRuntimeStatus(ctx)
			if err != nil {
				t.Fatalf("ReadRuntimeStatus: %v", err)
			}
			if len(status.NonResumable) != 1 || status.NonResumable[0].Reason != reason {
				t.Fatalf("NonResumable status = %+v, want reason %s", status.NonResumable, reason)
			}
		})
	}
}

func runManagerForResumeTest(t *testing.T, m *Manager) {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		done <- m.Run(ctx)
	}()
	t.Cleanup(func() {
		cancel()
		select {
		case err := <-done:
			if err != nil {
				t.Errorf("manager Run after cancel: %v", err)
			}
		case <-time.After(500 * time.Millisecond):
			t.Errorf("manager Run did not stop after cancel")
		}
	})
}
