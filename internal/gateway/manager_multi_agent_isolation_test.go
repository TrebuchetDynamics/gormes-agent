//go:build planned

package gateway

import (
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/hermes"
	"github.com/TrebuchetDynamics/gormes-agent/internal/session"
	"github.com/TrebuchetDynamics/gormes-agent/internal/skills"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tools"
)

func TestMultiAgentIsolation_AppliesToolPolicyAndSkillAllowlistPerRoutedAgent(t *testing.T) {
	ctx := context.Background()
	tg := newFakeChannel("telegram")
	fk := &fakeKernel{}

	reg := tools.NewRegistry()
	reg.MustRegister(&tools.MockTool{NameStr: "echo"})
	reg.MustRegister(&tools.MockTool{
		NameStr: "terminal",
		ExecuteFn: func(context.Context, json.RawMessage) (json.RawMessage, error) {
			t.Fatal("terminal tool executed despite agent deny policy")
			return nil, nil
		},
	})

	skillsRoot := t.TempDir()
	writeIsolationSkill(t, skillsRoot, "main-skill", "Main skill", "Main-only instructions.")
	writeIsolationSkill(t, skillsRoot, "alerts-skill", "Alerts skill", "Alerts-only instructions.")
	skillRuntime := skills.NewRuntime(skillsRoot, 8*1024, 5, "")

	m := NewManagerWithSubmitter(ManagerConfig{
		AllowedChats: map[string]string{"telegram": "42"},
		SessionMap:   session.NewMemMap(),
		ToolRegistry: reg,
		SkillRuntime: skillRuntime,
		AgentRouting: AgentRoutingConfig{
			Enabled: true,
			Agents: config.AgentsCfg{List: []config.AgentCfg{
				{
					ID:      "main",
					Default: true,
					Skills:  []string{"main-skill"},
					Tools:   config.AgentToolPolicy{Allow: []string{"echo"}},
				},
				{
					ID:     "alerts",
					Skills: []string{"alerts-skill"},
					Tools:  config.AgentToolPolicy{Deny: []string{"terminal"}},
				},
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
	alertsSubmit := fk.submitsSnapshot()[0]
	if got := registryDescriptorNames(alertsSubmit.Tools); !reflect.DeepEqual(got, []string{"echo"}) {
		t.Fatalf("alerts tool descriptors = %#v, want only echo after terminal deny", got)
	}
	decision := alertsSubmit.ToolSafety.DecideToolCall(hermes.ToolCall{ID: "call-terminal", Name: "terminal", Arguments: json.RawMessage(`{"command":"id"}`)})
	if decision.Allow {
		t.Fatal("alerts terminal call was allowed; want agent policy denial")
	}
	if !strings.Contains(string(decision.Content), "agent_tool_policy_denied") || !strings.Contains(string(decision.Content), `"agent_id":"alerts"`) {
		t.Fatalf("alerts denial payload = %s, want redacted agent policy evidence", decision.Content)
	}
	block, names, err := alertsSubmit.Skills.BuildSkillBlock(ctx, "alerts please")
	if err != nil {
		t.Fatalf("alerts BuildSkillBlock: %v", err)
	}
	if !reflect.DeepEqual(names, []string{"alerts-skill"}) {
		t.Fatalf("alerts skill names = %#v, want alerts-skill only", names)
	}
	if !strings.Contains(block, "Alerts-only instructions.") || strings.Contains(block, "Main-only instructions.") {
		t.Fatalf("alerts skill block not isolated:\n%s", block)
	}

	m.clearTurn()
	tg.pushInbound(InboundEvent{
		Platform: "telegram", ChatID: "42", ChatType: "private", UserID: "u1", MsgID: "m2",
		Kind: EventSubmit, Text: "main please",
	})
	waitFor(t, 200*time.Millisecond, func() bool { return len(fk.submitsSnapshot()) == 2 })
	mainSubmit := fk.submitsSnapshot()[1]
	if got := registryDescriptorNames(mainSubmit.Tools); !reflect.DeepEqual(got, []string{"echo"}) {
		t.Fatalf("main tool descriptors = %#v, want only echo from allow policy", got)
	}
	mainSkillBlock, mainNames, err := mainSubmit.Skills.BuildSkillBlock(ctx, "main please")
	if err != nil {
		t.Fatalf("main BuildSkillBlock: %v", err)
	}
	if !reflect.DeepEqual(mainNames, []string{"main-skill"}) {
		t.Fatalf("main skill names = %#v, want main-skill only", mainNames)
	}
	if !strings.Contains(mainSkillBlock, "Main-only instructions.") || strings.Contains(mainSkillBlock, "Alerts-only instructions.") {
		t.Fatalf("main skill block not isolated:\n%s", mainSkillBlock)
	}
}

func writeIsolationSkill(t *testing.T, root, slug, name, body string) {
	t.Helper()
	dir := filepath.Join(root, "active", slug)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir skill: %v", err)
	}
	raw := "---\nname: " + name + "\ndescription: " + name + " description\n---\n\n" + body + "\n"
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(raw), 0o600); err != nil {
		t.Fatalf("write skill: %v", err)
	}
}

func registryDescriptorNames(reg *tools.Registry) []string {
	if reg == nil {
		return nil
	}
	descs := reg.Descriptors()
	out := make([]string, 0, len(descs))
	for _, desc := range descs {
		out = append(out, desc.Name)
	}
	return out
}
