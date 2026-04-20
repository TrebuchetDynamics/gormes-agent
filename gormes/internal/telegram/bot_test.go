package telegram

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/XelHaku/golang-hermes-agent/gormes/internal/hermes"
	"github.com/XelHaku/golang-hermes-agent/gormes/internal/kernel"
	"github.com/XelHaku/golang-hermes-agent/gormes/internal/store"
	"github.com/XelHaku/golang-hermes-agent/gormes/internal/telemetry"
	"github.com/XelHaku/golang-hermes-agent/gormes/internal/tools"
)

// newTestKernel builds a Kernel with MockClient + NoopStore. Shared across
// Bot tests that don't care about the kernel's internals beyond "takes
// PlatformEvents, emits RenderFrames".
func newTestKernel(t *testing.T) *kernel.Kernel {
	t.Helper()
	reg := tools.NewRegistry()
	reg.MustRegister(&tools.EchoTool{})
	return kernel.New(kernel.Config{
		Model:             "hermes-agent",
		Endpoint:          "http://mock",
		Admission:         kernel.Admission{MaxBytes: 200_000, MaxLines: 10_000},
		Tools:             reg,
		MaxToolIterations: 10,
		MaxToolDuration:   5 * time.Second,
	}, hermes.NewMockClient(), store.NewNoop(), telemetry.New(), nil)
}

// TestBot_RejectsUnauthorisedChat: inbound message from a non-allowed chat
// produces zero Send calls and zero kernel.Submit calls.
func TestBot_RejectsUnauthorisedChat(t *testing.T) {
	mc := newMockClient()
	k := newTestKernel(t)
	b := New(Config{
		AllowedChatID:     11111,
		FirstRunDiscovery: false,
	}, mc, k, nil)

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	go k.Run(ctx)
	<-k.Render() // drain initial idle

	go func() { _ = b.Run(ctx) }()

	mc.pushTextUpdate(22222, "hello from nowhere")

	time.Sleep(50 * time.Millisecond)

	if got := len(mc.sentMessages()); got != 0 {
		t.Errorf("sent messages = %d, want 0 (silent drop for unauthorised chat)", got)
	}

	cancel()
	mc.closeUpdates()
	time.Sleep(20 * time.Millisecond)
}

// TestBot_FirstRunDiscoveryRepliesWithChatID: zero AllowedChatID +
// FirstRunDiscovery enabled → one "not authorised" reply naming the
// allowed_chat_id config key.
func TestBot_FirstRunDiscoveryRepliesWithChatID(t *testing.T) {
	mc := newMockClient()
	k := newTestKernel(t)
	b := New(Config{
		AllowedChatID:     0,
		FirstRunDiscovery: true,
	}, mc, k, nil)

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	go k.Run(ctx)
	<-k.Render()
	go func() { _ = b.Run(ctx) }()

	mc.pushTextUpdate(77777, "hi")
	time.Sleep(50 * time.Millisecond)

	got := mc.lastSentText()
	if !strings.Contains(got, "not authorised") {
		t.Errorf("reply = %q, want to contain 'not authorised'", got)
	}
	if !strings.Contains(got, "allowed_chat_id") {
		t.Errorf("reply = %q, want to mention allowed_chat_id config key", got)
	}

	cancel()
	mc.closeUpdates()
	time.Sleep(20 * time.Millisecond)
}
