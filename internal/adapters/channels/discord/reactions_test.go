package discord

import (
	"context"
	"os"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway"
)

func unsetDiscordEnvForTest(t *testing.T, key string) {
	t.Helper()
	old, ok := os.LookupEnv(key)
	if err := os.Unsetenv(key); err != nil {
		t.Fatalf("Unsetenv(%s): %v", key, err)
	}
	t.Cleanup(func() {
		if ok {
			_ = os.Setenv(key, old)
		} else {
			_ = os.Unsetenv(key)
		}
	})
}

func (m *mockSession) reactionsRemovedSnapshot() []mockReaction {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]mockReaction, len(m.reactionsRemove))
	copy(out, m.reactionsRemove)
	return out
}

func TestDiscordReactionLifecycleEnabledByDefault(t *testing.T) {
	unsetDiscordEnvForTest(t, "DISCORD_REACTIONS")
	ms := newMockSession()
	bot := New(Config{AllowedChannelID: "42"}, ms, nil)

	if err := bot.OnProcessingStart(context.Background(), "42", "m1"); err != nil {
		t.Fatalf("OnProcessingStart: %v", err)
	}
	if err := bot.OnProcessingComplete(context.Background(), "42", "m1", gateway.ProcessingOutcomeSuccess); err != nil {
		t.Fatalf("OnProcessingComplete: %v", err)
	}

	added := ms.reactionsAddedSnapshot()
	removed := ms.reactionsRemovedSnapshot()
	if len(added) != 2 || added[0].Emoji != "👀" || added[1].Emoji != "✅" {
		t.Fatalf("added reactions = %+v, want eyes then success", added)
	}
	if len(removed) != 1 || removed[0].Emoji != "👀" {
		t.Fatalf("removed reactions = %+v, want eyes removed", removed)
	}
}

func TestDiscordReactionLifecycleDisabledAndCancelled(t *testing.T) {
	t.Setenv("DISCORD_REACTIONS", "0")
	ms := newMockSession()
	bot := New(Config{AllowedChannelID: "42"}, ms, nil)

	if err := bot.OnProcessingStart(context.Background(), "42", "m2"); err != nil {
		t.Fatalf("OnProcessingStart disabled: %v", err)
	}
	if err := bot.OnProcessingComplete(context.Background(), "42", "m2", gateway.ProcessingOutcomeFailure); err != nil {
		t.Fatalf("OnProcessingComplete disabled: %v", err)
	}
	if added := ms.reactionsAddedSnapshot(); len(added) != 0 {
		t.Fatalf("added reactions while disabled = %+v", added)
	}

	t.Setenv("DISCORD_REACTIONS", "true")
	if err := bot.OnProcessingStart(context.Background(), "42", "m3"); err != nil {
		t.Fatalf("OnProcessingStart enabled: %v", err)
	}
	if err := bot.OnProcessingComplete(context.Background(), "42", "m3", gateway.ProcessingOutcomeCancelled); err != nil {
		t.Fatalf("OnProcessingComplete cancelled: %v", err)
	}
	added := ms.reactionsAddedSnapshot()
	removed := ms.reactionsRemovedSnapshot()
	if len(added) != 1 || added[0].Emoji != "👀" {
		t.Fatalf("added reactions = %+v, want only eyes for cancelled turn", added)
	}
	if len(removed) != 1 || removed[0].Emoji != "👀" {
		t.Fatalf("removed reactions = %+v, want eyes removed for cancelled turn", removed)
	}
}

func TestDiscordReactionLifecycleFailuresAreSwallowed(t *testing.T) {
	unsetDiscordEnvForTest(t, "DISCORD_REACTIONS")
	ms := newMockSession()
	ms.reactionErr = errUnderlying
	bot := New(Config{AllowedChannelID: "42"}, ms, nil)

	if err := bot.OnProcessingStart(context.Background(), "42", "m4"); err != nil {
		t.Fatalf("OnProcessingStart should swallow reaction error, got %v", err)
	}
	if err := bot.OnProcessingComplete(context.Background(), "42", "m4", gateway.ProcessingOutcomeFailure); err != nil {
		t.Fatalf("OnProcessingComplete should swallow reaction error, got %v", err)
	}
}
