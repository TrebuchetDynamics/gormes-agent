package discord

import (
	"encoding/json"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
)

func TestDiscordThreadParticipationTracker(t *testing.T) {
	statePath := filepath.Join(t.TempDir(), "state", "discord_threads.json")

	tracker := NewThreadParticipationTracker(ThreadParticipationOptions{
		Path:       statePath,
		MaxTracked: 5,
	})
	if tracker.Contains("thread-1") {
		t.Fatal("fresh tracker contains thread-1, want empty")
	}

	if ev, err := tracker.Mark("thread-1"); err != nil || ev.Code != "" {
		t.Fatalf("Mark(thread-1) evidence=%+v err=%v, want success", ev, err)
	}
	if ev, err := tracker.Mark("thread-2"); err != nil || ev.Code != "" {
		t.Fatalf("Mark(thread-2) evidence=%+v err=%v, want success", ev, err)
	}
	if ev, err := tracker.Mark("thread-1"); err != nil || ev.Code != "" {
		t.Fatalf("duplicate Mark evidence=%+v err=%v, want no-op success", ev, err)
	}

	raw, err := os.ReadFile(statePath)
	if err != nil {
		t.Fatalf("ReadFile(%s): %v", statePath, err)
	}
	var saved []string
	if err := json.Unmarshal(raw, &saved); err != nil {
		t.Fatalf("unmarshal state: %v", err)
	}
	if got, want := saved, []string{"thread-1", "thread-2"}; !equalStrings(got, want) {
		t.Fatalf("saved threads = %#v, want %#v", got, want)
	}

	reloaded := NewThreadParticipationTracker(ThreadParticipationOptions{
		Path:       statePath,
		MaxTracked: 5,
	})
	if !reloaded.Contains("thread-1") || !reloaded.Contains("thread-2") {
		t.Fatalf("reloaded tracker missing persisted threads: %+v", reloaded.Snapshot())
	}
}

func TestDiscordThreadParticipationTrackerCapsOldEntries(t *testing.T) {
	statePath := filepath.Join(t.TempDir(), "discord_threads.json")
	tracker := NewThreadParticipationTracker(ThreadParticipationOptions{
		Path:       statePath,
		MaxTracked: 3,
	})

	for _, id := range []string{"thread-1", "thread-2", "thread-3", "thread-4", "thread-5"} {
		if ev, err := tracker.Mark(id); err != nil || ev.Code != "" {
			t.Fatalf("Mark(%s) evidence=%+v err=%v, want success", id, ev, err)
		}
	}

	if got, want := tracker.Snapshot(), []string{"thread-3", "thread-4", "thread-5"}; !equalStrings(got, want) {
		t.Fatalf("snapshot = %#v, want newest capped %#v", got, want)
	}
}

func TestDiscordThreadParticipationTrackerCorruptStateFallsBackToEmpty(t *testing.T) {
	statePath := filepath.Join(t.TempDir(), "discord_threads.json")
	if err := os.WriteFile(statePath, []byte(`{bad json`), 0o600); err != nil {
		t.Fatalf("WriteFile(corrupt): %v", err)
	}

	tracker := NewThreadParticipationTracker(ThreadParticipationOptions{Path: statePath})
	if tracker.LoadEvidence().Code != "discord_thread_tracker_reset" {
		t.Fatalf("LoadEvidence = %+v, want discord_thread_tracker_reset", tracker.LoadEvidence())
	}
	if tracker.Contains("thread-1") {
		t.Fatal("corrupt state tracker contains thread-1, want empty")
	}
}

func TestDiscordThreadParticipationTrackerMissingDirectoryPersistsOnMark(t *testing.T) {
	statePath := filepath.Join(t.TempDir(), "missing", "deep", "discord_threads.json")
	tracker := NewThreadParticipationTracker(ThreadParticipationOptions{Path: statePath})

	if ev, err := tracker.Mark("thread-1"); err != nil || ev.Code != "" {
		t.Fatalf("Mark with missing directory evidence=%+v err=%v, want success", ev, err)
	}
	if _, err := os.Stat(statePath); err != nil {
		t.Fatalf("state file after mark: %v", err)
	}
}

func TestDiscordThreadParticipationSurvivesBotRestart(t *testing.T) {
	statePath := filepath.Join(t.TempDir(), "discord_threads.json")

	ms := newMockSession()
	b := New(Config{AllowedChannelID: "forum-100", ThreadStatePath: statePath}, ms, slog.Default())
	b.rememberThread(loadDiscordChannelFixture(t, "forum_thread_create.json"))

	msg := loadDiscordMessageCreateFixture(t, "forum_thread_message.json")
	if _, ok := b.toInboundEvent(msg.Message); !ok {
		t.Fatal("toInboundEvent(thread message) returned !ok")
	}
	if !b.hasParticipatedThread("thread-200") {
		t.Fatal("bot did not mark thread-200 as participated")
	}

	restarted := New(Config{AllowedChannelID: "forum-100", ThreadStatePath: statePath}, newMockSession(), slog.Default())
	if !restarted.hasParticipatedThread("thread-200") {
		t.Fatal("restarted bot did not load participated thread-200")
	}
	if _, ok := restarted.threadForMessageChannel("thread-200"); ok {
		t.Fatal("restarted bot has in-memory thread metadata, want only persisted participation")
	}
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
