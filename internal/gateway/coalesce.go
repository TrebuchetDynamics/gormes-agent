package gateway

import (
	"context"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway/coalescing"
)

type placeholderEditor = coalescing.PlaceholderEditor

type coalescerMessageSender = coalescing.MessageSender

// CoalescerEvidence carries a redacted operator signal from the coalescer.
// It never contains raw chatIDs, API response bodies, or credentials.
type CoalescerEvidence = coalescing.Evidence

// CoalescerEvidenceSink receives CoalescerEvidence for non-happy-path finalize
// outcomes. The sink must not block or panic; panics are not recovered here —
// callers must ensure the sink is safe to call from any goroutine.
type CoalescerEvidenceSink = coalescing.EvidenceSink

type coalescerOption = coalescing.Option

func coalescerEvidenceSink(sink CoalescerEvidenceSink) coalescerOption {
	return coalescing.EvidenceSinkOption(sink)
}

func coalescerFreshFinalAfter(d time.Duration) coalescerOption {
	return coalescing.FreshFinalAfter(d)
}

func coalescerNow(now func() time.Time) coalescerOption {
	return coalescing.Now(now)
}

func coalescerInitialTextSend() coalescerOption {
	return coalescing.InitialTextSend()
}

func coalescerStreamCursor(cursor string) coalescerOption {
	return coalescing.StreamCursor(cursor)
}

// coalescer batches outbound edits for one turn. The manager owns one
// instance per active turn and tears it down on terminal phases.
type coalescer struct {
	inner *coalescing.Coalescer
}

func newCoalescer(pe placeholderEditor, window time.Duration, chatID string, opts ...coalescerOption) *coalescer {
	return &coalescer{inner: coalescing.New(pe, window, chatID, opts...)}
}

func (c *coalescer) setPending(text string) {
	c.inner.SetPending(text)
}

func (c *coalescer) currentMessageID() string {
	return c.inner.CurrentMessageID()
}

func (c *coalescer) flushImmediate(ctx context.Context, text string) {
	c.inner.FlushImmediate(ctx, text)
}

func (c *coalescer) flushImmediateFinal(ctx context.Context, text string, finalize bool) {
	c.inner.FlushImmediateFinal(ctx, text, finalize)
}

func (c *coalescer) run(ctx context.Context) {
	c.inner.Run(ctx)
}
