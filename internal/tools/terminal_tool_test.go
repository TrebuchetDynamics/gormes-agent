package tools

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestTerminalToolRunsForegroundCommand(t *testing.T) {
	tool := NewTerminalTool(TerminalToolConfig{Workdir: t.TempDir(), DefaultTimeout: 5 * time.Second})
	out := executeTerminalTool(t, tool, `{"command":"printf hello"}`)

	if out["status"] != "completed" {
		t.Fatalf("status = %v, want completed: %#v", out["status"], out)
	}
	if out["exit_code"] != float64(0) {
		t.Fatalf("exit_code = %v, want 0", out["exit_code"])
	}
	if out["output"] != "hello" {
		t.Fatalf("output = %q, want hello", out["output"])
	}
}

func TestTerminalToolBlocksDangerousCommandWithoutApproval(t *testing.T) {
	tool := NewTerminalTool(TerminalToolConfig{Workdir: t.TempDir(), ApprovalMode: ApprovalModeManual})
	out := executeTerminalTool(t, tool, `{"command":"git reset --hard"}`)

	if out["status"] != "approval_required" {
		t.Fatalf("status = %v, want approval_required: %#v", out["status"], out)
	}
	if !strings.Contains(asString(out["error"]), "git reset --hard") {
		t.Fatalf("error = %v, want dangerous-command description", out["error"])
	}
}

func TestTerminalToolHardBlocksPythonRuntimeEvenWhenApprovalsOff(t *testing.T) {
	tool := NewTerminalTool(TerminalToolConfig{Workdir: t.TempDir(), ApprovalMode: ApprovalModeOff})
	out := executeTerminalTool(t, tool, `{"command":"python3 - <<'PY'\nimport urllib.request\nPY"}`)

	if out["status"] != "blocked" {
		t.Fatalf("status = %v, want blocked: %#v", out["status"], out)
	}
	if !strings.Contains(asString(out["error"]), "Python runtime execution is disabled") {
		t.Fatalf("error = %v, want Python hardline description", out["error"])
	}
	if out["exit_code"] != float64(-1) {
		t.Fatalf("exit_code = %v, want -1", out["exit_code"])
	}
}

func TestTerminalToolRejectsBackgroundUntilProcessRegistryPort(t *testing.T) {
	tool := NewTerminalTool(TerminalToolConfig{Workdir: t.TempDir()})
	out := executeTerminalTool(t, tool, `{"command":"sleep 10","background":true}`)

	if out["status"] != "unsupported" {
		t.Fatalf("status = %v, want unsupported: %#v", out["status"], out)
	}
	if !strings.Contains(asString(out["error"]), "background") {
		t.Fatalf("error = %v, want background guidance", out["error"])
	}
}

func executeTerminalTool(t *testing.T, tool *TerminalTool, args string) map[string]any {
	t.Helper()
	raw, err := tool.Execute(context.Background(), json.RawMessage(args))
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("unmarshal %s: %v", raw, err)
	}
	return out
}
