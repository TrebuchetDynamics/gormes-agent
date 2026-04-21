package subagent

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/gormes/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/gormes/internal/hermes"
	"github.com/TrebuchetDynamics/gormes-agent/gormes/internal/tools"
)

func TestDelegateTask_EndToEndChildToolLoop(t *testing.T) {
	cli := hermes.NewMockClient()
	cli.Script([]hermes.Event{
		{Kind: hermes.EventDone, FinishReason: "tool_calls", ToolCalls: []hermes.ToolCall{
			{ID: "call-1", Name: "echo", Arguments: json.RawMessage(`{"text":"hello from child"}`)},
		}},
	}, "sess-child")
	cli.Script([]hermes.Event{
		{Kind: hermes.EventToken, Token: "child completed", TokensOut: 2},
		{Kind: hermes.EventDone, FinishReason: "stop", TokensIn: 12, TokensOut: 2},
	}, "sess-child")

	reg := tools.NewRegistry()
	reg.MustRegister(&tools.MockTool{
		NameStr: "echo",
		ExecuteFn: func(ctx context.Context, args json.RawMessage) (json.RawMessage, error) {
			return json.RawMessage(`{"text":"hello from child"}`), nil
		},
	})

	runner := NewChatRunner(cli, reg, ChatRunnerConfig{Model: "hermes-agent", MaxToolDuration: 2 * time.Second})
	mgr := NewManager(config.DelegationCfg{
		DefaultMaxIterations: 8,
		DefaultTimeout:       45 * time.Second,
		MaxChildDepth:        1,
	}, runner, "")

	tool := NewDelegateTool(mgr)
	out, err := tool.Execute(context.Background(), json.RawMessage(`{"goal":"run child","allowed_tools":["echo"]}`))
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}

	var got struct {
		RunID   string `json:"run_id"`
		Status  string `json:"status"`
		Summary string `json:"summary"`
	}
	if err := json.Unmarshal(out, &got); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if got.Status != "completed" || got.Summary != "child completed" || got.RunID == "" {
		t.Fatalf("delegate_task output = %+v, want completed/child completed/non-empty run_id", got)
	}
	if len(cli.Requests()) != 2 {
		t.Fatalf("OpenStream requests = %d, want 2", len(cli.Requests()))
	}
}
