package signal

import (
	"context"
	"errors"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway"
)

func TestBot_Run_CallsTransportStartBeforeEvents(t *testing.T) {
	mt := newMockTransport()
	b := New(mt, nil)

	inbox := make(chan gateway.InboundEvent, 1)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan error, 1)
	go func() { done <- b.Run(ctx, inbox) }()

	select {
	case <-mt.started:
	case <-time.After(200 * time.Millisecond):
		t.Fatal("Run did not call Transport.Start")
	}

	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Run exit err = %v, want nil", err)
		}
	case <-time.After(200 * time.Millisecond):
		t.Fatal("Run did not exit on cancel")
	}

	if got := atomic.LoadInt32(&mt.startCalls); got != 1 {
		t.Fatalf("Transport.Start calls = %d, want 1", got)
	}
	if got := atomic.LoadInt32(&mt.closeCalls); got != 1 {
		t.Fatalf("Transport.Close calls = %d, want 1", got)
	}
}

func TestBot_Run_ReturnsStartError(t *testing.T) {
	mt := newMockTransport()
	mt.startErr = errors.New("boot failed")
	b := New(mt, nil)

	err := b.Run(context.Background(), make(chan gateway.InboundEvent, 1))
	if err == nil {
		t.Fatal("Run err = nil, want start error")
	}
	if !errors.Is(err, mt.startErr) {
		t.Fatalf("Run err = %v, want wrapping %v", err, mt.startErr)
	}
	if !strings.Contains(err.Error(), "signal: start") {
		t.Fatalf("Run err = %q, want 'signal: start' prefix", err.Error())
	}
}

func TestBot_Run_TransientReceiveErrorDoesNotExit(t *testing.T) {
	mt := newMockTransport()
	b := New(mt, nil)

	inbox := make(chan gateway.InboundEvent, 1)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan error, 1)
	go func() { done <- b.Run(ctx, inbox) }()

	mt.pushError(ReceiveError{Err: errors.New("reconnecting"), Fatal: false})
	mt.push(InboundMessage{
		ChatType:   ChatTypeDirect,
		SenderID:   "+15551112222",
		SenderUUID: "uuid-a",
		SenderName: "Alice",
		MessageID:  "msg-a",
		Text:       "hello",
	})

	select {
	case <-inbox:
	case err := <-done:
		t.Fatalf("Run exited unexpectedly: %v", err)
	case <-time.After(200 * time.Millisecond):
		t.Fatal("event not delivered after transient receive error")
	}

	cancel()
	<-done
}

func TestBot_Run_FatalReceiveErrorAborts(t *testing.T) {
	mt := newMockTransport()
	b := New(mt, nil)

	inbox := make(chan gateway.InboundEvent, 1)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan error, 1)
	go func() { done <- b.Run(ctx, inbox) }()

	fatal := errors.New("auth revoked")
	mt.pushError(ReceiveError{Err: fatal, Fatal: true})

	select {
	case err := <-done:
		if err == nil {
			t.Fatal("Run err = nil, want fatal error")
		}
		if !errors.Is(err, fatal) {
			t.Fatalf("Run err = %v, want wrapping %v", err, fatal)
		}
	case <-time.After(200 * time.Millisecond):
		t.Fatal("Run did not exit on fatal receive error")
	}

	if got := atomic.LoadInt32(&mt.closeCalls); got != 1 {
		t.Fatalf("Transport.Close calls = %d after fatal, want 1", got)
	}
}

func TestBot_SendAttachments_ForwardsDirectAttachments(t *testing.T) {
	mt := newMockTransport()
	b := New(mt, nil)

	inbox := make(chan gateway.InboundEvent, 1)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { _ = b.Run(ctx, inbox) }()

	mt.push(InboundMessage{
		ChatType:   ChatTypeDirect,
		SenderID:   "+15551112222",
		SenderName: "Alice",
		MessageID:  "msg-a",
		Text:       "hello",
	})
	select {
	case <-inbox:
	case <-time.After(200 * time.Millisecond):
		t.Fatal("inbound event not delivered")
	}

	attachments := []Attachment{
		{Filename: "note.txt", ContentType: "text/plain", Data: []byte("body")},
	}
	msgID, err := b.SendAttachments(context.Background(), "+15551112222", "see attached", attachments)
	if err != nil {
		t.Fatalf("SendAttachments err = %v", err)
	}
	if msgID != "direct-send-1" {
		t.Fatalf("SendAttachments msgID = %q, want direct-send-1", msgID)
	}

	direct := mt.directSnapshot()
	if len(direct) != 1 {
		t.Fatalf("direct send count = %d, want 1", len(direct))
	}
	if direct[0].RecipientID != "+15551112222" {
		t.Fatalf("direct recipient = %q, want +15551112222", direct[0].RecipientID)
	}
	if direct[0].Text != "see attached" {
		t.Fatalf("direct text = %q, want 'see attached'", direct[0].Text)
	}
	if direct[0].Options.ReplyToMessageID != "msg-a" {
		t.Fatalf("direct reply metadata = %q, want msg-a", direct[0].Options.ReplyToMessageID)
	}
	if len(direct[0].Options.Attachments) != 1 {
		t.Fatalf("attachment count = %d, want 1", len(direct[0].Options.Attachments))
	}
	if direct[0].Options.Attachments[0].Filename != "note.txt" {
		t.Fatalf("attachment filename = %q, want note.txt", direct[0].Options.Attachments[0].Filename)
	}
	if string(direct[0].Options.Attachments[0].Data) != "body" {
		t.Fatalf("attachment data = %q, want body", direct[0].Options.Attachments[0].Data)
	}
}

func TestBot_SendAttachments_ForwardsGroupAttachments(t *testing.T) {
	mt := newMockTransport()
	b := New(mt, nil)

	inbox := make(chan gateway.InboundEvent, 1)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { _ = b.Run(ctx, inbox) }()

	mt.push(InboundMessage{
		ChatType:   ChatTypeGroup,
		GroupID:    "grp-987==",
		GroupName:  "Ops",
		SenderID:   "+15557778888",
		SenderName: "Bob",
		MessageID:  "msg-g",
		Text:       "hello group",
	})
	select {
	case <-inbox:
	case <-time.After(200 * time.Millisecond):
		t.Fatal("inbound event not delivered")
	}

	attachments := []Attachment{
		{Filename: "report.pdf", ContentType: "application/pdf", Data: []byte("pdf")},
	}
	msgID, err := b.SendAttachments(context.Background(), "group:grp-987==", "here is the report", attachments)
	if err != nil {
		t.Fatalf("SendAttachments err = %v", err)
	}
	if msgID != "group-send-1" {
		t.Fatalf("SendAttachments msgID = %q, want group-send-1", msgID)
	}

	group := mt.groupSnapshot()
	if len(group) != 1 {
		t.Fatalf("group send count = %d, want 1", len(group))
	}
	if group[0].RecipientID != "grp-987==" {
		t.Fatalf("group recipient = %q, want native group ID", group[0].RecipientID)
	}
	if group[0].Options.ReplyToMessageID != "msg-g" {
		t.Fatalf("group reply metadata = %q, want msg-g", group[0].Options.ReplyToMessageID)
	}
	if len(group[0].Options.Attachments) != 1 {
		t.Fatalf("attachment count = %d, want 1", len(group[0].Options.Attachments))
	}
	if group[0].Options.Attachments[0].ContentType != "application/pdf" {
		t.Fatalf("attachment content type = %q, want application/pdf", group[0].Options.Attachments[0].ContentType)
	}
}

func TestBot_SendAttachments_RejectsUnknownChatID(t *testing.T) {
	mt := newMockTransport()
	b := New(mt, nil)
	_, err := b.SendAttachments(context.Background(), "group:missing", "oops", nil)
	if err == nil {
		t.Fatal("SendAttachments err = nil, want unknown chat error")
	}
}

// mockTransport implements Transport for bot test fixtures.
type mockTransport struct {
	events  chan InboundMessage
	errors  chan ReceiveError
	started chan struct{}

	startCalls int32
	closeCalls int32
	startErr   error

	mu     sync.Mutex
	direct []sentMessage
	group  []sentMessage
}

func newMockTransport() *mockTransport {
	return &mockTransport{
		events:  make(chan InboundMessage, 16),
		errors:  make(chan ReceiveError, 16),
		started: make(chan struct{}, 1),
	}
}

func (m *mockTransport) Start(_ context.Context) error {
	atomic.AddInt32(&m.startCalls, 1)
	select {
	case m.started <- struct{}{}:
	default:
	}
	return m.startErr
}

func (m *mockTransport) Events() <-chan InboundMessage { return m.events }

func (m *mockTransport) Errors() <-chan ReceiveError { return m.errors }

func (m *mockTransport) SendDirect(_ context.Context, recipientID, text string, opts SendOptions) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.direct = append(m.direct, sentMessage{RecipientID: recipientID, Text: text, Options: opts})
	return "direct-send-1", nil
}

func (m *mockTransport) SendGroup(_ context.Context, recipientID, text string, opts SendOptions) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.group = append(m.group, sentMessage{RecipientID: recipientID, Text: text, Options: opts})
	return "group-send-1", nil
}

func (m *mockTransport) Close() error {
	atomic.AddInt32(&m.closeCalls, 1)
	return nil
}

func (m *mockTransport) push(msg InboundMessage) { m.events <- msg }

func (m *mockTransport) pushError(err ReceiveError) { m.errors <- err }

func (m *mockTransport) directSnapshot() []sentMessage {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]sentMessage, len(m.direct))
	copy(out, m.direct)
	return out
}

func (m *mockTransport) groupSnapshot() []sentMessage {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]sentMessage, len(m.group))
	copy(out, m.group)
	return out
}
