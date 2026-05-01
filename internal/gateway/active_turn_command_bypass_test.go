package gateway

import (
	"context"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/kernel"
)

func TestManager_ActiveTurnSlashCommandBypass_ChannelNeutral(t *testing.T) {
	cases := []struct {
		name     string
		platform string
		chatID   string
		command  string
		wantText string
		wantKind kernel.PlatformEventKind
	}{
		{
			name:     "telegram help bypasses active turn",
			platform: "telegram",
			chatID:   "42",
			command:  "/help",
			wantText: "Gormes is online. Available commands:",
		},
		{
			name:     "discord stop bypasses active turn",
			platform: "discord",
			chatID:   "C42",
			command:  "/stop",
			wantKind: kernel.PlatformEventCancel,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ch := newFakeChannel(tc.platform)
			fk := &fakeKernel{}
			m := NewManagerWithSubmitter(ManagerConfig{
				AllowedChats: map[string]string{tc.platform: tc.chatID},
			}, fk, slog.Default())
			if err := m.Register(ch); err != nil {
				t.Fatalf("Register: %v", err)
			}

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			go func() { _ = m.Run(ctx) }()

			ch.pushInbound(InboundEvent{Platform: tc.platform, ChatID: tc.chatID, UserID: "u", MsgID: "m1", Kind: EventSubmit, Text: "start long turn"})
			waitFor(t, 200*time.Millisecond, func() bool { return len(fk.submitsSnapshot()) == 1 })

			ch.pushInbound(InboundEvent{Platform: tc.platform, ChatID: tc.chatID, UserID: "u", MsgID: "m2", Kind: EventSubmit, Text: tc.command})

			if tc.wantText != "" {
				waitFor(t, 200*time.Millisecond, func() bool {
					for _, sent := range ch.sentSnapshot() {
						if strings.Contains(sent.Text, tc.wantText) {
							return true
						}
					}
					return false
				})
			}
			if tc.wantKind != 0 {
				waitFor(t, 200*time.Millisecond, func() bool {
					for _, submit := range fk.submitsSnapshot() {
						if submit.Kind == tc.wantKind {
							return true
						}
					}
					return false
				})
			}
			for _, submit := range fk.submitsSnapshot() {
				if submit.Kind == kernel.PlatformEventSubmit && strings.HasPrefix(strings.TrimSpace(submit.Text), "/") {
					t.Fatalf("slash command leaked into model prompt: %+v", submit)
				}
			}
		})
	}
}

func TestManager_ActiveTurnSlashCommandDeniedDoesNotLeak(t *testing.T) {
	cases := []struct {
		name     string
		command  string
		wantText string
	}{
		{name: "busy reject mutator", command: "/new", wantText: "Gormes is busy"},
		{name: "unavailable command", command: "/browser", wantText: "/browser is recognized but unavailable"},
		{name: "unknown command", command: "/does-not-exist", wantText: "unknown command"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ch := newFakeChannel("slack")
			fk := &fakeKernel{}
			m := NewManagerWithSubmitter(ManagerConfig{
				AllowedChats: map[string]string{"slack": "C42"},
			}, fk, slog.Default())
			if err := m.Register(ch); err != nil {
				t.Fatalf("Register: %v", err)
			}

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			go func() { _ = m.Run(ctx) }()

			ch.pushInbound(InboundEvent{Platform: "slack", ChatID: "C42", UserID: "u", MsgID: "m1", Kind: EventSubmit, Text: "start long turn"})
			waitFor(t, 200*time.Millisecond, func() bool { return len(fk.submitsSnapshot()) == 1 })

			ch.pushInbound(InboundEvent{Platform: "slack", ChatID: "C42", UserID: "u", MsgID: "m2", Kind: EventSubmit, Text: tc.command})
			waitFor(t, 200*time.Millisecond, func() bool {
				for _, sent := range ch.sentSnapshot() {
					if strings.Contains(sent.Text, tc.wantText) {
						return true
					}
				}
				return false
			})

			if got := len(fk.submitsSnapshot()); got != 1 {
				t.Fatalf("kernel submits = %d, want only the original active turn", got)
			}
		})
	}
}
