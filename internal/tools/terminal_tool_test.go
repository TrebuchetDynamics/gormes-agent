package tools

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
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

func TestTerminalToolDefaultWorkdirExpandsTilde(t *testing.T) {
	home := t.TempDir()
	project := filepath.Join(home, "project")
	if err := os.MkdirAll(project, 0o755); err != nil {
		t.Fatalf("mkdir project: %v", err)
	}
	t.Setenv("HOME", home)

	tool := NewTerminalTool(TerminalToolConfig{Workdir: "~/project"})
	out := executeTerminalTool(t, tool, `{"command":"pwd"}`)

	if out["workdir"] != project {
		t.Fatalf("workdir = %v, want %q: %#v", out["workdir"], project, out)
	}
	if strings.TrimSpace(asString(out["output"])) != project {
		t.Fatalf("output = %q, want pwd %q", out["output"], project)
	}
}

func TestTerminalToolRecoversWhenConfiguredWorkdirDeleted(t *testing.T) {
	root := t.TempDir()
	deleted := filepath.Join(root, "child", "grandchild")
	if err := os.MkdirAll(deleted, 0o755); err != nil {
		t.Fatalf("mkdir deleted cwd: %v", err)
	}
	if err := os.RemoveAll(filepath.Join(root, "child")); err != nil {
		t.Fatalf("remove cwd parent: %v", err)
	}

	tool := NewTerminalTool(TerminalToolConfig{Workdir: deleted, DefaultTimeout: 5 * time.Second})
	out := executeTerminalTool(t, tool, `{"command":"pwd"}`)

	if out["status"] != "completed" {
		t.Fatalf("status = %v, want completed: %#v", out["status"], out)
	}
	if out["workdir"] != root {
		t.Fatalf("workdir = %v, want nearest existing ancestor %q: %#v", out["workdir"], root, out)
	}
	if strings.TrimSpace(asString(out["output"])) != root {
		t.Fatalf("output = %q, want pwd %q", out["output"], root)
	}
	if out["cwd_recovered"] != true {
		t.Fatalf("cwd_recovered = %v, want true: %#v", out["cwd_recovered"], out)
	}
	if !strings.Contains(asString(out["cwd_recovery"]), "terminal_cwd_recovered") {
		t.Fatalf("cwd_recovery = %v, want terminal_cwd_recovered evidence", out["cwd_recovery"])
	}
}

func TestTerminalToolRecoversWhenProcessCWDDeleted(t *testing.T) {
	original, err := os.Getwd()
	if err != nil {
		t.Fatalf("get original cwd: %v", err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(original); err != nil {
			t.Fatalf("restore cwd: %v", err)
		}
	})

	root := t.TempDir()
	deleted := filepath.Join(root, "wedged")
	if err := os.MkdirAll(deleted, 0o755); err != nil {
		t.Fatalf("mkdir deleted cwd: %v", err)
	}
	if err := os.Chdir(deleted); err != nil {
		t.Fatalf("chdir deleted cwd: %v", err)
	}
	if err := os.RemoveAll(deleted); err != nil {
		t.Fatalf("remove current cwd: %v", err)
	}

	want := filepath.Clean(os.TempDir())
	tool := NewTerminalTool(TerminalToolConfig{DefaultTimeout: 5 * time.Second})
	out := executeTerminalTool(t, tool, `{"command":"pwd"}`)

	if out["status"] != "completed" {
		t.Fatalf("status = %v, want completed: %#v", out["status"], out)
	}
	if out["workdir"] != want {
		t.Fatalf("workdir = %v, want temp dir %q: %#v", out["workdir"], want, out)
	}
	if strings.TrimSpace(asString(out["output"])) != want {
		t.Fatalf("output = %q, want pwd %q", out["output"], want)
	}
	if out["cwd_recovered"] != true {
		t.Fatalf("cwd_recovered = %v, want true: %#v", out["cwd_recovered"], out)
	}
}

func TestTerminalToolExplicitMissingWorkdirStillErrors(t *testing.T) {
	root := t.TempDir()
	tool := NewTerminalTool(TerminalToolConfig{Workdir: root, DefaultTimeout: 5 * time.Second})
	out := executeTerminalTool(t, tool, `{"command":"pwd","workdir":"missing"}`)

	if out["status"] != "error" {
		t.Fatalf("status = %v, want error: %#v", out["status"], out)
	}
	if !strings.Contains(asString(out["error"]), "resolve working directory") {
		t.Fatalf("error = %v, want resolve-working-directory evidence", out["error"])
	}
	if _, ok := out["cwd_recovered"]; ok {
		t.Fatalf("explicit missing workdir should not recover: %#v", out)
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
