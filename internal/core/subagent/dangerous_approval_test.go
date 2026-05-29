package subagent

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/tools"
)

type dangerousApprovalExecutor struct {
	called bool
}

func (e *dangerousApprovalExecutor) Execute(ctx context.Context, req tools.ToolRequest) (<-chan tools.ToolEvent, error) {
	e.called = true
	ch := make(chan tools.ToolEvent, 2)
	ch <- tools.ToolEvent{Type: "output", Output: json.RawMessage(`{"ok":true}`)}
	ch <- tools.ToolEvent{Type: "completed"}
	close(ch)
	return ch, nil
}

func TestSubagentDangerousCommand_DefaultDeniesNoninteractive(t *testing.T) {
	exec := &dangerousApprovalExecutor{}
	_, info, err := executeChildTool(context.Background(), SubagentConfig{
		EnabledTools: []string{"terminal"},
		toolExecutor: exec,
	}, nil, tools.ToolRequest{ToolName: "terminal", Input: json.RawMessage(`{"command":"git reset --hard"}`)})
	if err == nil {
		t.Fatalf("executeChildTool dangerous terminal err = nil, want noninteractive denial")
	}
	if !errors.Is(err, ErrSubagentApprovalDenied) {
		t.Fatalf("executeChildTool err = %v, want ErrSubagentApprovalDenied", err)
	}
	if exec.called {
		t.Fatalf("dangerous terminal executor was called; want denial before tool execution")
	}
	if info.Status != "approval_denied_noninteractive" {
		t.Fatalf("tool status = %q, want approval_denied_noninteractive", info.Status)
	}
}

func TestSubagentDangerousCommand_ExplicitOffApprovesRecoverable(t *testing.T) {
	exec := &dangerousApprovalExecutor{}
	out, info, err := executeChildTool(context.Background(), SubagentConfig{
		EnabledTools:                 []string{"terminal"},
		DangerousCommandApprovalMode: "off",
		toolExecutor:                 exec,
	}, nil, tools.ToolRequest{ToolName: "terminal", Input: json.RawMessage(`{"command":"git reset --hard"}`)})
	if err != nil {
		t.Fatalf("executeChildTool with explicit off err = %v, want nil", err)
	}
	if !exec.called {
		t.Fatalf("dangerous recoverable terminal executor was not called with explicit off mode")
	}
	if info.Status != "completed" {
		t.Fatalf("tool status = %q, want completed", info.Status)
	}
	if string(out) != `{"ok":true}` {
		t.Fatalf("output = %s, want fake executor output", out)
	}
}

func TestSubagentDangerousCommand_HardlineBlockedEvenWithOff(t *testing.T) {
	exec := &dangerousApprovalExecutor{}
	_, info, err := executeChildTool(context.Background(), SubagentConfig{
		EnabledTools:                 []string{"terminal"},
		DangerousCommandApprovalMode: "off",
		toolExecutor:                 exec,
	}, nil, tools.ToolRequest{ToolName: "terminal", Input: json.RawMessage(`{"command":"rm -rf /"}`)})
	if err == nil {
		t.Fatalf("executeChildTool hardline err = nil, want denial")
	}
	if !errors.Is(err, ErrSubagentApprovalDenied) {
		t.Fatalf("executeChildTool err = %v, want ErrSubagentApprovalDenied", err)
	}
	if exec.called {
		t.Fatalf("hardline executor was called; want block before tool execution")
	}
	if info.Status != "approval_denied_noninteractive" {
		t.Fatalf("tool status = %q, want approval_denied_noninteractive", info.Status)
	}
}

func TestSubagentDangerousCommand_BenignCommandStillRuns(t *testing.T) {
	exec := &dangerousApprovalExecutor{}
	_, info, err := executeChildTool(context.Background(), SubagentConfig{
		EnabledTools: []string{"terminal"},
		toolExecutor: exec,
	}, nil, tools.ToolRequest{ToolName: "terminal", Input: json.RawMessage(`{"command":"go test ./internal/subagent -count=1"}`)})
	if err != nil {
		t.Fatalf("executeChildTool benign err = %v, want nil", err)
	}
	if !exec.called {
		t.Fatalf("benign terminal executor was not called")
	}
	if info.Status != "completed" {
		t.Fatalf("tool status = %q, want completed", info.Status)
	}
}
