package tools

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestExecuteCodeToolModeCutoverDefaultStrict(t *testing.T) {
	tool := NewExecuteCodeTool()
	if tool.Mode != ExecuteCodeModeStrict {
		t.Fatalf("Mode = %q, want %q", tool.Mode, ExecuteCodeModeStrict)
	}
	if _, ok := tool.Sandbox.(*StrictModeSandbox); !ok {
		t.Fatalf("Sandbox = %T, want *StrictModeSandbox", tool.Sandbox)
	}

	raw, err := tool.Execute(context.Background(), json.RawMessage(`{"language":"sh","code":"curl https://example.com"}`))
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	var got map[string]any
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("unmarshal execute_code result: %v", err)
	}
	if got["status"] != "blocked" {
		t.Fatalf("status = %v, want blocked; raw=%s", got["status"], raw)
	}

	ref, err := json.Marshal(StrictModeBlockedEnvelopeShape())
	if err != nil {
		t.Fatalf("marshal strict envelope reference: %v", err)
	}
	var refMap map[string]any
	if err := json.Unmarshal(ref, &refMap); err != nil {
		t.Fatalf("unmarshal strict envelope reference: %v", err)
	}
	for key := range refMap {
		if _, ok := got[key]; !ok {
			t.Fatalf("blocked result missing strict envelope key %q: %s", key, raw)
		}
	}
}

func TestExecuteCodeToolModeCutoverProjectOptIn(t *testing.T) {
	projectDir := t.TempDir()
	projectSandbox := newProjectModeSandboxWithHooks(projectModeSandboxHooks{
		lookupEnv: func(key string) (string, bool) {
			if key == "TERMINAL_CWD" {
				return projectDir, true
			}
			return "", false
		},
		getwd: func() (string, error) {
			return "/fallback/process", nil
		},
		isDir: func(path string) bool {
			return path == projectDir || strings.Contains(path, "gormes-execute-code-")
		},
		lookPath: func(string) (string, error) {
			return "/bin/sh", nil
		},
	})

	tool := NewExecuteCodeTool(ExecuteCodeToolConfig{
		ConfigSet:      true,
		ConfigValue:    "project",
		DefaultMode:    ExecuteCodeModeStrict,
		ProjectSandbox: projectSandbox,
	})
	if tool.Mode != ExecuteCodeModeProject {
		t.Fatalf("Mode = %q, want %q", tool.Mode, ExecuteCodeModeProject)
	}
	if tool.Sandbox != projectSandbox {
		t.Fatalf("Sandbox = %T, want injected project sandbox", tool.Sandbox)
	}

	raw, err := tool.Execute(context.Background(), json.RawMessage(`{"language":"sh","code":"printf '%s' \"$PWD\"","timeout_ms":5000}`))
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	var result CodeExecutionResult
	if err := json.Unmarshal(raw, &result); err != nil {
		t.Fatalf("unmarshal result: %v; raw=%s", err, raw)
	}
	if result.Status != "success" {
		t.Fatalf("status = %q, want success; error=%q", result.Status, result.Error)
	}
	if strings.TrimSpace(result.Stdout) != projectDir {
		t.Fatalf("stdout cwd = %q, want %q", strings.TrimSpace(result.Stdout), projectDir)
	}

	blocked, err := tool.Execute(context.Background(), json.RawMessage(`{"language":"sh","code":"ls /tmp","timeout_ms":5000}`))
	if err != nil {
		t.Fatalf("Execute blocked command: %v", err)
	}
	var blockedResult CodeExecutionResult
	if err := json.Unmarshal(blocked, &blockedResult); err != nil {
		t.Fatalf("unmarshal blocked result: %v; raw=%s", err, blocked)
	}
	if blockedResult.Status != "blocked" || blockedResult.FilesystemAccess || blockedResult.NetworkAccess {
		t.Fatalf("blocked result = %+v, want blocked with strict access flags preserved", blockedResult)
	}
}

func TestExecuteCodeToolModeCutoverInvalidConfigFallsBackStrict(t *testing.T) {
	strictSandbox := newStrictModeSandboxWithLookPath(func(string) (string, error) {
		return "/bin/sh", nil
	})
	tool := NewExecuteCodeTool(ExecuteCodeToolConfig{
		ConfigSet:     true,
		ConfigValue:   "sk-live-secret-value",
		DefaultMode:   ExecuteCodeModeStrict,
		StrictSandbox: strictSandbox,
	})
	if tool.Mode != ExecuteCodeModeStrict {
		t.Fatalf("Mode = %q, want %q", tool.Mode, ExecuteCodeModeStrict)
	}
	if tool.Sandbox != strictSandbox {
		t.Fatalf("Sandbox = %T, want injected strict sandbox", tool.Sandbox)
	}
	if !hasExecuteCodeModeEvidence(tool.ModeEvidence, ExecuteCodeModeEvidenceInvalid) {
		t.Fatalf("ModeEvidence = %#v, want invalid config evidence", tool.ModeEvidence)
	}
	for _, ev := range tool.ModeEvidence {
		if strings.Contains(ev.Message, "sk-live-secret-value") {
			t.Fatalf("ModeEvidence leaked raw invalid config: %#v", ev)
		}
	}

	result, err := strictSandbox.Execute(context.Background(), CodeExecutionRequest{
		Language: "sh",
		Code:     "echo ok",
		Timeout:  5 * time.Second,
	})
	if err != nil {
		t.Fatalf("strict sandbox execute: %v", err)
	}
	if result.Status != "success" {
		t.Fatalf("strict sandbox status = %q, want success; error=%q", result.Status, result.Error)
	}
}
