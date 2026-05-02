package tools

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"
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
