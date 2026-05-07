package slack

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway"
)

func TestSlackChannelScopeAddsSkillsAndPrompt(t *testing.T) {
	mc := newMockClient()
	ch := NewChannel(mc, nil, ChannelConfig{
		RequireMention: false,
		ChannelSkillBindings: []gateway.ChannelSkillBinding{
			{ID: "C123", Skills: []string{"ops", "review", "ops"}},
		},
		ChannelPrompts: map[string]string{
			"C123": "Slack channel prompt",
		},
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	inbox := make(chan gateway.InboundEvent, 1)
	done := make(chan error, 1)
	go func() {
		done <- ch.Run(ctx, inbox)
	}()

	mc.pushEvent(Event{
		RequestID: "req-channel-scope",
		ChannelID: "C123",
		UserID:    "U1",
		Text:      "hello scoped channel",
		Timestamp: "1711111111.000200",
		ThreadTS:  "1711111111.000100",
	})

	select {
	case ev := <-inbox:
		if !reflect.DeepEqual(ev.AutoSkills, []string{"ops", "review"}) {
			t.Fatalf("AutoSkills = %#v, want ops/review", ev.AutoSkills)
		}
		if ev.ChannelPrompt != "Slack channel prompt" {
			t.Fatalf("ChannelPrompt = %q, want Slack channel prompt", ev.ChannelPrompt)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for Slack channel-scope event")
	}

	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Run after cancel = %v, want nil", err)
		}
	case <-time.After(time.Second):
		t.Fatal("Run did not return after cancellation")
	}
}
