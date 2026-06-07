package gateway

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"sync"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/kernel"
)

type fakeRememberedSourceStore struct {
	mu      sync.Mutex
	entries []RememberedSourceEntry
	err     error
}

func (f *fakeRememberedSourceStore) RememberSource(_ context.Context, entry RememberedSourceEntry) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.err != nil {
		return f.err
	}
	f.entries = append(f.entries, entry)
	return nil
}

func (f *fakeRememberedSourceStore) snapshot() []RememberedSourceEntry {
	f.mu.Lock()
	defer f.mu.Unlock()
	return cloneSlice(f.entries)
}

func TestManagerRememberSourceHook_PersistsAllowedInboundSource(t *testing.T) {
	store := &fakeRememberedSourceStore{}
	fk := &fakeKernel{}
	m := NewManagerWithSubmitter(ManagerConfig{
		AllowedChats:          map[string]string{"telegram": "-1001"},
		RememberedSourceStore: store,
	}, fk, slog.Default())
	tg := newFakeChannel("telegram")
	if err := m.Register(tg); err != nil {
		t.Fatalf("Register: %v", err)
	}

	err := m.handleInbound(context.Background(), InboundEvent{
		Platform: "telegram", ChatID: "-1001", ChatName: "Coaching Chat", ChatType: "group",
		UserID: "77", UserName: "Juan", ThreadID: "17585", MsgID: "gateway-msg", MessageID: "platform-msg",
		Kind: EventSubmit, Text: "hello",
	})
	if err != nil {
		t.Fatalf("handleInbound: %v", err)
	}
	if got := len(fk.submitsSnapshot()); got != 1 {
		t.Fatalf("kernel submits = %d, want 1", got)
	}
	entries := store.snapshot()
	if len(entries) != 1 {
		t.Fatalf("remembered entries = %d, want 1", len(entries))
	}
	got := entries[0]
	if got.Platform != "telegram" || got.ChatID != "-1001" || got.ThreadID != "17585" || got.MessageID != "platform-msg" || got.ID != "-1001:17585" {
		t.Fatalf("remembered entry = %+v, want normalized source metadata", got)
	}
}

func TestManagerRememberSourceHook_SkipsUnauthorizedOrPendingDiscovery(t *testing.T) {
	store := &fakeRememberedSourceStore{}
	m := NewManagerWithSubmitter(ManagerConfig{
		AllowedChats:          map[string]string{"telegram": "home"},
		AllowDiscovery:        map[string]bool{"telegram": true},
		RememberedSourceStore: store,
	}, &fakeKernel{}, slog.Default())
	if err := m.Register(newFakeChannel("telegram")); err != nil {
		t.Fatalf("Register: %v", err)
	}
	for _, ev := range []InboundEvent{
		{Platform: "telegram", ChatID: "intruder", Kind: EventSubmit, Text: "blocked"},
		{Platform: "telegram", ChatID: "discovery", Kind: EventSubmit, Text: "pair me"},
	} {
		if err := m.handleInbound(context.Background(), ev); err != nil {
			t.Fatalf("handleInbound(%q): %v", ev.ChatID, err)
		}
	}
	if entries := store.snapshot(); len(entries) != 0 {
		t.Fatalf("remembered unauthorized entries = %+v, want none", entries)
	}
}

func TestManagerRememberSourceHook_DegradesWithoutBlockingTurn(t *testing.T) {
	store := &fakeRememberedSourceStore{err: errors.New("/tmp/private/channel_directory_sources.json: disk full")}
	fk := &fakeKernel{}
	var logs strings.Builder
	m := NewManagerWithSubmitter(ManagerConfig{
		AllowedChats:          map[string]string{"telegram": "42"},
		RememberedSourceStore: store,
	}, fk, slog.New(slog.NewTextHandler(&logs, nil)))
	if err := m.Register(newFakeChannel("telegram")); err != nil {
		t.Fatalf("Register: %v", err)
	}
	if err := m.handleInbound(context.Background(), InboundEvent{Platform: "telegram", ChatID: "42", UserID: "u", Kind: EventSubmit, Text: "still submit"}); err != nil {
		t.Fatalf("handleInbound: %v", err)
	}
	got := fk.submitsSnapshot()
	if len(got) != 1 || got[0].Kind != kernel.PlatformEventSubmit || got[0].Text != "still submit" {
		t.Fatalf("kernel submits = %+v, want turn submitted despite store failure", got)
	}
	logText := logs.String()
	if !strings.Contains(logText, "channel_directory_source_unavailable") {
		t.Fatalf("logs = %q, want degraded evidence code", logText)
	}
	if strings.Contains(logText, "/tmp/private") {
		t.Fatalf("logs leaked host path: %q", logText)
	}
}
