package kernel

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/hermes"
	"github.com/TrebuchetDynamics/gormes-agent/internal/store"
	"github.com/TrebuchetDynamics/gormes-agent/internal/telemetry"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tools"
)

func captureGuidanceRequest(t *testing.T, cfg Config, text string) hermes.ChatRequest {
	t.Helper()
	mc := hermes.NewMockClient()
	mc.Script([]hermes.Event{{Kind: hermes.EventDone, FinishReason: "stop"}}, "sess-guidance")
	if cfg.Model == "" {
		cfg.Model = "hermes-agent"
	}
	if cfg.Endpoint == "" {
		cfg.Endpoint = "http://mock"
	}
	cfg.Admission = Admission{MaxBytes: 200_000, MaxLines: 10_000}

	k := New(cfg, mc, store.NewNoop(), telemetry.New(), nil)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	go k.Run(ctx)
	<-k.Render()
	if err := k.Submit(PlatformEvent{Kind: PlatformEventSubmit, Text: text}); err != nil {
		t.Fatalf("Submit: %v", err)
	}
	waitForFrameMatching(t, k.Render(), func(f RenderFrame) bool {
		return f.Phase == PhaseIdle && f.SessionID != ""
	}, 2*time.Second)
	reqs := mc.Requests()
	if len(reqs) == 0 {
		t.Fatal("mock client received zero requests")
	}
	return reqs[0]
}

func TestKernel_LiveTurnGuidanceGatedByCapabilities(t *testing.T) {
	reg := tools.NewRegistry()
	reg.MustRegister(&tools.MockTool{NameStr: "session_search"})
	reg.MustRegister(&tools.MockTool{NameStr: "read_file"})
	recall := &mockRecall{returnContent: "<memory-context>remembered</memory-context>"}
	skills := &stubSkillProvider{block: "<skills>\n## gormes-tdd-slice\nUse TDD.\n</skills>", names: []string{"gormes-tdd-slice"}}

	req := captureGuidanceRequest(t, Config{
		Model:  "gpt-5.5-codex",
		Tools:  reg,
		Recall: recall,
		Skills: skills,
	}, "continue the Gormes slice")
	if len(req.Messages) < 2 || req.Messages[0].Role != "system" {
		t.Fatalf("Messages = %#v, want leading guidance system messages before user", req.Messages)
	}
	joined := joinSystemMessages(req.Messages)
	for _, want := range []string{
		hermes.MemoryGuidance,
		hermes.SessionSearchGuidance,
		hermes.SkillsGuidance,
		hermes.ToolUseEnforcementGuidance,
		hermes.OpenAIModelExecutionGuidance,
		"<memory-context>remembered</memory-context>",
		"gormes-tdd-slice",
	} {
		if !strings.Contains(joined, want) {
			t.Fatalf("provider system guidance missing %q in:\n%s", want, joined)
		}
	}
	assertGuidanceOrder(t, joined, []string{
		hermes.SessionSearchGuidance,
		hermes.ToolUseEnforcementGuidance,
		hermes.OpenAIModelExecutionGuidance,
		hermes.MemoryGuidance,
		"<memory-context>remembered</memory-context>",
		hermes.SkillsGuidance,
		"gormes-tdd-slice",
	})
	last := req.Messages[len(req.Messages)-1]
	if last.Role != "user" || last.Content != "continue the Gormes slice" {
		t.Fatalf("final message = %+v, want raw user submit", last)
	}
}

func TestKernel_LiveTurnGuidanceOmitsUnavailableCapabilities(t *testing.T) {
	req := captureGuidanceRequest(t, Config{Model: "claude-opus-4-7"}, "plain turn")
	joined := joinSystemMessages(req.Messages)
	for _, notWant := range []string{
		hermes.MemoryGuidance,
		hermes.SessionSearchGuidance,
		hermes.SkillsGuidance,
		hermes.ToolUseEnforcementGuidance,
		hermes.OpenAIModelExecutionGuidance,
	} {
		if strings.Contains(joined, notWant) {
			t.Fatalf("provider system guidance unexpectedly contains %q in:\n%s", notWant, joined)
		}
	}
	if len(req.Messages) != 1 || req.Messages[0].Role != "user" {
		t.Fatalf("Messages = %#v, want user-only request when no guidance capabilities are active", req.Messages)
	}
}

func joinSystemMessages(messages []hermes.Message) string {
	var parts []string
	for _, msg := range messages {
		if msg.Role == "system" {
			parts = append(parts, msg.Content)
		}
	}
	return strings.Join(parts, "\n\n")
}

func assertGuidanceOrder(t *testing.T, haystack string, markers []string) {
	t.Helper()
	prev := -1
	for _, marker := range markers {
		idx := strings.Index(haystack, marker)
		if idx < 0 {
			t.Fatalf("missing marker %q in:\n%s", marker, haystack)
		}
		if idx <= prev {
			t.Fatalf("marker %q at %d must follow previous marker index %d in:\n%s", marker, idx, prev, haystack)
		}
		prev = idx
	}
}
