// Package fakes exposes test-only gateway.Channel fixtures shared across
// gateway and channel adapter test packages. Real channel adapters live under
// internal/channels/<platform>; the fakes here only exist so adapter-neutral
// tests can prove the gateway turn adapter does not hard-code Telegram (or any
// other) channel behavior.
package fakes

import (
	"context"
	"strconv"
	"sync"

	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway"
)

// Sent records one outbound message a fakes.Channel observed.
type Sent struct {
	ChatID string
	Text   string
	MsgID  string
}

// Channel is a minimal gateway.Channel implementation for tests. It does not
// pull in any platform SDK and does not change channel-specific identity,
// require-mention, delivery, or thread rules — it simply records the
// channel-neutral surface so a TurnAdapter test can prove neutrality.
type Channel struct {
	name string

	mu        sync.Mutex
	sent      []Sent
	nextMsgID int
}

// NewChannel returns a Channel whose Name() reports the given platform name.
func NewChannel(name string) *Channel {
	return &Channel{name: name, nextMsgID: 9000}
}

// Name implements gateway.Channel.
func (c *Channel) Name() string { return c.name }

// Run implements gateway.Channel. The fake never produces inbound traffic on
// its own; tests drive InboundEvents directly.
func (c *Channel) Run(ctx context.Context, _ chan<- gateway.InboundEvent) error {
	<-ctx.Done()
	return nil
}

// Send implements gateway.Channel by recording the outbound text.
func (c *Channel) Send(_ context.Context, chatID, text string) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	id := strconv.Itoa(c.nextMsgID)
	c.nextMsgID++
	c.sent = append(c.sent, Sent{ChatID: chatID, Text: text, MsgID: id})
	return id, nil
}

// SentSnapshot returns a copy of recorded outbound messages.
func (c *Channel) SentSnapshot() []Sent {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := make([]Sent, len(c.sent))
	copy(out, c.sent)
	return out
}
