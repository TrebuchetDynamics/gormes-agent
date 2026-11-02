package voicemode

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

func TestParse(t *testing.T) {
	tests := []struct {
		raw  string
		want Mode
		ok   bool
	}{
		{"off", ModeOff, true},
		{"OFF", ModeOff, true},
		{"disable", ModeOff, true},
		{"disabled", ModeOff, true},
		{"on", ModeVoiceOnly, true},
		{"enable", ModeVoiceOnly, true},
		{"voice_only", ModeVoiceOnly, true},
		{"voice-only", ModeVoiceOnly, true},
		{"tts", ModeAll, true},
		{"all", ModeAll, true},
		{"  tts  ", ModeAll, true},
		{"", "", false},
		{"maybe", "", false},
	}
	for _, tt := range tests {
		got, ok := Parse(tt.raw)
		if ok != tt.ok || got != tt.want {
			t.Errorf("Parse(%q) = (%q, %v), want (%q, %v)", tt.raw, got, ok, tt.want, tt.ok)
		}
	}
}

func TestModeLabel(t *testing.T) {
	if got := ModeOff.Label(); got != "off" {
		t.Fatalf("ModeOff.Label() = %q", got)
	}
	if got := ModeVoiceOnly.Label(); got != "voice only" {
		t.Fatalf("ModeVoiceOnly.Label() = %q", got)
	}
	if got := ModeAll.Label(); got != "all replies" {
		t.Fatalf("ModeAll.Label() = %q", got)
	}
}

func TestMemoryStoreRoundTripAndUnset(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryStore()

	if mode, ok, err := store.Get(ctx, "telegram:42"); err != nil || ok || mode != "" {
		t.Fatalf("unset Get = (%q, %v, %v), want unset with no error", mode, ok, err)
	}
	if err := store.Set(ctx, "telegram:42", ModeVoiceOnly); err != nil {
		t.Fatalf("Set: %v", err)
	}
	if mode, ok, err := store.Get(ctx, "telegram:42"); err != nil || !ok || mode != ModeVoiceOnly {
		t.Fatalf("Get = (%q, %v, %v), want voice_only", mode, ok, err)
	}
	list, err := store.List(ctx)
	if err != nil || len(list) != 1 || list["telegram:42"] != ModeVoiceOnly {
		t.Fatalf("List = (%#v, %v), want one voice_only entry", list, err)
	}
}

func TestStoreRejectsEmptyKeyAndInvalidMode(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryStore()
	if err := store.Set(ctx, "", ModeOff); err == nil {
		t.Fatal("Set with empty key succeeded, want error")
	}
	if err := store.Set(ctx, "telegram:42", Mode("invalid")); err == nil {
		t.Fatal("Set with invalid mode succeeded, want error")
	}
}

func TestFileStorePersistsAcrossInstances(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "gateway_voice_mode.json")

	first := NewFileStore(path)
	if err := first.Set(ctx, "telegram:42", ModeAll); err != nil {
		t.Fatalf("Set: %v", err)
	}

	second := NewFileStore(path)
	mode, ok, err := second.Get(ctx, "telegram:42")
	if err != nil || !ok || mode != ModeAll {
		t.Fatalf("reloaded Get = (%q, %v, %v), want all", mode, ok, err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("mode = %o, want 0600", info.Mode().Perm())
	}
}

func TestFileStoreMissingFileIsUnset(t *testing.T) {
	ctx := context.Background()
	store := NewFileStore(filepath.Join(t.TempDir(), "missing.json"))
	mode, ok, err := store.Get(ctx, "telegram:42")
	if err != nil || ok || mode != "" {
		t.Fatalf("Get on missing file = (%q, %v, %v), want unset with no error", mode, ok, err)
	}
}

func TestFileStoreConcurrentAccess(t *testing.T) {
	ctx := context.Background()
	store := NewFileStore(filepath.Join(t.TempDir(), "gateway_voice_mode.json"))

	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_ = store.Set(ctx, "telegram:42", ModeVoiceOnly)
			_, _, _ = store.Get(ctx, "telegram:42")
		}(i)
	}
	wg.Wait()

	mode, ok, err := store.Get(ctx, "telegram:42")
	if err != nil || !ok || mode != ModeVoiceOnly {
		t.Fatalf("Get after concurrent access = (%q, %v, %v)", mode, ok, err)
	}
}
