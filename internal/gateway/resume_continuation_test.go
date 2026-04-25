package gateway

import (
	"context"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/session"
)

func TestManager_SubmitFollowsCompressionContinuation(t *testing.T) {
	tg := newFakeChannel("telegram")
	fk := &fakeKernel{}
	smap := session.NewMemMap()
	ctx := context.Background()

	if err := smap.Put(ctx, "telegram:42", "sess-root"); err != nil {
		t.Fatalf("Put session root: %v", err)
	}
	for _, meta := range []session.Metadata{
		{SessionID: "sess-root", UpdatedAt: 10},
		{SessionID: "sess-fork", ParentSessionID: "sess-root", LineageKind: session.LineageKindFork, UpdatedAt: 40},
		{SessionID: "sess-child", ParentSessionID: "sess-root", LineageKind: session.LineageKindCompression, UpdatedAt: 20},
		{SessionID: "sess-live", ParentSessionID: "sess-child", LineageKind: session.LineageKindCompression, UpdatedAt: 30},
	} {
		if err := smap.PutMetadata(ctx, meta); err != nil {
			t.Fatalf("PutMetadata(%s): %v", meta.SessionID, err)
		}
	}

	m := NewManagerWithSubmitter(ManagerConfig{
		AllowedChats: map[string]string{"telegram": "42"},
		SessionMap:   smap,
	}, fk, slog.Default())
	if err := m.Register(tg); err != nil {
		t.Fatalf("Register: %v", err)
	}

	runCtx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { _ = m.Run(runCtx) }()

	tg.pushInbound(InboundEvent{
		Platform: "telegram",
		ChatID:   "42",
		MsgID:    "m1",
		Kind:     EventSubmit,
		Text:     "hello",
	})

	waitFor(t, 200*time.Millisecond, func() bool {
		return len(fk.submitsSnapshot()) == 1
	})

	got := fk.submitsSnapshot()[0]
	if got.SessionID != "sess-live" {
		t.Fatalf("submitted SessionID = %q, want live compression descendant sess-live", got.SessionID)
	}
	for _, want := range []string{
		"**Session ID:** `sess-live`",
		"**Requested Session ID:** `sess-root`",
		"**Resume Continuation:** `sess-root` -> `sess-child` -> `sess-live`",
	} {
		if !strings.Contains(got.SessionContext, want) {
			t.Fatalf("SessionContext missing %q in:\n%s", want, got.SessionContext)
		}
	}
	if strings.Contains(got.SessionContext, "sess-fork") {
		t.Fatalf("SessionContext included fork child in compression continuation:\n%s", got.SessionContext)
	}
}

func TestManager_SubmitReportsUnresolvedContinuationFallback(t *testing.T) {
	tg := newFakeChannel("telegram")
	fk := &fakeKernel{}
	base := session.NewMemMap()
	ctx := context.Background()
	if err := base.Put(ctx, "telegram:42", "sess-root"); err != nil {
		t.Fatalf("Put session root: %v", err)
	}
	smap := unresolvedLineageMap{
		Map: base,
		resolution: session.LineageResolution{
			RequestedSessionID: "sess-root",
			LiveSessionID:      "sess-root",
			Path:               []string{"sess-root", "sess-child", "sess-root"},
			Status:             session.LineageStatusLoop,
		},
	}

	m := NewManagerWithSubmitter(ManagerConfig{
		AllowedChats: map[string]string{"telegram": "42"},
		SessionMap:   smap,
	}, fk, slog.Default())
	if err := m.Register(tg); err != nil {
		t.Fatalf("Register: %v", err)
	}

	runCtx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { _ = m.Run(runCtx) }()

	tg.pushInbound(InboundEvent{
		Platform: "telegram",
		ChatID:   "42",
		MsgID:    "m1",
		Kind:     EventSubmit,
		Text:     "hello",
	})

	waitFor(t, 200*time.Millisecond, func() bool {
		return len(fk.submitsSnapshot()) == 1
	})

	got := fk.submitsSnapshot()[0]
	if got.SessionID != "sess-root" {
		t.Fatalf("submitted SessionID = %q, want unresolved fallback sess-root", got.SessionID)
	}
	for _, want := range []string{
		"**Session ID:** `sess-root`",
		"**Requested Session ID:** `sess-root`",
		"**Resume Continuation Status:** `loop`",
	} {
		if !strings.Contains(got.SessionContext, want) {
			t.Fatalf("SessionContext missing %q in:\n%s", want, got.SessionContext)
		}
	}
}

type unresolvedLineageMap struct {
	session.Map
	resolution session.LineageResolution
}

func (m unresolvedLineageMap) ResolveLineageTip(context.Context, string) (session.LineageResolution, error) {
	return m.resolution, nil
}
