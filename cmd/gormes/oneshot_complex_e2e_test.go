package main

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/TrebuchetDynamics/gormes-agent/internal/audit"
	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/hermes"
	"github.com/TrebuchetDynamics/gormes-agent/internal/kernel"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tools"
)

func TestOneshotChatComplexE2E_MultiRoundToolsPrintsOnlyFinalAnswer(t *testing.T) {
	setupOneshotFlagTestEnv(t)

	mock := hermes.NewMockClient()
	mock.Script([]hermes.Event{
		{Kind: hermes.EventToken, Token: "planning text must not reach stdout", TokensOut: 6},
		{
			Kind:         hermes.EventDone,
			FinishReason: "tool_calls",
			ToolCalls: []hermes.ToolCall{
				{ID: "call_alpha", Name: "alpha_probe", Arguments: json.RawMessage(`{"target":"alpha"}`)},
				{ID: "call_beta", Name: "beta_probe", Arguments: json.RawMessage(`{"target":"beta"}`)},
			},
			TokensIn:  30,
			TokensOut: 6,
		},
	}, "sess-complex-round-1")
	mock.Script([]hermes.Event{{
		Kind:         hermes.EventDone,
		FinishReason: "tool_calls",
		ToolCalls: []hermes.ToolCall{
			{ID: "call_gamma", Name: "gamma_probe", Arguments: json.RawMessage(`{"target":"gamma","depends_on":["alpha","beta"]}`)},
		},
		TokensIn:  70,
		TokensOut: 2,
	}}, "sess-complex-round-2")
	mock.Script([]hermes.Event{
		{Kind: hermes.EventToken, Token: "final clean chat answer", TokensOut: 4},
		{Kind: hermes.EventDone, FinishReason: "stop", TokensIn: 100, TokensOut: 4},
	}, "sess-complex-final")

	reg := tools.NewRegistry()
	reg.MustRegister(&tools.MockTool{NameStr: "alpha_probe", ExecuteFn: staticJSONToolResult(`{"alpha":"ready"}`)})
	reg.MustRegister(&tools.MockTool{NameStr: "beta_probe", ExecuteFn: staticJSONToolResult(`{"beta":"ready"}`)})
	reg.MustRegister(&tools.MockTool{NameStr: "gamma_probe", ExecuteFn: staticJSONToolResult(`{"gamma":"alpha+beta complete"}`)})

	cmd := newRootCommandWithRuntime(rootRuntime{
		runTUI: func(*cobra.Command, []string) error {
			t.Fatal("runTUI was called for oneshot")
			return nil
		},
		newOneshotClient: func(context.Context, config.Config, oneshotInvocation) (hermes.Client, error) {
			return mock, nil
		},
		configureOneshotKernel: func(cfg *kernel.Config) {
			cfg.Tools = reg
			cfg.MaxToolIterations = 4
		},
	})

	stdout, stderr, err := executeOneshotFlagCommand(cmd, "--model", "fixture-model", "chat", "-q", "run the three probe workflow")
	if err != nil {
		t.Fatalf("Execute() error = %v\nstderr=%s\nstdout=%s", err, stderr, stdout)
	}
	if stdout != "final clean chat answer\n" {
		t.Fatalf("stdout = %q, want only final answer", stdout)
	}
	if stderr != "" {
		t.Fatalf("stderr = %q, want empty on successful complex oneshot", stderr)
	}
	for _, forbidden := range []string{"planning text", "call_alpha", "call_beta", "call_gamma", "sess-complex", "alpha_probe", "beta_probe", "gamma_probe", "tool_calls"} {
		if strings.Contains(stdout, forbidden) {
			t.Fatalf("stdout leaked %q: %q", forbidden, stdout)
		}
	}

	requests := mock.Requests()
	if len(requests) != 3 {
		t.Fatalf("OpenStream requests = %d, want 3", len(requests))
	}
	assertOneshotToolMessageOrder(t, requests[1].Messages, []string{"call_alpha", "call_beta"})
	assertOneshotToolMessageOrder(t, requests[2].Messages, []string{"call_alpha", "call_beta", "call_gamma"})
	assertOneshotToolMessageContains(t, requests[1].Messages, "call_alpha", `"alpha":"ready"`)
	assertOneshotToolMessageContains(t, requests[1].Messages, "call_beta", `"beta":"ready"`)
	assertOneshotToolMessageContains(t, requests[2].Messages, "call_gamma", `"gamma":"alpha+beta complete"`)

	records := readOneshotAuditRecords(t)
	assertOneshotAuditCompleted(t, records, []string{"alpha_probe", "beta_probe", "gamma_probe"})
}

func TestOneshotChatComplexE2E_ForcedSkillsInjectOnlyAllowlistedProcedures(t *testing.T) {
	setupOneshotFlagTestEnv(t)
	skillsRoot := t.TempDir()
	t.Setenv("GORMES_SKILLS_ROOT", skillsRoot)
	writeChatCommandSkill(t, skillsRoot, "gormes-chat-debugger")
	writeChatCommandSkill(t, skillsRoot, "stdout-leak-auditor")
	writeChatCommandSkill(t, skillsRoot, "unrelated-skill")

	mock := hermes.NewMockClient()
	mock.Script([]hermes.Event{
		{Kind: hermes.EventToken, Token: "skill constrained final", TokensOut: 3},
		{Kind: hermes.EventDone, FinishReason: "stop", TokensIn: 45, TokensOut: 3},
	}, "sess-forced-skills")

	cmd := newRootCommandWithRuntime(rootRuntime{
		runTUI: func(*cobra.Command, []string) error {
			t.Fatal("runTUI was called for forced-skill oneshot")
			return nil
		},
		newOneshotClient: func(context.Context, config.Config, oneshotInvocation) (hermes.Client, error) {
			return mock, nil
		},
	})

	stdout, stderr, err := executeOneshotFlagCommand(
		cmd,
		"--model", "fixture-model",
		"--skills", "gormes-chat-debugger,stdout-leak-auditor",
		"chat", "-q", "debug why chat output leaks tool traces",
	)
	if err != nil {
		t.Fatalf("Execute() error = %v\nstderr=%s\nstdout=%s", err, stderr, stdout)
	}
	if stdout != "skill constrained final\n" {
		t.Fatalf("stdout = %q, want only final answer", stdout)
	}
	if stderr != "" {
		t.Fatalf("stderr = %q, want empty on successful forced-skill chat", stderr)
	}

	requests := mock.Requests()
	if len(requests) != 1 {
		t.Fatalf("OpenStream requests = %d, want 1", len(requests))
	}
	var skillBlock string
	for _, msg := range requests[0].Messages {
		if msg.Role == "system" && strings.Contains(msg.Content, "gormes-chat-debugger") {
			skillBlock = msg.Content
		}
	}
	if skillBlock == "" {
		t.Fatalf("first request missing forced skill block in messages %#v", requests[0].Messages)
	}
	for _, want := range []string{"gormes-chat-debugger", "stdout-leak-auditor", "Use gormes-chat-debugger.", "Use stdout-leak-auditor."} {
		if !strings.Contains(skillBlock, want) {
			t.Fatalf("skill block missing %q\nblock=%s", want, skillBlock)
		}
	}
	if strings.Contains(skillBlock, "unrelated-skill") || strings.Contains(skillBlock, "Use unrelated-skill.") {
		t.Fatalf("skill block included non-allowlisted skill:\n%s", skillBlock)
	}
	for _, forbidden := range []string{"gormes-chat-debugger", "stdout-leak-auditor", "unrelated-skill", "sess-forced-skills"} {
		if strings.Contains(stdout, forbidden) {
			t.Fatalf("stdout leaked forced-skill/provider internals %q: %q", forbidden, stdout)
		}
	}
}

func TestOneshotChatComplexE2E_ProviderSetupFailureRedactsAPIKeyFlag(t *testing.T) {
	setupOneshotFlagTestEnv(t)
	secret := "sk-live-super-secret-chat-token"

	cmd := newRootCommandWithRuntime(rootRuntime{
		runTUI: func(*cobra.Command, []string) error {
			t.Fatal("runTUI was called for oneshot provider failure")
			return nil
		},
		newOneshotClient: func(context.Context, config.Config, oneshotInvocation) (hermes.Client, error) {
			return nil, errors.New("upstream rejected Authorization Bearer " + secret)
		},
	})

	stdout, stderr, err := executeOneshotFlagCommand(cmd, "--model", "fixture-model", "--api-key", secret, "chat", "-q", "hello")
	if err == nil {
		t.Fatalf("Execute() error = nil, want provider setup failure\nstdout=%s\nstderr=%s", stdout, stderr)
	}
	if code := exitCodeFromError(err); code != 1 {
		t.Fatalf("exit code = %d, want 1 (err=%v)", code, err)
	}
	if stdout != "" {
		t.Fatalf("stdout = %q, want empty on provider setup failure", stdout)
	}
	for _, want := range []string{"gormes chat -q: provider setup failed", "[REDACTED]"} {
		if !strings.Contains(stderr, want) {
			t.Fatalf("stderr missing %q\nstderr=%s", want, stderr)
		}
	}
	if strings.Contains(stderr, secret) || strings.Contains(stderr, "Bearer "+secret) {
		t.Fatalf("stderr leaked API key:\n%s", stderr)
	}
	for _, forbidden := range []string{"api_server", "gateway", "session_id", "sess-"} {
		if strings.Contains(stdout, forbidden) || strings.Contains(stderr, forbidden) {
			t.Fatalf("failure output leaked %q\nstdout=%s\nstderr=%s", forbidden, stdout, stderr)
		}
	}
}

func staticJSONToolResult(body string) func(context.Context, json.RawMessage) (json.RawMessage, error) {
	return func(context.Context, json.RawMessage) (json.RawMessage, error) {
		return json.RawMessage(body), nil
	}
}

func assertOneshotToolMessageOrder(t *testing.T, messages []hermes.Message, want []string) {
	t.Helper()
	var got []string
	for _, msg := range messages {
		if msg.Role == "tool" {
			got = append(got, msg.ToolCallID)
		}
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("tool message order = %#v, want %#v in messages %#v", got, want, messages)
	}
}

func assertOneshotToolMessageContains(t *testing.T, messages []hermes.Message, callID, want string) {
	t.Helper()
	for _, msg := range messages {
		if msg.Role == "tool" && msg.ToolCallID == callID {
			if !strings.Contains(msg.Content, want) {
				t.Fatalf("tool message %q content = %q, want substring %q", callID, msg.Content, want)
			}
			return
		}
	}
	t.Fatalf("tool message %q not found in %#v", callID, messages)
}

func assertOneshotAuditCompleted(t *testing.T, records []audit.Record, wantTools []string) {
	t.Helper()
	seen := map[string]bool{}
	for _, rec := range records {
		if rec.Status == "completed" {
			seen[rec.Tool] = true
		}
	}
	for _, tool := range wantTools {
		if !seen[tool] {
			t.Fatalf("completed audit for %q missing in %#v", tool, records)
		}
	}
}
