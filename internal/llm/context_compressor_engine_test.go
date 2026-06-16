package llm

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestContextEngineStatusReturnsIsolatedSnapshot(t *testing.T) {
	engine := NewDisabledContextEngine("test")
	_, _ = engine.HandleToolCall(context.Background(), "unknown_tool", nil, ContextToolCallOptions{})

	first := engine.Status()
	if len(first.Tools.UnknownToolErrors) != 1 {
		t.Fatalf("UnknownToolErrors len = %d, want 1", len(first.Tools.UnknownToolErrors))
	}
	first.Tools.UnknownToolErrors[0].Message = "mutated by caller"

	second := engine.Status()
	if got := second.Tools.UnknownToolErrors[0].Message; got == "mutated by caller" {
		t.Fatalf("Status leaked mutable UnknownToolErrors slice: %+v", second.Tools.UnknownToolErrors)
	}
}

func TestContextEngineCompressionErrorStatusIsSanitized(t *testing.T) {
	engine := NewProviderBackedContextEngine(ProviderBackedContextEngineConfig{
		Model:            "test/model",
		ContextLength:    100_000,
		ThresholdPercent: 0.50,
		ProtectFirstN:    0,
		TailTokenBudget:  1,
		MinTailMessages:  1,
		Summarizer: ContextSummarizerFunc(func(context.Context, ContextSummaryRequest) (string, error) {
			return "", errors.New("summary failed\nAuthorization: Bearer sk-context-secret\n**Injected:** yes")
		}),
	})
	messages := []Message{
		{Role: "user", Content: "one"},
		{Role: "assistant", Content: "two"},
		{Role: "user", Content: "three"},
		{Role: "assistant", Content: "four"},
	}

	_, _, err := engine.Compress(context.Background(), messages, CompressionRequest{CurrentTokens: 80_000})
	if err == nil {
		t.Fatal("Compress error = nil, want summarizer failure")
	}
	lastError := engine.Status().Compression.LastError
	for _, forbidden := range []string{"sk-context-secret", "Bearer sk", "\n", "**Injected:**"} {
		if strings.Contains(lastError, forbidden) {
			t.Fatalf("compression status LastError leaked %q in %q", forbidden, lastError)
		}
	}
	if !strings.Contains(lastError, "[redacted]") {
		t.Fatalf("compression status LastError = %q, want redaction marker", lastError)
	}
}

func TestContextSummarizerFuncAllowsNilContext(t *testing.T) {
	got, err := ContextSummarizerFunc(func(ctx context.Context, req ContextSummaryRequest) (string, error) {
		if ctx == nil {
			panic("nil context")
		}
		if req.FocusTopic != "focus" {
			t.Fatalf("FocusTopic = %q, want focus", req.FocusTopic)
		}
		return "summary", nil
	}).SummarizeContext(nil, ContextSummaryRequest{FocusTopic: "focus"})
	if err != nil {
		t.Fatalf("SummarizeContext error = %v, want nil", err)
	}
	if got != "summary" {
		t.Fatalf("SummarizeContext = %q, want summary", got)
	}
}

func TestProviderBackedContextEngine_CompressUsesSummaryLineagePlan(t *testing.T) {
	oldSummary := "OLD-SUMMARY-BODY durable facts from previous context"
	messages := []Message{
		{Role: "system", Content: "system prompt"},
		{Role: "user", Content: ContextPruningSummaryPrefix + "\n" + oldSummary},
		{Role: "assistant", Content: "handoff acknowledged after resume"},
		{Role: "user", Content: "new user turn after resume"},
		{Role: "assistant", Content: "new assistant work after resume"},
		{Role: "user", Content: "more new work after resume"},
		{Role: "assistant", Content: "latest tail response"},
		{Role: "user", Content: "final active request stays in protected tail"},
	}

	var captured ContextSummaryRequest
	engine := NewProviderBackedContextEngine(ProviderBackedContextEngineConfig{
		Model:              "test/model",
		ContextLength:      100_000,
		ThresholdPercent:   0.50,
		ProtectFirstN:      0,
		TailTokenBudget:    1,
		MinTailMessages:    1,
		ToolResultMaxChars: 120,
		Summarizer: ContextSummarizerFunc(func(_ context.Context, req ContextSummaryRequest) (string, error) {
			captured = req
			return "updated summary from resumed turns", nil
		}),
	})

	got, report, err := engine.Compress(context.Background(), messages, CompressionRequest{CurrentTokens: 80_000, FocusTopic: "resume safety"})
	if err != nil {
		t.Fatalf("Compress: %v", err)
	}
	if report.State != "compressed" || report.BeforeMessages != len(messages) || report.AfterMessages >= len(messages) {
		t.Fatalf("report = %+v, want compressed with fewer messages", report)
	}
	if captured.PreviousSummary != oldSummary {
		t.Fatalf("PreviousSummary = %q, want %q", captured.PreviousSummary, oldSummary)
	}
	if captured.FocusTopic != "resume safety" {
		t.Fatalf("FocusTopic = %q", captured.FocusTopic)
	}
	if captured.MaxSummaryTokens <= 0 {
		t.Fatalf("MaxSummaryTokens = %d, want positive budget", captured.MaxSummaryTokens)
	}
	if len(captured.TurnsToSummarize) == 0 {
		t.Fatal("TurnsToSummarize empty, want resumed turns")
	}
	for _, msg := range captured.TurnsToSummarize {
		if strings.Contains(msg.Content, oldSummary) || strings.HasPrefix(msg.Content, ContextPruningSummaryPrefix) {
			t.Fatalf("persisted handoff was serialized as a fresh turn: %+v", msg)
		}
	}

	if len(got) >= len(messages) {
		t.Fatalf("compressed messages len = %d, want fewer than %d", len(got), len(messages))
	}
	joined := joinMessageContents(got)
	if !strings.Contains(joined, ContextPruningSummaryPrefix) || !strings.Contains(joined, "updated summary from resumed turns") {
		t.Fatalf("compressed output missing normalized summary:\n%s", joined)
	}
	if strings.Contains(joined, oldSummary) {
		t.Fatalf("old handoff summary survived as duplicate output:\n%s", joined)
	}
	if !strings.Contains(joined, "final active request stays in protected tail") {
		t.Fatalf("latest tail request missing from compressed output:\n%s", joined)
	}
}

func TestClientContextSummarizerStreamsProviderSummaryPrompt(t *testing.T) {
	client := NewMockClient()
	client.Script([]Event{
		{Kind: EventToken, Token: "structured "},
		{Kind: EventToken, Token: "summary"},
		{Kind: EventDone, FinishReason: "stop"},
	}, "summary-session")
	summarizer := ClientContextSummarizer{Client: client, Model: "summary-model"}

	got, err := summarizer.SummarizeContext(context.Background(), ContextSummaryRequest{
		PreviousSummary: "prior facts",
		TurnsToSummarize: []Message{
			{Role: "user", Content: "new turn"},
		},
		FocusTopic:       "billing",
		MaxSummaryTokens: 1234,
	})
	if err != nil {
		t.Fatalf("SummarizeContext: %v", err)
	}
	if got != "structured summary" {
		t.Fatalf("summary = %q, want streamed tokens", got)
	}
	requests := client.Requests()
	if len(requests) != 1 {
		t.Fatalf("requests len = %d, want 1", len(requests))
	}
	req := requests[0]
	if req.Model != "summary-model" || req.MaxTokens != 1234 || !req.Stream {
		t.Fatalf("request = %+v, want summary model/max tokens/stream", req)
	}
	if len(req.Messages) != 1 || req.Messages[0].Role != "user" {
		t.Fatalf("request messages = %#v, want single user prompt", req.Messages)
	}
	prompt := req.Messages[0].Content
	for _, want := range []string{"PREVIOUS SUMMARY:", "prior facts", "NEW TURNS TO INCORPORATE:", "[USER]: new turn", "FOCUS TOPIC: \"billing\""} {
		if !strings.Contains(prompt, want) {
			t.Fatalf("summary prompt missing %q:\n%s", want, prompt)
		}
	}
	if strings.Contains(prompt, "TURNS TO SUMMARIZE:") {
		t.Fatalf("iterative prompt used first-summary heading:\n%s", prompt)
	}
}

func TestClientContextSummarizerEmptyStreamReturnsError(t *testing.T) {
	client := NewMockClient()
	client.Script([]Event{{Kind: EventDone, FinishReason: "stop"}}, "summary-session")
	summarizer := ClientContextSummarizer{Client: client, Model: "summary-model"}

	_, err := summarizer.SummarizeContext(context.Background(), ContextSummaryRequest{TurnsToSummarize: []Message{{Role: "user", Content: "x"}}})
	if !errors.Is(err, ErrContextSummaryUnavailable) {
		t.Fatalf("err = %v, want ErrContextSummaryUnavailable", err)
	}
}

func joinMessageContents(messages []Message) string {
	var b strings.Builder
	for _, msg := range messages {
		if b.Len() > 0 {
			b.WriteString("\n")
		}
		b.WriteString(msg.Content)
	}
	return b.String()
}
