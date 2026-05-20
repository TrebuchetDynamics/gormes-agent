package e2e

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/audit"
	"github.com/TrebuchetDynamics/gormes-agent/internal/hermes"
	"github.com/TrebuchetDynamics/gormes-agent/internal/kernel"
	"github.com/TrebuchetDynamics/gormes-agent/internal/memory"
	"github.com/TrebuchetDynamics/gormes-agent/internal/telemetry"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tools"
)

func TestComplexE2E_MultiStepToolChainAuditAndPersistence(t *testing.T) {
	dir := t.TempDir()
	mem, err := memory.OpenSqlite(filepath.Join(dir, "memory.db"), 32, nil)
	if err != nil {
		t.Fatalf("OpenSqlite: %v", err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := mem.Close(ctx); err != nil {
			t.Fatalf("close memory store: %v", err)
		}
	})

	reg := tools.NewRegistry()
	reg.MustRegister(complexE2ETool{name: "inspect_fixture", response: `{"observation":"fixture ready","risk":"low"}`})
	reg.MustRegister(complexE2ETool{name: "summarize_fixture", response: `{"summary":"fixture ready with low risk"}`})
	reg.MustRegister(complexE2ETool{name: "persist_decision", response: `{"stored":true,"decision":"ship"}`})

	provider := hermes.NewMockClient()
	provider.Script([]hermes.Event{{
		Kind:         hermes.EventDone,
		FinishReason: "tool_calls",
		ToolCalls: []hermes.ToolCall{
			{ID: "call_inspect", Name: "inspect_fixture", Arguments: json.RawMessage(`{"target":"complex-e2e"}`)},
			{ID: "call_summarize", Name: "summarize_fixture", Arguments: json.RawMessage(`{"style":"evidence-first"}`)},
		},
		TokensIn:  50,
		TokensOut: 3,
	}}, "complex-provider-session")
	provider.Script([]hermes.Event{{
		Kind:         hermes.EventDone,
		FinishReason: "tool_calls",
		ToolCalls: []hermes.ToolCall{
			{ID: "call_persist", Name: "persist_decision", Arguments: json.RawMessage(`{"decision":"ship","because":"tools agreed"}`)},
		},
		TokensIn:  80,
		TokensOut: 2,
	}}, "complex-provider-session")
	finalText := "Complex E2E completed: fixture ready, low risk, decision stored."
	provider.Script([]hermes.Event{
		{Kind: hermes.EventToken, Token: finalText, TokensOut: 10},
		{Kind: hermes.EventDone, FinishReason: "stop", TokensIn: 110, TokensOut: 10},
	}, "complex-provider-session")

	auditPath := filepath.Join(dir, "tool-audit.jsonl")
	k := kernel.New(kernel.Config{
		Model:             "complex-e2e-model",
		Endpoint:          "mock://complex-e2e",
		Admission:         kernel.Admission{MaxBytes: 200_000, MaxLines: 10_000},
		Tools:             reg,
		MaxToolIterations: 4,
		MaxToolDuration:   2 * time.Second,
		InitialSessionID:  "sess-complex-e2e",
		ChatKey:           "telegram:complex-e2e",
		ToolAudit:         audit.NewJSONLWriter(auditPath),
	}, provider, mem, telemetry.New(), nil)

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- k.Run(ctx) }()
	t.Cleanup(func() {
		cancel()
		select {
		case err := <-done:
			if err != nil {
				t.Fatalf("kernel run: %v", err)
			}
		case <-time.After(2 * time.Second):
			t.Fatal("kernel did not stop after cleanup cancel")
		}
	})
	waitForFrame(t, k.Render(), func(f kernel.RenderFrame) bool { return f.Phase == kernel.PhaseIdle }, time.Second)

	userInput := "Run the complex fixture workflow and report the decision."
	if err := k.Submit(kernel.PlatformEvent{Kind: kernel.PlatformEventSubmit, Text: userInput}); err != nil {
		t.Fatalf("submit turn: %v", err)
	}
	final := waitForFrame(t, k.Render(), func(f kernel.RenderFrame) bool {
		return f.Phase == kernel.PhaseIdle && len(f.History) >= 2 && f.History[len(f.History)-1].Role == "assistant"
	}, 5*time.Second)
	if got := final.History[len(final.History)-1].Content; got != finalText {
		t.Fatalf("final assistant content = %q, want %q", got, finalText)
	}

	requests := provider.Requests()
	if len(requests) != 3 {
		t.Fatalf("provider request count = %d, want 3", len(requests))
	}
	assertComplexInitialRequest(t, requests[0], userInput)
	assertComplexContinuation(t, requests[1], []string{"call_inspect", "call_summarize"})
	assertComplexContinuation(t, requests[2], []string{"call_inspect", "call_summarize", "call_persist"})
	assertComplexAudits(t, auditPath, []string{"inspect_fixture", "summarize_fixture", "persist_decision"})
	assertComplexPersistedTurn(t, mem.DB(), "telegram:complex-e2e", "user", userInput)
	assertComplexPersistedTurn(t, mem.DB(), "telegram:complex-e2e", "assistant", finalText)
}

func TestComplexE2E_MixedToolBatchPreservesEveryOutcomeAndRecovers(t *testing.T) {
	dir := t.TempDir()
	mem, err := memory.OpenSqlite(filepath.Join(dir, "memory.db"), 32, nil)
	if err != nil {
		t.Fatalf("OpenSqlite: %v", err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := mem.Close(ctx); err != nil {
			t.Fatalf("close memory store: %v", err)
		}
	})

	reg := tools.NewRegistry()
	reg.MustRegister(complexE2ETool{name: "stable_fixture", response: `{"status":"ok","value":"alpha"}`})
	reg.MustRegister(complexE2ETool{name: "failing_fixture", err: errors.New("fixture failed after partial work")})

	provider := hermes.NewMockClient()
	provider.Script([]hermes.Event{{
		Kind:         hermes.EventDone,
		FinishReason: "tool_calls",
		ToolCalls: []hermes.ToolCall{
			{ID: "call_stable", Name: "stable_fixture", Arguments: json.RawMessage(`{"target":"mixed-success"}`)},
			{ID: "call_failing", Name: "failing_fixture", Arguments: json.RawMessage(`{"target":"mixed-failure"}`)},
			{ID: "call_missing", Name: "missing_fixture", Arguments: json.RawMessage(`{"target":"unknown"}`)},
		},
		TokensIn:  60,
		TokensOut: 4,
	}}, "mixed-tool-session")
	finalText := "Mixed tool batch recovered: one success, two failures, no raw tool noise leaked."
	provider.Script([]hermes.Event{
		{Kind: hermes.EventToken, Token: finalText, TokensOut: 12},
		{Kind: hermes.EventDone, FinishReason: "stop", TokensIn: 120, TokensOut: 12},
	}, "mixed-tool-session")

	auditPath := filepath.Join(dir, "tool-audit.jsonl")
	k := kernel.New(kernel.Config{
		Model:             "complex-e2e-model",
		Endpoint:          "mock://complex-e2e-mixed",
		Admission:         kernel.Admission{MaxBytes: 200_000, MaxLines: 10_000},
		Tools:             reg,
		MaxToolIterations: 2,
		MaxToolDuration:   2 * time.Second,
		InitialSessionID:  "sess-complex-mixed",
		ChatKey:           "telegram:complex-mixed",
		ToolAudit:         audit.NewJSONLWriter(auditPath),
	}, provider, mem, telemetry.New(), nil)

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- k.Run(ctx) }()
	t.Cleanup(func() {
		cancel()
		select {
		case err := <-done:
			if err != nil {
				t.Fatalf("kernel run: %v", err)
			}
		case <-time.After(2 * time.Second):
			t.Fatal("kernel did not stop after cleanup cancel")
		}
	})
	waitForFrame(t, k.Render(), func(f kernel.RenderFrame) bool { return f.Phase == kernel.PhaseIdle }, time.Second)

	userInput := "Run the mixed tool batch and summarize every outcome safely."
	if err := k.Submit(kernel.PlatformEvent{Kind: kernel.PlatformEventSubmit, Text: userInput}); err != nil {
		t.Fatalf("submit turn: %v", err)
	}
	final := waitForFrame(t, k.Render(), func(f kernel.RenderFrame) bool {
		return f.Phase == kernel.PhaseIdle && len(f.History) >= 2 && f.History[len(f.History)-1].Role == "assistant"
	}, 5*time.Second)
	if got := final.History[len(final.History)-1].Content; got != finalText {
		t.Fatalf("final assistant content = %q, want %q", got, finalText)
	}
	if strings.Contains(finalText, "<tool_call") || strings.Contains(finalText, "unknown tool:") {
		t.Fatalf("final assistant leaked raw tool details: %q", finalText)
	}

	requests := provider.Requests()
	if len(requests) != 2 {
		t.Fatalf("provider request count = %d, want 2", len(requests))
	}
	assertComplexContinuation(t, requests[1], []string{"call_stable", "call_failing", "call_missing"})
	assertComplexToolReplyContains(t, requests[1], "call_stable", `"status":"ok"`)
	assertComplexToolReplyContains(t, requests[1], "call_failing", "fixture failed after partial work")
	assertComplexToolReplyContains(t, requests[1], "call_missing", "unknown tool")
	assertComplexAuditStatus(t, auditPath, "stable_fixture", "completed")
	assertComplexAuditStatus(t, auditPath, "failing_fixture", "failed")
	assertComplexAuditStatus(t, auditPath, "missing_fixture", "failed")
	assertComplexPersistedTurn(t, mem.DB(), "telegram:complex-mixed", "user", userInput)
	assertComplexPersistedTurn(t, mem.DB(), "telegram:complex-mixed", "assistant", finalText)
}

func TestComplexE2E_ToolFailureIsAuditedAndConversationRecovers(t *testing.T) {
	dir := t.TempDir()
	mem, err := memory.OpenSqlite(filepath.Join(dir, "memory.db"), 32, nil)
	if err != nil {
		t.Fatalf("OpenSqlite: %v", err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := mem.Close(ctx); err != nil {
			t.Fatalf("close memory store: %v", err)
		}
	})

	reg := tools.NewRegistry()
	reg.MustRegister(complexE2ETool{name: "unstable_fixture", err: errors.New("fixture unavailable")})

	provider := hermes.NewMockClient()
	provider.Script([]hermes.Event{{
		Kind:         hermes.EventDone,
		FinishReason: "tool_calls",
		ToolCalls: []hermes.ToolCall{{
			ID:        "call_unstable",
			Name:      "unstable_fixture",
			Arguments: json.RawMessage(`{"target":"negative-path"}`),
		}},
		TokensIn:  40,
		TokensOut: 1,
	}}, "complex-failure-session")
	finalText := "Recovered from the fixture failure and reported it safely."
	provider.Script([]hermes.Event{
		{Kind: hermes.EventToken, Token: finalText, TokensOut: 9},
		{Kind: hermes.EventDone, FinishReason: "stop", TokensIn: 75, TokensOut: 9},
	}, "complex-failure-session")

	auditPath := filepath.Join(dir, "tool-audit.jsonl")
	k := kernel.New(kernel.Config{
		Model:             "complex-e2e-model",
		Endpoint:          "mock://complex-e2e-failure",
		Admission:         kernel.Admission{MaxBytes: 200_000, MaxLines: 10_000},
		Tools:             reg,
		MaxToolIterations: 2,
		MaxToolDuration:   2 * time.Second,
		InitialSessionID:  "sess-complex-failure",
		ChatKey:           "telegram:complex-failure",
		ToolAudit:         audit.NewJSONLWriter(auditPath),
	}, provider, mem, telemetry.New(), nil)

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- k.Run(ctx) }()
	t.Cleanup(func() {
		cancel()
		select {
		case err := <-done:
			if err != nil {
				t.Fatalf("kernel run: %v", err)
			}
		case <-time.After(2 * time.Second):
			t.Fatal("kernel did not stop after cleanup cancel")
		}
	})
	waitForFrame(t, k.Render(), func(f kernel.RenderFrame) bool { return f.Phase == kernel.PhaseIdle }, time.Second)

	userInput := "Run the unstable fixture and recover cleanly."
	if err := k.Submit(kernel.PlatformEvent{Kind: kernel.PlatformEventSubmit, Text: userInput}); err != nil {
		t.Fatalf("submit turn: %v", err)
	}
	final := waitForFrame(t, k.Render(), func(f kernel.RenderFrame) bool {
		return f.Phase == kernel.PhaseIdle && len(f.History) >= 2 && f.History[len(f.History)-1].Role == "assistant"
	}, 5*time.Second)
	if got := final.History[len(final.History)-1].Content; got != finalText {
		t.Fatalf("final assistant content = %q, want %q", got, finalText)
	}

	requests := provider.Requests()
	if len(requests) != 2 {
		t.Fatalf("provider request count = %d, want 2", len(requests))
	}
	assertComplexContinuation(t, requests[1], []string{"call_unstable"})
	assertComplexAuditStatus(t, auditPath, "unstable_fixture", "failed")
	assertComplexPersistedTurn(t, mem.DB(), "telegram:complex-failure", "user", userInput)
	assertComplexPersistedTurn(t, mem.DB(), "telegram:complex-failure", "assistant", finalText)
}

type complexE2ETool struct {
	name     string
	response string
	err      error
}

func (t complexE2ETool) Name() string        { return t.name }
func (t complexE2ETool) Description() string { return "complex E2E fixture tool " + t.name }
func (t complexE2ETool) Schema() json.RawMessage {
	return json.RawMessage(`{"type":"object","additionalProperties":true}`)
}
func (t complexE2ETool) Timeout() time.Duration { return time.Second }
func (t complexE2ETool) Execute(context.Context, json.RawMessage) (json.RawMessage, error) {
	if t.err != nil {
		return nil, t.err
	}
	return json.RawMessage(t.response), nil
}

func assertComplexInitialRequest(t *testing.T, req hermes.ChatRequest, userInput string) {
	t.Helper()
	for _, name := range []string{"inspect_fixture", "summarize_fixture", "persist_decision"} {
		if !hasToolDescriptor(req.Tools, name) {
			t.Fatalf("initial tools missing %q: %+v", name, req.Tools)
		}
	}
	if len(req.Messages) == 0 || req.Messages[len(req.Messages)-1].Role != "user" || req.Messages[len(req.Messages)-1].Content != userInput {
		t.Fatalf("initial request last message = %+v, want user input %q", req.Messages, userInput)
	}
}

func assertComplexContinuation(t *testing.T, req hermes.ChatRequest, wantToolReplies []string) {
	t.Helper()
	seen := map[string]string{}
	for _, msg := range req.Messages {
		if msg.Role == "tool" {
			seen[msg.ToolCallID] = msg.Content
		}
	}
	for _, id := range wantToolReplies {
		content, ok := seen[id]
		if !ok {
			t.Fatalf("continuation missing tool reply %q in messages %+v", id, req.Messages)
		}
		if !strings.Contains(content, "{") {
			t.Fatalf("tool reply %q content = %q, want JSON object", id, content)
		}
	}
}

func assertComplexToolReplyContains(t *testing.T, req hermes.ChatRequest, callID, want string) {
	t.Helper()
	for _, msg := range req.Messages {
		if msg.Role == "tool" && msg.ToolCallID == callID {
			if !strings.Contains(msg.Content, want) {
				t.Fatalf("tool reply %q content = %q, want substring %q", callID, msg.Content, want)
			}
			return
		}
	}
	t.Fatalf("tool reply %q not found in messages %+v", callID, req.Messages)
}

func assertComplexAudits(t *testing.T, path string, tools []string) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for {
		records := readAuditRecords(t, path)
		seen := map[string]bool{}
		for _, rec := range records {
			if rec.Status == "completed" {
				seen[rec.Tool] = true
			}
		}
		missing := missingComplexTools(tools, seen)
		if len(missing) == 0 {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("missing completed audit records %v in %+v", missing, records)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func assertComplexAuditStatus(t *testing.T, path, toolName, status string) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for {
		records := readAuditRecords(t, path)
		for _, rec := range records {
			if rec.Tool == toolName && rec.Status == status {
				return
			}
		}
		if time.Now().After(deadline) {
			t.Fatalf("audit record for %s/%s not found in %+v", toolName, status, records)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func missingComplexTools(want []string, seen map[string]bool) []string {
	var missing []string
	for _, name := range want {
		if !seen[name] {
			missing = append(missing, name)
		}
	}
	return missing
}

func assertComplexPersistedTurn(t *testing.T, db *sql.DB, chatID, role, content string) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for {
		var count int
		err := db.QueryRowContext(context.Background(),
			`SELECT COUNT(*) FROM turns WHERE chat_id = ? AND role = ? AND content = ?`,
			chatID, role, content,
		).Scan(&count)
		if err == nil && count > 0 {
			return
		}
		if time.Now().After(deadline) {
			if err != nil {
				t.Fatalf("query persisted %s turn: %v", role, err)
			}
			t.Fatalf("persisted %s turn %q for chat %q not found", role, content, chatID)
		}
		time.Sleep(10 * time.Millisecond)
	}
}
