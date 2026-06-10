package stream

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway/delivery/routing"
	"github.com/TrebuchetDynamics/gormes-agent/internal/kernel"
	"github.com/TrebuchetDynamics/gormes-agent/internal/llm"
)

type recordingStreamSink struct {
	calls             []streamDeliveryCall
	errs              map[string]error
	mutateFirstFrame  bool
	mutateFirstTarget bool
}

type streamDeliveryCall struct {
	target routing.Target
	frame  kernel.RenderFrame
}

func (s *recordingStreamSink) DeliverFrame(_ context.Context, target routing.Target, frame kernel.RenderFrame) error {
	if len(s.calls) == 0 {
		if s.mutateFirstFrame && len(frame.History) > 0 {
			frame.History[0].Content = "mutated-by-first-target"
			if len(frame.History[0].ToolCalls) > 0 && len(frame.History[0].ToolCalls[0].Arguments) > 0 {
				frame.History[0].ToolCalls[0].Arguments[0] = '['
			}
			if frame.ContextStatus != nil && frame.ContextStatus.Boundary.Last != nil {
				frame.ContextStatus.Boundary.Last.OldSessionID = "mutated-boundary"
			}
			if frame.ContextStatus != nil && len(frame.ContextStatus.Tools.UnknownToolErrors) > 0 {
				frame.ContextStatus.Tools.UnknownToolErrors[0].Message = "mutated-tool-error"
			}
			if frame.ContextStatus != nil && len(frame.ContextStatus.Replay.Gaps) > 0 {
				frame.ContextStatus.Replay.Gaps[0].Message = "mutated-replay-gap"
			}
			if len(frame.RetryStatus.Schedule) > 0 {
				frame.RetryStatus.Schedule[0] = 99
			}
		}
		if s.mutateFirstTarget {
			target.ChatID = "mutated-target"
		}
	}
	s.calls = append(s.calls, streamDeliveryCall{target: target, frame: frame})
	if s.errs != nil {
		return s.errs[target.String()]
	}
	return nil
}

func TestStreamConsumer_FanOutAllowsNilContext(t *testing.T) {
	sink := &recordingStreamSink{}
	consumer := NewStreamConsumer(sink)
	targets := []routing.Target{{Platform: "telegram", ChatID: "42", IsExplicit: true}}

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("FanOut panicked with nil context: %v", r)
		}
	}()
	results := consumer.FanOut(nil, kernel.RenderFrame{SessionID: "sess-nilctx"}, targets)

	if len(results) != 1 || len(sink.calls) != 1 {
		t.Fatalf("fanout with nil context results=%d calls=%d, want 1/1", len(results), len(sink.calls))
	}
}

func TestStreamConsumer_FanOutsToMultipleTargets(t *testing.T) {
	sink := &recordingStreamSink{}
	consumer := NewStreamConsumer(sink)
	frame := kernel.RenderFrame{Phase: kernel.PhaseStreaming, DraftText: "partial", SessionID: "sess-1"}
	targets := []routing.Target{
		{Platform: "telegram", ChatID: "42", IsExplicit: true},
		{Platform: "discord", ChatID: "99", IsExplicit: true},
	}

	results := consumer.FanOut(context.Background(), frame, targets)

	if len(results) != 2 {
		t.Fatalf("results len = %d, want 2", len(results))
	}
	if len(sink.calls) != 2 {
		t.Fatalf("sink calls = %d, want 2", len(sink.calls))
	}
	for i, want := range targets {
		if sink.calls[i].target != want {
			t.Fatalf("call %d target = %+v, want %+v", i, sink.calls[i].target, want)
		}
		if sink.calls[i].frame.SessionID != "sess-1" {
			t.Fatalf("call %d SessionID = %q, want %q", i, sink.calls[i].frame.SessionID, "sess-1")
		}
		if results[i].Target != want || results[i].Err != nil {
			t.Fatalf("result %d = %+v, want target %+v with nil err", i, results[i], want)
		}
	}
}

func TestStreamConsumer_FanOutIsolatesMutableFramePerTarget(t *testing.T) {
	sink := &recordingStreamSink{mutateFirstFrame: true, mutateFirstTarget: true}
	consumer := NewStreamConsumer(sink)
	frame := kernel.RenderFrame{
		Phase:     kernel.PhaseStreaming,
		SessionID: "sess-aliased",
		History: []llm.Message{{
			Role:    "assistant",
			Content: "original",
			ToolCalls: []llm.ToolCall{{
				ID:        "call-1",
				Name:      "example",
				Arguments: []byte(`{"path":"/tmp/example"}`),
			}},
		}},
		ContextStatus: &llm.ContextStatus{
			Boundary: llm.ContextBoundaryStatus{Last: &llm.CompressionBoundary{OldSessionID: "old-session"}},
			Tools:    llm.ContextToolStatus{UnknownToolErrors: []llm.ContextToolError{{Message: "unknown tool"}}},
			Replay:   llm.ContextReplayStatus{Gaps: []llm.ContextReplayGap{{Message: "missing replay"}}},
		},
		RetryStatus: kernel.RetryStatus{Schedule: []time.Duration{time.Second}},
	}
	targets := []routing.Target{
		{Platform: "telegram", ChatID: "42", IsExplicit: true},
		{Platform: "discord", ChatID: "99", IsExplicit: true},
	}

	results := consumer.FanOut(context.Background(), frame, targets)

	if len(sink.calls) != 2 || len(results) != 2 {
		t.Fatalf("fanout lengths calls=%d results=%d, want 2/2", len(sink.calls), len(results))
	}
	if sink.calls[1].frame.History[0].Content != "original" {
		t.Fatalf("second delivery frame history = %q, want isolated original", sink.calls[1].frame.History[0].Content)
	}
	if frame.History[0].Content != "original" {
		t.Fatalf("caller frame history = %q, want original", frame.History[0].Content)
	}
	if got := string(sink.calls[1].frame.History[0].ToolCalls[0].Arguments); got != `{"path":"/tmp/example"}` {
		t.Fatalf("second delivery tool arguments = %q, want isolated original", got)
	}
	if got := string(frame.History[0].ToolCalls[0].Arguments); got != `{"path":"/tmp/example"}` {
		t.Fatalf("caller frame tool arguments = %q, want original", got)
	}
	if got := sink.calls[1].frame.ContextStatus.Boundary.Last.OldSessionID; got != "old-session" {
		t.Fatalf("second delivery context boundary = %q, want isolated original", got)
	}
	if got := frame.ContextStatus.Boundary.Last.OldSessionID; got != "old-session" {
		t.Fatalf("caller frame context boundary = %q, want original", got)
	}
	if got := sink.calls[1].frame.ContextStatus.Tools.UnknownToolErrors[0].Message; got != "unknown tool" {
		t.Fatalf("second delivery context tool error = %q, want isolated original", got)
	}
	if got := frame.ContextStatus.Tools.UnknownToolErrors[0].Message; got != "unknown tool" {
		t.Fatalf("caller frame context tool error = %q, want original", got)
	}
	if got := sink.calls[1].frame.ContextStatus.Replay.Gaps[0].Message; got != "missing replay" {
		t.Fatalf("second delivery context replay gap = %q, want isolated original", got)
	}
	if got := frame.ContextStatus.Replay.Gaps[0].Message; got != "missing replay" {
		t.Fatalf("caller frame context replay gap = %q, want original", got)
	}
	if got := sink.calls[1].frame.RetryStatus.Schedule[0]; got != time.Second {
		t.Fatalf("second delivery retry schedule = %v, want isolated original", got)
	}
	if got := frame.RetryStatus.Schedule[0]; got != time.Second {
		t.Fatalf("caller frame retry schedule = %v, want original", got)
	}
	if results[0].Target.ChatID != "42" || targets[0].ChatID != "42" {
		t.Fatalf("target mutation leaked results=%+v targets=%+v", results[0].Target, targets[0])
	}
}

func TestStreamConsumer_FanOutStopsAfterContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	sink := &cancelingStreamSink{cancel: cancel}
	consumer := NewStreamConsumer(sink)
	targets := []routing.Target{
		{Platform: "telegram", ChatID: "42", IsExplicit: true},
		{Platform: "discord", ChatID: "99", IsExplicit: true},
	}

	results := consumer.FanOut(ctx, kernel.RenderFrame{SessionID: "sess-cancel"}, targets)

	if len(sink.calls) != 1 {
		t.Fatalf("sink calls = %d, want stop after cancellation", len(sink.calls))
	}
	if len(results) != 1 || results[0].Target != targets[0] {
		t.Fatalf("results = %+v, want only first target", results)
	}
}

type cancelingStreamSink struct {
	calls  []routing.Target
	cancel context.CancelFunc
}

func (s *cancelingStreamSink) DeliverFrame(_ context.Context, target routing.Target, _ kernel.RenderFrame) error {
	s.calls = append(s.calls, target)
	s.cancel()
	return context.Canceled
}

func TestStreamConsumer_FanOutStopsWhenSinkReturnsContextCanceled(t *testing.T) {
	sink := &recordingStreamSink{errs: map[string]error{"telegram:42": context.Canceled}}
	consumer := NewStreamConsumer(sink)
	targets := []routing.Target{
		{Platform: "telegram", ChatID: "42", IsExplicit: true},
		{Platform: "discord", ChatID: "99", IsExplicit: true},
	}

	results := consumer.FanOut(context.Background(), kernel.RenderFrame{SessionID: "sess-cancel-err"}, targets)

	if len(sink.calls) != 1 {
		t.Fatalf("sink calls = %d, want stop after context.Canceled delivery error", len(sink.calls))
	}
	if len(results) != 1 || !errors.Is(results[0].Err, context.Canceled) {
		t.Fatalf("results = %+v, want one context.Canceled result", results)
	}
}

func TestStreamConsumer_FanOutContinuesAfterError(t *testing.T) {
	sink := &recordingStreamSink{
		errs: map[string]error{"telegram:42": errors.New("send failed")},
	}
	consumer := NewStreamConsumer(sink)
	frame := kernel.RenderFrame{Phase: kernel.PhaseIdle, SessionID: "sess-2"}
	targets := []routing.Target{
		{Platform: "telegram", ChatID: "42", IsExplicit: true},
		{Platform: "discord", ChatID: "99", IsExplicit: true},
	}

	results := consumer.FanOut(context.Background(), frame, targets)

	if len(sink.calls) != 2 {
		t.Fatalf("sink calls = %d, want 2", len(sink.calls))
	}
	if results[0].Err == nil {
		t.Fatal("first result error = nil, want non-nil")
	}
	if results[1].Err != nil {
		t.Fatalf("second result error = %v, want nil", results[1].Err)
	}
}
