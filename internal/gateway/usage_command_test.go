package gateway

import (
	"context"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/kernel"
	"github.com/TrebuchetDynamics/gormes-agent/internal/llm"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/telemetry"
)

func TestGatewayUsageCommand_RendersRunningFrameBeforeCachedFrameAndAccountLimits(t *testing.T) {
	ch := newFakeChannel("telegram")
	account := llm.AccountUsageSnapshot{
		Provider:  "openai-codex",
		Plan:      "Pro",
		FetchedAt: time.Date(2026, 4, 28, 18, 0, 0, 0, time.UTC),
		Windows: []llm.AccountUsageWindow{{
			Label:       "Session",
			UsedPercent: floatPtr(15),
		}},
	}
	m := NewManagerWithSubmitter(ManagerConfig{
		AllowedChats: map[string]string{"telegram": "42"},
		AccountUsage: func(context.Context, InboundEvent) (llm.AccountUsageSnapshot, error) {
			return account, nil
		},
	}, &fakeKernel{}, slog.Default())
	if err := m.Register(ch); err != nil {
		t.Fatalf("Register: %v", err)
	}
	m.rememberUsageFrame(kernel.RenderFrame{Phase: kernel.PhaseIdle, Model: "cached-model", SessionID: "cached-session", Telemetry: telemetry.Snapshot{TokensInTotal: 1, TokensOutTotal: 2}})
	m.pinTurn("telegram", "42", "active-msg")
	m.rememberUsageFrame(kernel.RenderFrame{Phase: kernel.PhaseStreaming, Model: "running-model", SessionID: "running-session", Telemetry: telemetry.Snapshot{TokensInTotal: 11, TokensOutTotal: 22}})

	if err := m.handleInbound(context.Background(), InboundEvent{Platform: "telegram", ChatID: "42", Kind: EventUsage, Text: "/usage"}); err != nil {
		t.Fatalf("handleInbound(/usage): %v", err)
	}

	sent := ch.sentSnapshot()
	if len(sent) != 1 {
		t.Fatalf("sent messages = %d, want 1", len(sent))
	}
	for _, want := range []string{"Usage source: running turn", "Model: running-model", "Session: running-session", "Tokens: 11 in / 22 out", "Provider: openai-codex (Pro)", "Session: 85% remaining (15% used)"} {
		if !strings.Contains(sent[0].Text, want) {
			t.Fatalf("usage output missing %q:\n%s", want, sent[0].Text)
		}
	}
	if strings.Contains(sent[0].Text, "cached-model") {
		t.Fatalf("usage output used cached frame despite active running frame:\n%s", sent[0].Text)
	}
}

func TestGatewayUsageCommand_UsesCachedFrameWhenNoActiveTurn(t *testing.T) {
	ch := newFakeChannel("telegram")
	m := NewManagerWithSubmitter(ManagerConfig{AllowedChats: map[string]string{"telegram": "42"}}, &fakeKernel{}, slog.Default())
	if err := m.Register(ch); err != nil {
		t.Fatalf("Register: %v", err)
	}
	m.rememberUsageFrame(kernel.RenderFrame{Phase: kernel.PhaseIdle, Model: "cached-model", SessionID: "cached-session", Telemetry: telemetry.Snapshot{TokensInTotal: 3, TokensOutTotal: 4}})

	if err := m.handleInbound(context.Background(), InboundEvent{Platform: "telegram", ChatID: "42", Kind: EventUsage, Text: "/usage"}); err != nil {
		t.Fatalf("handleInbound(/usage): %v", err)
	}

	got := ch.sentSnapshot()[0].Text
	for _, want := range []string{"Usage source: cached turn", "Model: cached-model", "Tokens: 3 in / 4 out", "Provider: unavailable"} {
		if !strings.Contains(got, want) {
			t.Fatalf("usage output missing %q:\n%s", want, got)
		}
	}
}

func TestGatewayUsageCommand_RegisteredInSharedRegistry(t *testing.T) {
	cmd, ok := ResolveCommand("/usage")
	if !ok {
		t.Fatal("/usage did not resolve through gateway CommandRegistry")
	}
	if cmd.Kind != EventUsage || cmd.ActiveTurnPolicy != CommandActiveTurnPolicyImmediate {
		t.Fatalf("/usage command = (%v,%q), want EventUsage immediate", cmd.Kind, cmd.ActiveTurnPolicy)
	}
}

func floatPtr(v float64) *float64 { return &v }
