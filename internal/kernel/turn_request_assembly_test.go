package kernel

import (
	"context"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/llm"
	"github.com/TrebuchetDynamics/gormes-agent/internal/persistence/store"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/telemetry"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tools"
)

func TestKernelTurnRequestAssemblyPreservesGuidancePrefillToolAndUsageOrder(t *testing.T) {
	registry := tools.NewRegistry()
	registry.MustRegister(&tools.MockTool{NameStr: "session_search"})
	registry.MustRegister(&tools.MockTool{NameStr: "web_search"})

	recall := &mockRecall{returnContent: "<memory-context>remembered</memory-context>"}
	skills := &stubSkillProvider{
		block: "<skills>\n## gormes-tdd-slice\nUse TDD.\n</skills>",
		names: []string{"gormes-tdd-slice"},
	}
	usage := &stubSkillUsageRecorder{}

	k := New(Config{
		Model:          "gpt-5.5-codex",
		Endpoint:       "http://mock",
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
	wantRoles := []string{"system", "system", "system", "system", "system", "system", "system", "system", "user", "assistant", "user"}
	if gotRoles := messageRoles(messages); !reflect.DeepEqual(gotRoles, wantRoles) {
		t.Fatalf("message roles = %#v, want %#v\nmessages=%#v", gotRoles, wantRoles, messages)
	}
	assertMessageContains(t, messages[0], "<session-context>active</session-context>")
	assertMessageContains(t, messages[1], llm.SessionSearchGuidance)
	assertMessageContains(t, messages[2], llm.ToolUseEnforcementGuidance)
	assertMessageContains(t, messages[3], llm.OpenAIModelExecutionGuidance)
	assertMessageContains(t, messages[4], llm.ResearchQualityGuidance)
	assertMessageContains(t, messages[5], llm.MemoryGuidance)
	assertMessageContains(t, messages[5], "<memory-context>remembered</memory-context>")
	assertMessageContains(t, messages[6], llm.SkillsGuidance)
	assertMessageContains(t, messages[7], "gormes-tdd-slice")
	if messages[8].Content != "example request" || messages[9].Content != "example answer" || messages[10].Content != "ship the slice" {
		t.Fatalf("tail messages = %#v, want prefill before current user", messages[8:])
	}
	if recall.lastInput.UserMessage != "ship the slice" || recall.lastInput.ChatKey != "telegram:42" || recall.lastInput.SessionID != "sess-request-assembly" {
		t.Fatalf("recall input = %#v, want user/chat/session evidence", recall.lastInput)
	}
	if usage.calls != 1 || !reflect.DeepEqual(usage.got, [][]string{{"gormes-tdd-slice"}}) {
		t.Fatalf("skill usage = calls %d got %#v, want one recorded selected skill", usage.calls, usage.got)
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
