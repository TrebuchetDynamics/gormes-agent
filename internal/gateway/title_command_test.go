package gateway

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/persistence/session"
)

// TestTitleCommand_SetPathPersistsManualFlag drives handleTitleCommand with
// "/title Friendly Greeting", reads back via GetMetadata, and asserts
// TitleManuallySet==true.
func TestTitleCommand_SetPathPersistsManualFlag(t *testing.T) {
	ctx := context.Background()
	smap := session.NewMemMap()
	now := time.Date(2026, 4, 29, 12, 0, 0, 0, time.UTC)
	if err := smap.Put(ctx, "telegram:42", "sess-flag-test"); err != nil {
		t.Fatalf("seed session map: %v", err)
	}
	m := NewManagerWithSubmitter(ManagerConfig{
		AllowedChats: map[string]string{"telegram": "42"},
		SessionMap:   smap,
		Now:          func() time.Time { return now },
	}, &fakeKernel{}, slog.Default())
	ch := newFakeChannel("telegram")
	if err := m.Register(ch); err != nil {
		t.Fatal(err)
	}

	ev := InboundEvent{
		Platform: "telegram",
		ChatID:   "42",
		MsgID:    "77",
		Kind:     EventTitle,
		Text:     "/title Friendly Greeting",
	}
	if err := m.handleInbound(ctx, ev); err != nil {
		t.Fatalf("handleInbound: %v", err)
	}

	meta, ok, err := smap.GetMetadata(ctx, "sess-flag-test")
	if err != nil {
		t.Fatalf("GetMetadata: %v", err)
	}
	if !ok {
		t.Fatal("metadata not persisted after /title set")
	}
	if !meta.TitleManuallySet {
		t.Fatal("TitleManuallySet = false after /title set path, want true")
	}
	if meta.Title != "Friendly Greeting" {
		t.Fatalf("Title = %q, want %q", meta.Title, "Friendly Greeting")
	}
}

// TestTitleCommand_ShowPathDoesNotMutateManualFlag drives handleTitleCommand
// with "/title" (no arg) on a session that already has TitleManuallySet=true
// and asserts the flag is not flipped to false.
func TestTitleCommand_ShowPathDoesNotMutateManualFlag(t *testing.T) {
	ctx := context.Background()
	smap := session.NewMemMap()
	now := time.Date(2026, 4, 29, 12, 0, 0, 0, time.UTC)
	if err := smap.Put(ctx, "telegram:42", "sess-show-test"); err != nil {
		t.Fatalf("seed session map: %v", err)
	}
	// Seed metadata with manual flag already true.
	if err := smap.PutMetadata(ctx, session.Metadata{
		SessionID:        "sess-show-test",
		Source:           "telegram",
		ChatID:           "42",
		Title:            "My Manual Title",
		TitleManuallySet: true,
		UpdatedAt:        now.Unix(),
	}); err != nil {
		t.Fatalf("seed metadata: %v", err)
	}
	m := NewManagerWithSubmitter(ManagerConfig{
		AllowedChats: map[string]string{"telegram": "42"},
		SessionMap:   smap,
		Now:          func() time.Time { return now },
	}, &fakeKernel{}, slog.Default())
	ch := newFakeChannel("telegram")
	if err := m.Register(ch); err != nil {
		t.Fatal(err)
	}

	// Show path: no arg.
	ev := InboundEvent{
		Platform: "telegram",
		ChatID:   "42",
		MsgID:    "78",
		Kind:     EventTitle,
		Text:     "/title",
	}
	if err := m.handleInbound(ctx, ev); err != nil {
		t.Fatalf("handleInbound show path: %v", err)
	}

	meta, ok, err := smap.GetMetadata(ctx, "sess-show-test")
	if err != nil {
		t.Fatalf("GetMetadata after show: %v", err)
	}
	if !ok {
		t.Fatal("metadata disappeared after show path")
	}
	if !meta.TitleManuallySet {
		t.Fatal("TitleManuallySet flipped to false by show path, want unchanged true")
	}
}
