package streaming

import (
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/adapters/channels/discord/legacy/protocol"
)

type sendCall struct {
	channelID string
	text      string
}

type editCall struct {
	channelID string
	messageID string
	text      string
}

type mockClient struct {
	mu          sync.Mutex
	selfID      string
	nextMessage int
	texts       []string
	sends       []sendCall
	edits       []editCall

	editErr error
}

var _ protocol.Client = (*mockClient)(nil)

func newMockClient(selfID string) *mockClient {
	return &mockClient{selfID: selfID, nextMessage: 1000}
}

func (m *mockClient) Open() error { return nil }

func (m *mockClient) Close() error { return nil }

func (m *mockClient) SelfID() string { return m.selfID }

func (m *mockClient) SetMessageHandler(func(protocol.InboundMessage)) {}

func (m *mockClient) Send(channelID, text string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	id := fmt.Sprintf("msg-%d", m.nextMessage)
	m.nextMessage++
	m.texts = append(m.texts, text)
	m.sends = append(m.sends, sendCall{channelID: channelID, text: text})
	return id, nil
}

func (m *mockClient) Edit(channelID, messageID, text string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.texts = append(m.texts, text)
	m.edits = append(m.edits, editCall{channelID: channelID, messageID: messageID, text: text})
	return m.editErr
}

func (m *mockClient) Typing(string) error { return nil }

func (m *mockClient) sendCalls() []sendCall {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]sendCall, len(m.sends))
	copy(out, m.sends)
	return out
}

func (m *mockClient) editCalls() []editCall {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]editCall, len(m.edits))
	copy(out, m.edits)
	return out
}

func (m *mockClient) lastSentText() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(m.texts) == 0 {
		return ""
	}
	return m.texts[len(m.texts)-1]
}

func TestCoalescerFlushImmediateEditFailureFallsBackToSend(t *testing.T) {
	mc := newMockClient("bot-1")
	mc.editErr = errors.New("edit failed")

	c := NewCoalescer(mc, time.Second, "chan-1")
	c.FlushImmediate("⏳")
	c.FlushImmediate("final")

	if got := len(mc.sendCalls()); got != 2 {
		t.Fatalf("send calls = %d, want 2 with fallback send", got)
	}
	if got := len(mc.editCalls()); got != 1 {
		t.Fatalf("edit calls = %d, want 1", got)
	}
	if got := mc.lastSentText(); got != "final" {
		t.Fatalf("last sent text = %q, want final", got)
	}
}
