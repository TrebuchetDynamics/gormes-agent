package stream

import (
	"context"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway/delivery/routing"
	"github.com/TrebuchetDynamics/gormes-agent/internal/kernel"
	"github.com/TrebuchetDynamics/gormes-agent/internal/llm"
)

// StreamDeliverySink is the minimal contract for replaying one render frame to
// one resolved destination.
type StreamDeliverySink interface {
	DeliverFrame(ctx context.Context, target routing.Target, frame kernel.RenderFrame) error
}

// DeliveryResult captures the outcome for one attempted fan-out target.
type DeliveryResult struct {
	Target routing.Target
	Err    error
}

// StreamConsumer fans one kernel frame out to one or more delivery targets in
// a deterministic order.
type StreamConsumer struct {
	sink StreamDeliverySink
}

func NewStreamConsumer(sink StreamDeliverySink) *StreamConsumer {
	return &StreamConsumer{sink: sink}
}

func (c *StreamConsumer) FanOut(ctx context.Context, frame kernel.RenderFrame, targets []routing.Target) []DeliveryResult {
	results := make([]DeliveryResult, 0, len(targets))
	if c == nil || c.sink == nil {
		return results
	}
	for _, target := range targets {
		if ctx.Err() != nil {
			break
		}
		err := c.sink.DeliverFrame(ctx, target, cloneRenderFrame(frame))
		results = append(results, DeliveryResult{Target: target, Err: err})
	}
	return results
}

func cloneRenderFrame(frame kernel.RenderFrame) kernel.RenderFrame {
	frame.History = cloneMessages(frame.History)
	frame.SoulEvents = append([]kernel.SoulEntry(nil), frame.SoulEvents...)
	frame.RetryStatus.Schedule = append([]time.Duration(nil), frame.RetryStatus.Schedule...)
	if frame.ContextStatus != nil {
		status := *frame.ContextStatus
		if status.Boundary.Last != nil {
			boundary := *status.Boundary.Last
			status.Boundary.Last = &boundary
		}
		status.Tools.UnknownToolErrors = append([]llm.ContextToolError(nil), status.Tools.UnknownToolErrors...)
		status.Replay.Gaps = append([]llm.ContextReplayGap(nil), status.Replay.Gaps...)
		frame.ContextStatus = &status
	}
	if frame.ApprovalState != nil {
		approval := *frame.ApprovalState
		approval.Choices = append([]kernel.ApprovalChoice(nil), approval.Choices...)
		frame.ApprovalState = &approval
	}
	if frame.ClarifyState != nil {
		clarify := *frame.ClarifyState
		clarify.Choices = append([]string(nil), clarify.Choices...)
		frame.ClarifyState = &clarify
	}
	if frame.SecretState != nil {
		secret := *frame.SecretState
		frame.SecretState = &secret
	}
	return frame
}

func cloneToolCalls(calls []llm.ToolCall) []llm.ToolCall {
	if len(calls) == 0 {
		return nil
	}
	out := make([]llm.ToolCall, len(calls))
	for i, call := range calls {
		out[i] = call
		out[i].Arguments = append([]byte(nil), call.Arguments...)
	}
	return out
}

func cloneMessages(messages []llm.Message) []llm.Message {
	if len(messages) == 0 {
		return nil
	}
	out := make([]llm.Message, len(messages))
	for i, msg := range messages {
		out[i] = msg
		out[i].ContentParts = append([]llm.MessageContentPart(nil), msg.ContentParts...)
		out[i].ToolCalls = cloneToolCalls(msg.ToolCalls)
		if msg.CacheControl != nil {
			cache := *msg.CacheControl
			out[i].CacheControl = &cache
		}
		if msg.Reasoning != nil {
			reasoning := *msg.Reasoning
			out[i].Reasoning = &reasoning
		}
		if msg.ReasoningContent != nil {
			reasoningContent := *msg.ReasoningContent
			out[i].ReasoningContent = &reasoningContent
		}
	}
	return out
}
