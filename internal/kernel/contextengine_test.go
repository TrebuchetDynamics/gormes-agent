package kernel

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/llm"
	"github.com/TrebuchetDynamics/gormes-agent/internal/persistence/store"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/telemetry"
)

func TestKernel_ContextStatusUpdatesFromStreamWithoutHiddenCompression(t *testing.T) {
	engine := &compressSpyContextEngine{
		DisabledContextEngine: llm.NewDisabledContextEngine("compression disabled by config"),
	}
	engine.UpdateModelContext(llm.ContextModelContext{
		Model:            "hermes-agent",
		ContextLength:    1000,
		ThresholdPercent: 0.75,
	})

	mc := llm.NewMockClient()
	mc.Script([]llm.Event{
		{Kind: llm.EventToken, Token: "ok", TokensOut: 1},
		{Kind: llm.EventDone, FinishReason: "stop", TokensIn: 740, TokensOut: 8},
	}, "sess-context")
	k := New(Config{
		Model:         "hermes-agent",
		Endpoint:      "http://mock",
		Admission:     Admission{MaxBytes: 200_000, MaxLines: 10_000},
		ContextEngine: engine,
	}, mc, store.NewNoop(), telemetry.New(), nil)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	go k.Run(ctx)
	initial := <-k.Render()

	if err := k.Submit(PlatformEvent{Kind: PlatformEventSubmit, Text: "hi"}); err != nil {
		t.Fatal(err)
	}
	_, final := drainUntilIdle(t, k.Render(), initial.Seq, 2*time.Second)

	if engine.compressCalls != 0 {
		t.Fatalf("Compress was called %d times; compression must stay an explicit engine boundary", engine.compressCalls)
	}
	if final.ContextStatus == nil {
		t.Fatal("final.ContextStatus is nil, want status snapshot")
	}
	if final.ContextStatus.LastPromptTokens != 740 || final.ContextStatus.LastCompletionTokens != 8 {
		t.Fatalf("context status usage = %#v, want prompt=740 completion=8", final.ContextStatus)
	}
	if final.ContextStatus.Budget.State != "pressure" {
		t.Fatalf("budget state = %q, want pressure", final.ContextStatus.Budget.State)
	}
	if final.ContextStatus.Compression.Enabled {
		t.Fatalf("compression status = %#v, want disabled", final.ContextStatus.Compression)
	}
}

func TestKernel_ContextStatusToolReplaysThroughMockClient(t *testing.T) {
	engine := llm.NewDisabledContextEngine("compression disabled by config")
	engine.UpdateModelContext(llm.ContextModelContext{
		Model:            "hermes-agent",
		ContextLength:    8000,
		ThresholdPercent: 0.75,
	})

	mc := llm.NewMockClient()
	mc.Script([]llm.Event{{
		Kind:         llm.EventDone,
		FinishReason: "tool_calls",
		ToolCalls: []llm.ToolCall{{
			ID:        "call_context_status",
			Name:      llm.ContextStatusToolName,
			Arguments: json.RawMessage(`{}`),
		}},
	}}, "sess-context")
	mc.Script([]llm.Event{{
		Kind:         llm.EventDone,
		FinishReason: "stop",
		TokensIn:     120,
		TokensOut:    4,
	}}, "sess-context")

	k := New(Config{
		Model:         "hermes-agent",
		Endpoint:      "http://mock",
		Admission:     Admission{MaxBytes: 200_000, MaxLines: 10_000},
		ContextEngine: engine,
	}, mc, store.NewNoop(), telemetry.New(), nil)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	go k.Run(ctx)
	initial := <-k.Render()

	if err := k.Submit(PlatformEvent{Kind: PlatformEventSubmit, Text: "status"}); err != nil {
		t.Fatal(err)
	}
	drainUntilIdle(t, k.Render(), initial.Seq, 2*time.Second)

	requests := mc.Requests()
	if len(requests) != 2 {
		t.Fatalf("OpenStream calls = %d, want 2", len(requests))
	}
	if !hasToolDescriptor(requests[0].Tools, llm.ContextStatusToolName) {
		t.Fatalf("first request tools = %#v, want context status tool descriptor", requests[0].Tools)
	}
	var toolMsg *llm.Message
	for i := range requests[1].Messages {
		if requests[1].Messages[i].Role == "tool" && requests[1].Messages[i].ToolCallID == "call_context_status" {
			toolMsg = &requests[1].Messages[i]
			break
		}
	}
	if toolMsg == nil {
		t.Fatalf("second request messages = %#v, want context status tool result", requests[1].Messages)
	}
	var status llm.ContextStatus
	if err := json.Unmarshal([]byte(toolMsg.Content), &status); err != nil {
		t.Fatalf("decode context status tool result: %v\n%s", err, toolMsg.Content)
	}
	if status.ContextLength != 8000 || status.Compression.DisabledReason != "compression disabled by config" {
		t.Fatalf("status tool payload = %#v, want disabled context status", status)
	}
}

func TestKernel_UnknownContextToolReturnsStructuredErrorAndStatus(t *testing.T) {
	engine := llm.NewDisabledContextEngine("compression disabled by config")
	mc := llm.NewMockClient()
	mc.Script([]llm.Event{{
		Kind:         llm.EventDone,
		FinishReason: "tool_calls",
		ToolCalls: []llm.ToolCall{{
			ID:        "call_missing",
			Name:      "missing_context_tool",
			Arguments: json.RawMessage(`{"query":"x"}`),
		}},
	}}, "sess-context")
	mc.Script([]llm.Event{{
		Kind:         llm.EventDone,
		FinishReason: "stop",
		TokensIn:     120,
		TokensOut:    4,
	}}, "sess-context")

	k := New(Config{
		Model:         "hermes-agent",
		Endpoint:      "http://mock",
		Admission:     Admission{MaxBytes: 200_000, MaxLines: 10_000},
		ContextEngine: engine,
	}, mc, store.NewNoop(), telemetry.New(), nil)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	go k.Run(ctx)
	initial := <-k.Render()

	if err := k.Submit(PlatformEvent{Kind: PlatformEventSubmit, Text: "status"}); err != nil {
		t.Fatal(err)
	}
	drainUntilIdle(t, k.Render(), initial.Seq, 2*time.Second)

	requests := mc.Requests()
	if len(requests) != 2 {
		t.Fatalf("OpenStream calls = %d, want 2", len(requests))
	}
	var toolPayload string
	for i := range requests[1].Messages {
		if requests[1].Messages[i].Role == "tool" && requests[1].Messages[i].ToolCallID == "call_missing" {
			toolPayload = requests[1].Messages[i].Content
			break
		}
	}
	if toolPayload == "" {
		t.Fatalf("second request messages = %#v, want missing context tool result", requests[1].Messages)
	}
	if !strings.Contains(toolPayload, `"type":"unknown_context_tool"`) || !strings.Contains(toolPayload, `"tool":"missing_context_tool"`) {
		t.Fatalf("unknown context tool payload = %s, want structured unknown_context_tool error", toolPayload)
	}
	status := engine.Status()
	if len(status.Tools.UnknownToolErrors) != 1 || status.Tools.UnknownToolErrors[0].Tool != "missing_context_tool" {
		t.Fatalf("status unknown tool errors = %#v, want missing_context_tool", status.Tools.UnknownToolErrors)
	}
}

func TestKernel_ResetSession_NotifiesContextEngineSessionEndBeforeReset(t *testing.T) {
	engine := &sessionEndSpyContextEngine{
		DisabledContextEngine: llm.NewDisabledContextEngine("compression disabled by config"),
	}
	mc := llm.NewMockClient()
	mc.Script([]llm.Event{
		{Kind: llm.EventToken, Token: "ok", TokensOut: 1},
		{Kind: llm.EventDone, FinishReason: "stop", TokensIn: 12, TokensOut: 3},
	}, "sess-context-boundary")

	k := New(Config{
		Model:         "hermes-agent",
		Endpoint:      "http://mock",
		Admission:     Admission{MaxBytes: 200_000, MaxLines: 10_000},
		ContextEngine: engine,
	}, mc, store.NewNoop(), telemetry.New(), nil)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	go k.Run(ctx)
	<-k.Render()

	if err := k.Submit(PlatformEvent{Kind: PlatformEventSubmit, Text: "remember this"}); err != nil {
		t.Fatal(err)
	}
	completed := waitForFrameMatching(t, k.Render(), func(f RenderFrame) bool {
		return f.Phase == PhaseIdle && len(f.History) >= 2 && f.SessionID != ""
	}, 2*time.Second)

	if err := k.ResetSession(); err != nil {
		t.Fatalf("ResetSession: %v", err)
	}

	waitForFrameMatching(t, k.Render(), func(f RenderFrame) bool {
		return f.Phase == PhaseIdle && len(f.History) == 0 && f.SessionID == ""
	}, time.Second)

	if engine.sessionEndCalls != 1 {
		t.Fatalf("OnSessionEnd calls = %d, want 1", engine.sessionEndCalls)
	}
	if engine.sessionEndID != completed.SessionID {
		t.Fatalf("OnSessionEnd sessionID = %q, want outgoing %q", engine.sessionEndID, completed.SessionID)
	}
	if got := len(engine.sessionEndMessages); got < 2 {
		t.Fatalf("OnSessionEnd messages len = %d, want completed transcript", got)
	}
	if engine.sessionEndMessages[0].Role != "user" || !strings.Contains(engine.sessionEndMessages[0].Content, "remember this") {
		t.Fatalf("OnSessionEnd first message = %#v, want outgoing user transcript", engine.sessionEndMessages[0])
	}
	if got := strings.Join(engine.order, ","); got != "end,reset" {
		t.Fatalf("context engine hook order = %q, want end,reset", got)
	}
}

func TestKernel_ResetSession_ContextEngineSessionEndFailureStillResets(t *testing.T) {
	engine := &sessionEndSpyContextEngine{
		DisabledContextEngine: llm.NewDisabledContextEngine("compression disabled by config"),
		sessionEndErr:         errors.New("flush failed"),
	}
	mc := llm.NewMockClient()
	mc.Script([]llm.Event{
		{Kind: llm.EventToken, Token: "ok", TokensOut: 1},
		{Kind: llm.EventDone, FinishReason: "stop", TokensIn: 2, TokensOut: 1},
	}, "sess-context-failing-boundary")

	k := New(Config{
		Model:         "hermes-agent",
		Endpoint:      "http://mock",
		Admission:     Admission{MaxBytes: 200_000, MaxLines: 10_000},
		ContextEngine: engine,
	}, mc, store.NewNoop(), telemetry.New(), nil)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	go k.Run(ctx)
	<-k.Render()

	if err := k.Submit(PlatformEvent{Kind: PlatformEventSubmit, Text: "reset despite hook failure"}); err != nil {
		t.Fatal(err)
	}
	waitForFrameMatching(t, k.Render(), func(f RenderFrame) bool {
		return f.Phase == PhaseIdle && len(f.History) >= 2 && f.SessionID != ""
	}, 2*time.Second)

	if err := k.ResetSession(); err != nil {
		t.Fatalf("ResetSession must tolerate OnSessionEnd failure, got %v", err)
	}
	if len(k.history) != 0 || k.sessionID != "" {
		t.Fatalf("after reset = history %d session %q, want cleared", len(k.history), k.sessionID)
	}
	if engine.sessionEndCalls != 1 || engine.resetCalls != 1 {
		t.Fatalf("hook calls end=%d reset=%d, want 1/1", engine.sessionEndCalls, engine.resetCalls)
	}
}

type compressSpyContextEngine struct {
	*llm.DisabledContextEngine
	compressCalls int
}

func (s *compressSpyContextEngine) ShouldCompress(int) bool {
	return true
}

func (s *compressSpyContextEngine) Compress(ctx context.Context, messages []llm.Message, req llm.CompressionRequest) ([]llm.Message, llm.CompressionReport, error) {
	s.compressCalls++
	return s.DisabledContextEngine.Compress(ctx, messages, req)
}

type sessionEndSpyContextEngine struct {
	*llm.DisabledContextEngine
	sessionEndErr      error
	sessionEndCalls    int
	sessionEndID       string
	sessionEndMessages []llm.Message
	resetCalls         int
	order              []string
}

func (s *sessionEndSpyContextEngine) OnSessionEnd(_ context.Context, sessionID string, messages []llm.Message) error {
	s.sessionEndCalls++
	s.sessionEndID = sessionID
	s.sessionEndMessages = append([]llm.Message(nil), messages...)
	s.order = append(s.order, "end")
	return s.sessionEndErr
}

func (s *sessionEndSpyContextEngine) OnSessionReset() {
	s.resetCalls++
	s.order = append(s.order, "reset")
	s.DisabledContextEngine.OnSessionReset()
}

func hasToolDescriptor(tools []llm.ToolDescriptor, name string) bool {
	for _, tool := range tools {
		if tool.Name == name {
			return true
		}
	}
	return false
}
