package kernel

import (
	"context"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/kernel/testfixtures"
	"github.com/TrebuchetDynamics/gormes-agent/internal/llm"
	"github.com/TrebuchetDynamics/gormes-agent/internal/persistence/store"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/telemetry"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tools"
)

func TestKernelTurnRequestAssemblyPreservesGuidancePrefillToolAndUsageOrder(t *testing.T) {
	registry := tools.NewRegistry()
	registry.MustRegister(&tools.MockTool{NameStr: "session_search"})
	registry.MustRegister(&tools.MockTool{NameStr: "web_search"})

	recall := &testfixtures.RecallProvider{ReturnContent: "<memory-context>remembered</memory-context>"}
	skills := &testfixtures.SkillProvider{
		Block: "<skills>\n## gormes-tdd-slice\nUse TDD.\n</skills>",
		Names: []string{"gormes-tdd-slice"},
	}
	usage := &testfixtures.SkillUsageRecorder{}

	k := New(Config{
		Model:          "gpt-5.5-codex",
		Endpoint:       "http://mock",
		SystemPrompt:   llm.DefaultAgentIdentity,
		Tools:          registry,
		Recall:         recall,
		RecallDeadline: time.Second,
		ChatKey:        "telegram:42",
		Skills:         skills,
		SkillUsage:     usage,
		PrefillMessages: []llm.Message{
			{Role: "user", Content: "example request"},
			{Role: "assistant", Content: "example answer"},
		},
	}, llm.NewMockClient(), store.NewNoop(), telemetry.New(), nil)
	k.sessionID = "sess-request-assembly"

	request := k.buildTurnRequest(context.Background(), turnRequestAssemblyInput{
		Model:          "gpt-5.5-codex",
		SessionID:      k.sessionID,
		UserText:       "ship the slice",
		UserMessage:    llm.Message{Role: "user", Content: "ship the slice"},
		SessionContext: "<session-context>active</session-context>",
		Reasoning:      llm.ReasoningEffortEvidence{Forwarded: true, Effort: "medium"},
	})

	if request.Model != "gpt-5.5-codex" || request.SessionID != "sess-request-assembly" || !request.Stream {
		t.Fatalf("request metadata = %#v, want selected model/session streaming request", request)
	}
	if request.ReasoningEffort == nil || *request.ReasoningEffort != "medium" {
		t.Fatalf("ReasoningEffort = %#v, want forwarded medium", request.ReasoningEffort)
	}
	if !hasToolDescriptor(request.Tools, "session_search") || !hasToolDescriptor(request.Tools, "web_search") {
		t.Fatalf("tools = %#v, want registry tool descriptors", request.Tools)
	}

	messages := request.Messages
	wantRoles := []string{"system", "system", "system", "system", "system", "system", "system", "system", "system", "user", "assistant", "user"}
	if gotRoles := messageRoles(messages); !reflect.DeepEqual(gotRoles, wantRoles) {
		t.Fatalf("message roles = %#v, want %#v\nmessages=%#v", gotRoles, wantRoles, messages)
	}
	assertMessageContains(t, messages[0], "You are Gorm")
	assertMessageContains(t, messages[0], "run by gormes")
	assertMessageContains(t, messages[1], "<session-context>active</session-context>")
	assertMessageContains(t, messages[2], llm.SessionSearchGuidance)
	assertMessageContains(t, messages[3], llm.ToolUseEnforcementGuidance)
	assertMessageContains(t, messages[4], llm.OpenAIModelExecutionGuidance)
	assertMessageContains(t, messages[5], llm.ResearchQualityGuidance)
	assertMessageContains(t, messages[6], llm.MemoryGuidance)
	assertMessageContains(t, messages[6], "<memory-context>remembered</memory-context>")
	assertMessageContains(t, messages[7], llm.SkillsGuidance)
	assertMessageContains(t, messages[8], "gormes-tdd-slice")
	if messages[9].Content != "example request" || messages[10].Content != "example answer" || messages[11].Content != "ship the slice" {
		t.Fatalf("tail messages = %#v, want prefill before current user", messages[9:])
	}
	if recall.LastInput.UserMessage != "ship the slice" || recall.LastInput.ChatKey != "telegram:42" || recall.LastInput.SessionID != "sess-request-assembly" {
		t.Fatalf("recall input = %#v, want user/chat/session evidence", recall.LastInput)
	}
	if usage.Calls != 1 || !reflect.DeepEqual(usage.Got, [][]string{{"gormes-tdd-slice"}}) {
		t.Fatalf("skill usage = calls %d got %#v, want one recorded selected skill", usage.Calls, usage.Got)
	}
}

// TestKernelTurnRequestAssemblyIncludesPriorTurnHistory proves that messages
// from prior turns are included in the assembled provider request, matching
// Hermes conversation_loop.py which sends full api_messages history.
func TestKernelTurnRequestAssemblyIncludesPriorTurnHistory(t *testing.T) {
	k := New(Config{
		Model:    "claude-3-5-sonnet",
		Endpoint: "http://mock",
	}, llm.NewMockClient(), store.NewNoop(), telemetry.New(), nil)
	k.sessionID = "sess-history-test"

	// Simulate two prior turns already in k.history.
	k.history = []llm.Message{
		{Role: "user", Content: "first question"},
		{Role: "assistant", Content: "first answer"},
		{Role: "user", Content: "second question"},  // ← current user message
	}

	request := k.buildTurnRequest(context.Background(), turnRequestAssemblyInput{
		Model:       "claude-3-5-sonnet",
		SessionID:   k.sessionID,
		UserText:    "second question",
		UserMessage: llm.Message{Role: "user", Content: "second question"},
	})

	// Find conversation messages (non-system).
	var convMsgs []llm.Message
	for _, m := range request.Messages {
		if m.Role != "system" {
			convMsgs = append(convMsgs, m)
		}
	}

	if len(convMsgs) < 3 {
		t.Fatalf("conversation messages = %d, want at least 3 (user1, assistant1, user2); got %+v", len(convMsgs), convMsgs)
	}
	if convMsgs[len(convMsgs)-3].Content != "first question" {
		t.Errorf("prior user turn missing: got %+v", convMsgs)
	}
	if convMsgs[len(convMsgs)-2].Content != "first answer" {
		t.Errorf("prior assistant turn missing: got %+v", convMsgs)
	}
	if convMsgs[len(convMsgs)-1].Content != "second question" {
		t.Errorf("current user turn wrong: got %+v", convMsgs)
	}
}

// TestKernelTurnRequestAssemblyFreshKernelNoHistory ensures a fresh kernel
// (no prior history) still sends the current user message correctly.
func TestKernelTurnRequestAssemblyFreshKernelNoHistory(t *testing.T) {
	k := New(Config{
		Model:    "claude-3-5-sonnet",
		Endpoint: "http://mock",
	}, llm.NewMockClient(), store.NewNoop(), telemetry.New(), nil)
	k.sessionID = "sess-fresh"

	request := k.buildTurnRequest(context.Background(), turnRequestAssemblyInput{
		Model:       "claude-3-5-sonnet",
		SessionID:   k.sessionID,
		UserText:    "hello",
		UserMessage: llm.Message{Role: "user", Content: "hello"},
	})

	var convMsgs []llm.Message
	for _, m := range request.Messages {
		if m.Role != "system" {
			convMsgs = append(convMsgs, m)
		}
	}
	if len(convMsgs) != 1 || convMsgs[0].Content != "hello" {
		t.Fatalf("fresh kernel: conv messages = %+v, want exactly [user:hello]", convMsgs)
	}
}

func messageRoles(messages []llm.Message) []string {
	roles := make([]string, len(messages))
	for i, msg := range messages {
		roles[i] = msg.Role
	}
	return roles
}

func assertMessageContains(t *testing.T, msg llm.Message, want string) {
	t.Helper()
	if msg.Role != "system" || !strings.Contains(msg.Content, want) {
		t.Fatalf("message = %#v, want system message containing %q", msg, want)
	}
}
