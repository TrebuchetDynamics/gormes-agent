package tools

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/toolcompact"
)

type fakeCodeSandbox struct {
	req    CodeExecutionRequest
	result CodeExecutionResult
	err    error
}

func (f *fakeCodeSandbox) Execute(_ context.Context, req CodeExecutionRequest) (CodeExecutionResult, error) {
	f.req = req
	return f.result, f.err
}

func TestExecuteCodeTool_UsesRequestedShellLanguage(t *testing.T) {
	sandbox := &fakeCodeSandbox{
		result: CodeExecutionResult{Status: "success", Language: "sh"},
	}
	tool := &ExecuteCodeTool{Sandbox: sandbox}

	out, err := tool.Execute(context.Background(), json.RawMessage(`{"language":"sh","code":"printf hi"}`))
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if sandbox.req.Language != "sh" {
		t.Fatalf("sandbox language = %q, want sh", sandbox.req.Language)
	}
	if !strings.Contains(string(out), `"language":"sh"`) {
		t.Fatalf("output = %s, want language field", out)
	}
}

func TestExecuteCodeTool_DefaultsCodeOnlyCallsToShell(t *testing.T) {
	sandbox := &fakeCodeSandbox{
		result: CodeExecutionResult{Status: "success", Language: "sh"},
	}
	tool := &ExecuteCodeTool{Sandbox: sandbox}

	_, err := tool.Execute(context.Background(), json.RawMessage(`{"code":"printf hi"}`))
	if err != nil {
		t.Fatalf("Execute code-only args: %v", err)
	}
	if sandbox.req.Language != "sh" {
		t.Fatalf("sandbox language = %q, want sh default", sandbox.req.Language)
	}

	var schema struct {
		Required []string `json:"required"`
	}
	if err := json.Unmarshal(tool.Schema(), &schema); err != nil {
		t.Fatalf("Schema invalid JSON: %v\n%s", err, tool.Schema())
	}
	if !containsString(schema.Required, "code") {
		t.Fatalf("schema required = %v, want code", schema.Required)
	}
	if containsString(schema.Required, "language") {
		t.Fatalf("schema required = %v, language must be optional for Hermes code-only compatibility", schema.Required)
	}
	if strings.Contains(strings.ToLower(string(tool.Schema())), "python") {
		t.Fatalf("schema = %s, must not advertise Python", tool.Schema())
	}
	if strings.Contains(strings.ToLower(tool.Description()), "python") {
		t.Fatalf("description = %q, must not advertise Python", tool.Description())
	}
}

func TestExecuteCodeToolRejectsPythonLanguage(t *testing.T) {
	sandbox := &fakeCodeSandbox{}
	tool := &ExecuteCodeTool{Sandbox: sandbox}

	raw, err := tool.Execute(context.Background(), json.RawMessage(`{"language":"python","code":"print('hi')"}`))
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if sandbox.req.Language != "" {
		t.Fatalf("sandbox language = %q, want sandbox not called", sandbox.req.Language)
	}
	var out CodeExecutionResult
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("unmarshal %s: %v", raw, err)
	}
	if out.Status != "blocked" {
		t.Fatalf("status = %q, want blocked: %#v", out.Status, out)
	}
	if !strings.Contains(out.Error, "Python runtime execution is disabled") {
		t.Fatalf("error = %q, want Python disabled message", out.Error)
	}
}

func TestExecuteCodeTool_CompactsLargeStdoutWhenOptedIn(t *testing.T) {
	var stdout strings.Builder
	for i := 0; i < 30; i++ {
		stdout.WriteString("ok  \tgithub.com/example/project/pkg\t0.001s\n")
	}
	stdout.WriteString("--- FAIL: TestWidgetHandlesOverflow (0.00s)\n")
	stdout.WriteString("    widget_test.go:42: got overflow=false, want true\n")
	stdout.WriteString("FAIL\n")
	stdout.WriteString("FAIL\tgithub.com/example/project/widget\t0.123s\n")

	sandbox := &fakeCodeSandbox{
		result: CodeExecutionResult{Status: "error", Language: "sh", ExitCode: 1, Stdout: stdout.String()},
	}
	tool := &ExecuteCodeTool{
		Sandbox: sandbox,
		OutputCompaction: toolcompact.Config{
			Mode:           toolcompact.ModeAuto,
			ThresholdBytes: 128,
			HeadLines:      2,
			TailLines:      2,
		},
	}

	raw, err := tool.Execute(context.Background(), json.RawMessage(`{"code":"go test ./..."}`))
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	var out CodeExecutionResult
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("unmarshal %s: %v", raw, err)
	}
	if !strings.Contains(out.Stdout, "TestWidgetHandlesOverflow") || !strings.Contains(out.Stdout, "widget_test.go:42") {
		t.Fatalf("compacted stdout lost diagnostics:\n%s", out.Stdout)
	}
	if strings.Contains(out.Stdout, "github.com/example/project/pkg\t0.001s") {
		t.Fatalf("compacted stdout kept noisy passing package wall:\n%s", out.Stdout)
	}
	if out.Compaction == nil || out.Compaction.Stdout == nil {
		t.Fatalf("compaction evidence missing: %#v", out.Compaction)
	}
	if !out.Compaction.Stdout.Applied || out.Compaction.Stdout.Reducer != toolcompact.ReducerGoTest {
		t.Fatalf("stdout compaction = %#v, want applied go_test", out.Compaction.Stdout)
	}
	if out.Compaction.Stdout.OriginalBytes <= out.Compaction.Stdout.CompactedBytes {
		t.Fatalf("stdout bytes = %#v, want reduction", out.Compaction.Stdout)
	}
}

func TestExecuteCodeTool_FullOutputBypassesCompaction(t *testing.T) {
	stdout := strings.Repeat("ok  \tgithub.com/example/project/pkg\t0.001s\n", 30)
	sandbox := &fakeCodeSandbox{
		result: CodeExecutionResult{Status: "success", Language: "sh", Stdout: stdout},
	}
	tool := &ExecuteCodeTool{
		Sandbox: sandbox,
		OutputCompaction: toolcompact.Config{
			Mode:           toolcompact.ModeAuto,
			ThresholdBytes: 128,
		},
	}

	raw, err := tool.Execute(context.Background(), json.RawMessage(`{"code":"go test ./...","full_output":true}`))
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	var out CodeExecutionResult
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("unmarshal %s: %v", raw, err)
	}
	if out.Stdout != stdout {
		t.Fatal("full_output changed stdout")
	}
	if out.Compaction != nil {
		t.Fatalf("compaction = %#v, want nil for full_output bypass", out.Compaction)
	}
}

func TestExecuteCodeToolStrictModeUsesProfileSubprocessHome(t *testing.T) {
	operatorHome := t.TempDir()
	profileRoot := t.TempDir()
	profileHome := filepath.Join(profileRoot, "home")
	if err := os.MkdirAll(profileHome, 0o700); err != nil {
		t.Fatalf("mkdir profile home: %v", err)
	}
	t.Setenv("HOME", operatorHome)

	tool := NewExecuteCodeTool(ExecuteCodeToolConfig{
		DefaultMode: ExecuteCodeModeStrict,
		SubprocessHome: func() (string, bool) {
			return profileHome, true
		},
	})
	raw, err := tool.Execute(context.Background(), json.RawMessage(`{"language":"sh","code":"printf '%s' \"$HOME\""}`))
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	var out CodeExecutionResult
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("unmarshal %s: %v", raw, err)
	}
	if out.Status != "success" {
		t.Fatalf("status = %q, want success: %#v", out.Status, out)
	}
	if strings.TrimSpace(out.Stdout) != profileHome {
		t.Fatalf("HOME = %q, want profile subprocess home %q", out.Stdout, profileHome)
	}
	if strings.Contains(out.Stdout, operatorHome) {
		t.Fatalf("execute_code leaked operator HOME %q in stdout %q", operatorHome, out.Stdout)
	}
}

func TestExecuteCodeToolProjectModeUsesProfileSubprocessHome(t *testing.T) {
	operatorHome := t.TempDir()
	profileRoot := t.TempDir()
	profileHome := filepath.Join(profileRoot, "home")
	if err := os.MkdirAll(profileHome, 0o700); err != nil {
		t.Fatalf("mkdir profile home: %v", err)
	}
	t.Setenv("HOME", operatorHome)

	tool := NewExecuteCodeTool(ExecuteCodeToolConfig{
		DefaultMode: ExecuteCodeModeProject,
		SubprocessHome: func() (string, bool) {
			return profileHome, true
		},
	})
	raw, err := tool.Execute(context.Background(), json.RawMessage(`{"language":"sh","code":"printf '%s' \"$HOME\""}`))
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	var out CodeExecutionResult
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("unmarshal %s: %v", raw, err)
	}
	if out.Status != "success" {
		t.Fatalf("status = %q, want success: %#v", out.Status, out)
	}
	if strings.TrimSpace(out.Stdout) != profileHome {
		t.Fatalf("HOME = %q, want profile subprocess home %q", out.Stdout, profileHome)
	}
}

func TestExecuteCodeToolProjectModeFailsClosedWithProfileWorkspaceAllowList(t *testing.T) {
	root := t.TempDir()
	project := filepath.Join(root, "project")
	if err := os.MkdirAll(project, 0o755); err != nil {
		t.Fatalf("mkdir project: %v", err)
	}
	scope, err := NewProfileWorkspaceScope(ProfileWorkspaceScopeOptions{
		ProjectRoots: []string{project},
		OperatorHome: root,
	})
	if err != nil {
		t.Fatalf("NewProfileWorkspaceScope: %v", err)
	}

	tool := NewExecuteCodeTool(ExecuteCodeToolConfig{
		DefaultMode:    ExecuteCodeModeProject,
		WorkspaceScope: scope,
	})
	raw, err := tool.Execute(context.Background(), json.RawMessage(`{"language":"sh","code":"printf should-not-run"}`))
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	out := decodeCodeExecutionResult(t, raw)
	if out.Status != "blocked" {
		t.Fatalf("status = %q, want blocked: %#v", out.Status, out)
	}
	if out.Evidence != ProfileWorkspaceScopeViolation {
		t.Fatalf("evidence = %q, want %q: %#v", out.Evidence, ProfileWorkspaceScopeViolation, out)
	}
	if !strings.Contains(out.Error, "fail closed") {
		t.Fatalf("error = %q, want fail-closed guidance", out.Error)
	}
}

func TestExecuteCodeToolStrictModeStillRunsWithProfileWorkspaceAllowList(t *testing.T) {
	root := t.TempDir()
	project := filepath.Join(root, "project")
	if err := os.MkdirAll(project, 0o755); err != nil {
		t.Fatalf("mkdir project: %v", err)
	}
	scope, err := NewProfileWorkspaceScope(ProfileWorkspaceScopeOptions{
		ProjectRoots: []string{project},
		OperatorHome: root,
	})
	if err != nil {
		t.Fatalf("NewProfileWorkspaceScope: %v", err)
	}

	tool := NewExecuteCodeTool(ExecuteCodeToolConfig{
		DefaultMode:    ExecuteCodeModeStrict,
		WorkspaceScope: scope,
	})
	raw, err := tool.Execute(context.Background(), json.RawMessage(`{"language":"sh","code":"printf strict-ok"}`))
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	out := decodeCodeExecutionResult(t, raw)
	if out.Status != "success" {
		t.Fatalf("status = %q, want success: %#v", out.Status, out)
	}
	if strings.TrimSpace(out.Stdout) != "strict-ok" {
		t.Fatalf("stdout = %q, want strict-ok", out.Stdout)
	}
}

func TestLocalCodeSandbox_TruncatesStdoutAndStderr(t *testing.T) {
	sandbox := NewLocalCodeSandbox()

	result, err := sandbox.Execute(context.Background(), CodeExecutionRequest{
		Language:         "sh",
		Code:             `printf 'stdout-limit'; printf 'stderr-limit' >&2`,
		StdoutLimitBytes: 6,
		StderrLimitBytes: 5,
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if result.Status != "success" {
		t.Fatalf("status = %q, want success", result.Status)
	}
	if !result.StdoutTruncated || !strings.Contains(result.Stdout, "[truncated at 6 bytes]") {
		t.Fatalf("stdout = %q, want truncation marker", result.Stdout)
	}
	if !result.StderrTruncated || !strings.Contains(result.Stderr, "[truncated at 5 bytes]") {
		t.Fatalf("stderr = %q, want truncation marker", result.Stderr)
	}
}

func TestLocalCodeSandbox_TimesOut(t *testing.T) {
	sandbox := NewLocalCodeSandbox()

	result, err := sandbox.Execute(context.Background(), CodeExecutionRequest{
		Language: "sh",
		Code:     `sleep 1`,
		Timeout:  20 * time.Millisecond,
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if result.Status != "timeout" {
		t.Fatalf("status = %q, want timeout", result.Status)
	}
	if result.Error == "" || !strings.Contains(result.Error, "timed out") {
		t.Fatalf("error = %q, want timeout detail", result.Error)
	}
}

func TestLocalCodeSandbox_ContextCancelReturnsInterrupted(t *testing.T) {
	sandbox := NewLocalCodeSandbox()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	result, err := sandbox.Execute(ctx, CodeExecutionRequest{
		Language: "sh",
		Code:     `sleep 5`,
		Timeout:  5 * time.Second,
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if result.Status != "interrupted" {
		t.Fatalf("status = %q, want interrupted", result.Status)
	}
	if result.Error == "" || !strings.Contains(result.Error, "interrupted") {
		t.Fatalf("error = %q, want interrupted detail", result.Error)
	}
}

func TestLocalCodeSandbox_BlocksFilesystemAccess(t *testing.T) {
	sandbox := NewLocalCodeSandbox()

	result, err := sandbox.Execute(context.Background(), CodeExecutionRequest{
		Language: "sh",
		Code:     `touch blocked.txt`,
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if result.Status != "blocked" {
		t.Fatalf("status = %q, want blocked", result.Status)
	}
	if !strings.Contains(strings.ToLower(result.Error), "filesystem") {
		t.Fatalf("error = %q, want filesystem guard detail", result.Error)
	}
}

func TestLocalCodeSandbox_BlocksNetworkAccess(t *testing.T) {
	sandbox := NewLocalCodeSandbox()

	result, err := sandbox.Execute(context.Background(), CodeExecutionRequest{
		Language: "sh",
		Code:     `curl https://example.com`,
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if result.Status != "blocked" {
		t.Fatalf("status = %q, want blocked", result.Status)
	}
	if !strings.Contains(strings.ToLower(result.Error), "network") {
		t.Fatalf("error = %q, want network guard detail", result.Error)
	}
}
