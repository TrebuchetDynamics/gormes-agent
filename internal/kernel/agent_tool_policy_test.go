package kernel

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/llm"
	"github.com/TrebuchetDynamics/gormes-agent/internal/persistence/store"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/telemetry"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tools"
)

func TestAgentToolPolicyBlocksBeforeExecution(t *testing.T) {
	calls := 0
	reg := tools.NewRegistry()
	reg.MustRegister(&tools.MockTool{
		NameStr: "terminal",
		ExecuteFn: func(context.Context, json.RawMessage) (json.RawMessage, error) {
			calls++
			return json.RawMessage(`{"ran":true}`), nil
		},
	})
	k := newKernelWithRegistry(t, reg)
	k.cfg.ToolSafety = NewAgentToolSafetyPolicy(AgentToolSafetyOptions{
		AgentID: "family",
		Deny:    []string{"terminal"},
	})

	res := k.executeToolCalls(context.Background(), []llm.ToolCall{
		{ID: "call-terminal", Name: "terminal", Arguments: json.RawMessage(`{"command":"id"}`)},
	})
	if calls != 0 {
		t.Fatalf("terminal executed %d times, want blocked before execution", calls)
	}
	if len(res) != 1 || !strings.Contains(res[0].Content, "agent_tool_policy_denied") || !strings.Contains(res[0].Content, `"agent_id":"family"`) {
		t.Fatalf("tool result = %#v, want agent policy denial", res)
	}
}

func TestAgentPlatformEventFiltersPromptVisibleToolDescriptors(t *testing.T) {
	mc := llm.NewMockClient()
	mc.Script([]llm.Event{{Kind: llm.EventDone, FinishReason: "stop"}}, "sess-agent-tools")

	full := tools.NewRegistry()
	full.MustRegister(&tools.MockTool{NameStr: "echo"})
	full.MustRegister(&tools.MockTool{NameStr: "terminal"})
	filtered := full.FilterPolicy([]string{"echo"}, nil)

	k := New(Config{
		Model:           "hermes-agent",
		Endpoint:        "http://mock",
		Admission:       Admission{MaxBytes: 200_000, MaxLines: 10_000},
		Tools:           full,
		MaxToolDuration: 5 * time.Second,
	}, mc, store.NewNoop(), telemetry.New(), nil)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	go k.Run(ctx)
	<-k.Render()
	if err := k.Submit(PlatformEvent{Kind: PlatformEventSubmit, Text: "use tools", Tools: filtered}); err != nil {
		t.Fatal(err)
	}
	waitForFrameMatching(t, k.Render(), func(f RenderFrame) bool {
		return f.Phase == PhaseIdle && f.SessionID != ""
	}, 2*time.Second)

	reqs := mc.Requests()
	if len(reqs) == 0 {
		t.Fatal("mock client received zero requests")
	}
	got := make([]string, 0, len(reqs[0].Tools))
	for _, tool := range reqs[0].Tools {
		got = append(got, tool.Name)
	}
	if len(got) != 1 || got[0] != "echo" {
		t.Fatalf("request tools = %#v, want only echo", got)
	}
}
