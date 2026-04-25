package kernel

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/hermes"
	"github.com/TrebuchetDynamics/gormes-agent/internal/store"
	"github.com/TrebuchetDynamics/gormes-agent/internal/telemetry"
)

func TestKernel_ContextStatusUpdatesFromStreamWithoutHiddenCompression(t *testing.T) {
	engine := &compressSpyContextEngine{
		DisabledContextEngine: hermes.NewDisabledContextEngine("compression disabled by config"),
	}
	engine.UpdateModelContext(hermes.ContextModelContext{
		Model:            "hermes-agent",
		ContextLength:    1000,
		ThresholdPercent: 0.75,
	})

	mc := hermes.NewMockClient()
	mc.Script([]hermes.Event{
		{Kind: hermes.EventToken, Token: "ok", TokensOut: 1},
		{Kind: hermes.EventDone, FinishReason: "stop", TokensIn: 740, TokensOut: 8},
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
	engine := hermes.NewDisabledContextEngine("compression disabled by config")
	engine.UpdateModelContext(hermes.ContextModelContext{
		Model:            "hermes-agent",
		ContextLength:    8000,
		ThresholdPercent: 0.75,
	})

	mc := hermes.NewMockClient()
	mc.Script([]hermes.Event{{
		Kind:         hermes.EventDone,
		FinishReason: "tool_calls",
		ToolCalls: []hermes.ToolCall{{
			ID:        "call_context_status",
			Name:      hermes.ContextStatusToolName,
			Arguments: json.RawMessage(`{}`),
		}},
	}}, "sess-context")
	mc.Script([]hermes.Event{{
		Kind:         hermes.EventDone,
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
	if !hasToolDescriptor(requests[0].Tools, hermes.ContextStatusToolName) {
		t.Fatalf("first request tools = %#v, want context status tool descriptor", requests[0].Tools)
	}
	var toolMsg *hermes.Message
	for i := range requests[1].Messages {
		if requests[1].Messages[i].Role == "tool" && requests[1].Messages[i].ToolCallID == "call_context_status" {
			toolMsg = &requests[1].Messages[i]
			break
		}
	}
	if toolMsg == nil {
		t.Fatalf("second request messages = %#v, want context status tool result", requests[1].Messages)
	}
	var status hermes.ContextStatus
	if err := json.Unmarshal([]byte(toolMsg.Content), &status); err != nil {
		t.Fatalf("decode context status tool result: %v\n%s", err, toolMsg.Content)
	}
	if status.ContextLength != 8000 || status.Compression.DisabledReason != "compression disabled by config" {
		t.Fatalf("status tool payload = %#v, want disabled context status", status)
	}
}

func TestKernel_UnknownContextToolReturnsStructuredErrorAndStatus(t *testing.T) {
	engine := hermes.NewDisabledContextEngine("compression disabled by config")
	mc := hermes.NewMockClient()
	mc.Script([]hermes.Event{{
		Kind:         hermes.EventDone,
		FinishReason: "tool_calls",
		ToolCalls: []hermes.ToolCall{{
			ID:        "call_missing",
			Name:      "missing_context_tool",
			Arguments: json.RawMessage(`{"query":"x"}`),
		}},
	}}, "sess-context")
	mc.Script([]hermes.Event{{
		Kind:         hermes.EventDone,
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

type compressSpyContextEngine struct {
	*hermes.DisabledContextEngine
	compressCalls int
}

func (s *compressSpyContextEngine) ShouldCompress(int) bool {
	return true
}

func (s *compressSpyContextEngine) Compress(ctx context.Context, messages []hermes.Message, req hermes.CompressionRequest) ([]hermes.Message, hermes.CompressionReport, error) {
	s.compressCalls++
	return s.DisabledContextEngine.Compress(ctx, messages, req)
}

func hasToolDescriptor(tools []hermes.ToolDescriptor, name string) bool {
	for _, tool := range tools {
		if tool.Name == name {
			return true
		}
	}
	return false
}
