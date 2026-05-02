package gateway

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/session"
)

func TestMultiAgentGatewayRuntime_SubmitRoutesAccountsToAgentSessionKeys(t *testing.T) {
	ctx := context.Background()
	tg := newFakeChannel("telegram")
	fk := &fakeKernel{}
	smap := session.NewMemMap()

	mainWorkspace := writeAgentWorkspace(t, "main workspace guidance")
	alertsWorkspace := writeAgentWorkspace(t, "alerts workspace guidance")
	mainAgentDir := writeAgentMemory(t, "main durable user")
	alertsAgentDir := writeAgentMemory(t, "alerts durable user")

	m := NewManagerWithSubmitter(ManagerConfig{
		AllowedChats: map[string]string{"telegram": "42"},
		SessionMap:   smap,
		AgentRouting: AgentRoutingConfig{
			Enabled: true,
			Agents: config.AgentsCfg{List: []config.AgentCfg{
				{ID: "main", Name: "Main", Workspace: mainWorkspace, AgentDir: mainAgentDir, Default: true, Model: "gpt-main"},
				{ID: "alerts", Name: "Alerts", Workspace: alertsWorkspace, AgentDir: alertsAgentDir, Model: "gpt-alerts"},
			}},
			Bindings: []config.AgentBindingCfg{
				{AgentID: "alerts", Match: config.AgentBindingMatchCfg{Channel: "telegram", AccountID: "alerts"}},
				{AgentID: "main", Match: config.AgentBindingMatchCfg{Channel: "telegram"}},
			},
		},
	}, fk, slog.Default())
	if err := m.Register(tg); err != nil {
		t.Fatalf("Register: %v", err)
	}
	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	go func() { _ = m.Run(runCtx) }()

	tg.pushInbound(InboundEvent{
		Platform: "telegram", AccountID: "alerts", ChatID: "42", ChatType: "private", UserID: "u1", MsgID: "m1",
		Kind: EventSubmit, Text: "alerts please",
	})
	waitFor(t, 200*time.Millisecond, func() bool { return len(fk.submitsSnapshot()) == 1 })
	first := fk.submitsSnapshot()[0]
	if first.Model != "gpt-alerts" {
		t.Fatalf("first submit Model = %q, want per-agent model", first.Model)
	}
	assertContainsAll(t, first.SessionContext,
		"**Agent ID:** `alerts`",
		"**Agent Binding:** `account`",
		"**Session Key:** `agent:alerts:telegram:42`",
		"alerts workspace guidance",
		"alerts durable user",
	)
	if stored, err := smap.Get(ctx, "agent:alerts:telegram:42"); err != nil || stored == "" || stored != first.SessionID {
		t.Fatalf("alerts session map = %q, %v; want submitted session id %q", stored, err, first.SessionID)
	}
	if legacy, _ := smap.Get(ctx, "telegram:42"); legacy != "" {
		t.Fatalf("legacy unscoped session map key populated: %q", legacy)
	}

	m.clearTurn()
	tg.pushInbound(InboundEvent{
		Platform: "telegram", ChatID: "42", ChatType: "private", UserID: "u1", MsgID: "m2",
		Kind: EventSubmit, Text: "main please",
	})
	waitFor(t, 200*time.Millisecond, func() bool { return len(fk.submitsSnapshot()) == 2 })
	second := fk.submitsSnapshot()[1]
	if second.Model != "gpt-main" {
		t.Fatalf("second submit Model = %q, want default-agent model", second.Model)
	}
	assertContainsAll(t, second.SessionContext,
		"**Agent ID:** `main`",
		"**Agent Binding:** `account`",
		"**Session Key:** `agent:main:telegram:42`",
		"main workspace guidance",
		"main durable user",
	)
	if first.SessionID == second.SessionID {
		t.Fatalf("two routed agents shared session id %q", first.SessionID)
	}
	fk.mu.Lock()
	resets := fk.resets
	fk.mu.Unlock()
	if resets != 1 {
		t.Fatalf("kernel resets = %d, want one reset when switching agent session keys", resets)
	}
}

func TestMultiAgentGatewayRuntime_StatusUsesRoutedAgentSession(t *testing.T) {
	ctx := context.Background()
	smap := session.NewMemMap()
	if err := smap.Put(ctx, "agent:alerts:telegram:42", "sess-alerts"); err != nil {
		t.Fatalf("Put: %v", err)
	}
	if err := smap.PutMetadata(ctx, session.Metadata{SessionID: "sess-alerts", Title: "Alerts Session", CreatedAt: 100, UpdatedAt: 200}); err != nil {
		t.Fatalf("PutMetadata: %v", err)
	}
	m := NewManagerWithSubmitter(ManagerConfig{
		AllowedChats: map[string]string{"telegram": "42"},
		SessionMap:   smap,
		AgentRouting: AgentRoutingConfig{
			Enabled: true,
			Agents: config.AgentsCfg{List: []config.AgentCfg{
				{ID: "main", Default: true},
				{ID: "alerts", Name: "Alerts"},
			}},
			Bindings: []config.AgentBindingCfg{
				{AgentID: "alerts", Match: config.AgentBindingMatchCfg{Channel: "telegram", AccountID: "alerts"}},
			},
		},
	}, &fakeKernel{}, slog.Default())

	status := m.formatGatewayStatus(ctx, InboundEvent{Platform: "telegram", AccountID: "alerts", ChatID: "42", ChatType: "private"})
	assertContainsAll(t, status,
		"**Session ID:** `sess-alerts`",
		"**Title:** Alerts Session",
		"**Agent ID:** `alerts`",
		"**Agent Binding:** `account`",
	)
}

func writeAgentWorkspace(t *testing.T, guidance string) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "AGENTS.md"), []byte(guidance), 0o600); err != nil {
		t.Fatalf("write AGENTS.md: %v", err)
	}
	return dir
}

func writeAgentMemory(t *testing.T, user string) string {
	t.Helper()
	dir := t.TempDir()
	memDir := filepath.Join(dir, "memory")
	if err := os.MkdirAll(memDir, 0o755); err != nil {
		t.Fatalf("mkdir memory: %v", err)
	}
	if err := os.WriteFile(filepath.Join(memDir, "USER.md"), []byte(user), 0o600); err != nil {
		t.Fatalf("write USER.md: %v", err)
	}
	return dir
}

func assertContainsAll(t *testing.T, got string, wants ...string) {
	t.Helper()
	for _, want := range wants {
		if !strings.Contains(got, want) {
			t.Fatalf("missing %q in:\n%s", want, got)
		}
	}
}
