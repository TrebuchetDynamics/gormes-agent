package gateway

import (
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestVoiceModeStore_PutLookupPersistAndSortsEntries(t *testing.T) {
	path := filepath.Join(t.TempDir(), "gateway", "gateway_voice_mode.json")
	store, err := OpenVoiceModeStore(path)
	if err != nil {
		t.Fatalf("OpenVoiceModeStore: %v", err)
	}

	earlier := time.Date(2026, 4, 23, 15, 0, 0, 0, time.UTC)
	later := earlier.Add(2 * time.Minute)

	if err := store.Put(VoiceModeRecord{
		Platform:          " Telegram ",
		ChatID:            " 42 ",
		ChatName:          " Ops Room ",
		Mode:              VoiceModeAll,
		UpdatedAt:         later,
		UpdatedByUserID:   " u2 ",
		UpdatedByUserName: " Grace ",
	}); err != nil {
		t.Fatalf("Put(telegram): %v", err)
	}
	if err := store.Put(VoiceModeRecord{
		Platform:          "discord",
		ChatID:            "ops-room",
		ChatName:          "Ops Bridge",
		Mode:              VoiceModeVoiceOnly,
		UpdatedAt:         earlier,
		UpdatedByUserID:   "u1",
		UpdatedByUserName: "Ada",
	}); err != nil {
		t.Fatalf("Put(discord): %v", err)
	}

	reopened, err := OpenVoiceModeStore(path)
	if err != nil {
		t.Fatalf("OpenVoiceModeStore(reopen): %v", err)
	}

	got, ok := reopened.Lookup("telegram", "42")
	if !ok {
		t.Fatal("Lookup(telegram, 42) = missing, want persisted record")
	}
	if got.Mode != VoiceModeAll || got.ChatName != "Ops Room" || got.UpdatedByUserName != "Grace" {
		t.Fatalf("lookup = %+v, want normalized persisted telegram record", got)
	}

	snapshot := reopened.Snapshot()
	if len(snapshot) != 2 {
		t.Fatalf("snapshot len = %d, want 2", len(snapshot))
	}
	if snapshot[0].Platform != "discord" || snapshot[1].Platform != "telegram" {
		t.Fatalf("snapshot order = %+v, want deterministic platform sort", snapshot)
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%q): %v", path, err)
	}
	var persisted []VoiceModeRecord
	if err := json.Unmarshal(raw, &persisted); err != nil {
		t.Fatalf("json.Unmarshal(%q): %v", raw, err)
	}
	if len(persisted) != 2 {
		t.Fatalf("persisted len = %d, want 2", len(persisted))
	}
}

func TestManager_Inbound_VoiceCommandsPersistAndAcknowledge(t *testing.T) {
	tg := newFakeChannel("telegram")
	fk := &fakeKernel{}
	store, err := OpenVoiceModeStore(filepath.Join(t.TempDir(), "gateway_voice_mode.json"))
	if err != nil {
		t.Fatalf("OpenVoiceModeStore: %v", err)
	}

	m := NewManagerWithSubmitter(ManagerConfig{
		AllowedChats: map[string]string{"telegram": "42"},
		VoiceModes:   store,
	}, fk, slog.Default())
	if err := m.Register(tg); err != nil {
		t.Fatalf("Register: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { _ = m.Run(ctx) }()

	pushVoice := func(text string) {
		tg.pushInbound(InboundEvent{
			Platform: "telegram",
			ChatID:   "42",
			ChatName: "Ops Room",
			UserID:   "u7",
			UserName: "Ada",
			Kind:     EventVoice,
			Text:     text,
		})
	}

	pushVoice("on")
	waitFor(t, 200*time.Millisecond, func() bool {
		got, ok := store.Lookup("telegram", "42")
		return ok && got.Mode == VoiceModeVoiceOnly && len(tg.sentSnapshot()) == 1
	})
	if got := tg.sentSnapshot()[0].Text; !strings.Contains(got, "voice_only") {
		t.Fatalf("voice on ack = %q, want voice_only confirmation", got)
	}

	pushVoice("status")
	waitFor(t, 200*time.Millisecond, func() bool {
		return len(tg.sentSnapshot()) == 2
	})
	if got := tg.sentSnapshot()[1].Text; !strings.Contains(got, "Current voice mode: voice_only") {
		t.Fatalf("voice status ack = %q, want current-mode detail", got)
	}

	pushVoice("tts")
	waitFor(t, 200*time.Millisecond, func() bool {
		got, ok := store.Lookup("telegram", "42")
		return ok && got.Mode == VoiceModeAll && len(tg.sentSnapshot()) == 3
	})
	if got := tg.sentSnapshot()[2].Text; !strings.Contains(got, "all") {
		t.Fatalf("voice tts ack = %q, want all-mode confirmation", got)
	}

	pushVoice("off")
	waitFor(t, 200*time.Millisecond, func() bool {
		got, ok := store.Lookup("telegram", "42")
		return ok && got.Mode == VoiceModeOff && len(tg.sentSnapshot()) == 4
	})
	if got := tg.sentSnapshot()[3].Text; !strings.Contains(got, "off") {
		t.Fatalf("voice off ack = %q, want off-mode confirmation", got)
	}

	if n := len(fk.submitsSnapshot()); n != 0 {
		t.Fatalf("voice control should not submit to kernel, got %d submits", n)
	}
}

func TestManager_Inbound_VoiceToggleAndUnknownUsage(t *testing.T) {
	tg := newFakeChannel("telegram")
	store, err := OpenVoiceModeStore(filepath.Join(t.TempDir(), "gateway_voice_mode.json"))
	if err != nil {
		t.Fatalf("OpenVoiceModeStore: %v", err)
	}

	m := NewManagerWithSubmitter(ManagerConfig{
		AllowedChats: map[string]string{"telegram": "42"},
		VoiceModes:   store,
	}, &fakeKernel{}, slog.Default())
	if err := m.Register(tg); err != nil {
		t.Fatalf("Register: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { _ = m.Run(ctx) }()

	tg.pushInbound(InboundEvent{
		Platform: "telegram",
		ChatID:   "42",
		ChatName: "Ops Room",
		Kind:     EventVoice,
		Text:     "toggle",
	})

	waitFor(t, 200*time.Millisecond, func() bool {
		got, ok := store.Lookup("telegram", "42")
		return ok && got.Mode == VoiceModeVoiceOnly && len(tg.sentSnapshot()) == 1
	})

	tg.pushInbound(InboundEvent{
		Platform: "telegram",
		ChatID:   "42",
		ChatName: "Ops Room",
		Kind:     EventVoice,
		Text:     "wat",
	})

	waitFor(t, 200*time.Millisecond, func() bool {
		return len(tg.sentSnapshot()) == 2
	})
	if got := tg.sentSnapshot()[1].Text; !strings.Contains(got, "/voice on") || !strings.Contains(got, "/voice tts") {
		t.Fatalf("voice usage = %q, want command help", got)
	}
}
