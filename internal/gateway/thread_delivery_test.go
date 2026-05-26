package gateway

import (
	"context"
	"log/slog"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/hermes"
	"github.com/TrebuchetDynamics/gormes-agent/internal/kernel"
)

type threadAwareFakeChannel struct {
	name    string
	inbox   chan<- InboundEvent
	started chan struct{}

	mu            sync.Mutex
	nextMsgID     int
	threadSends   []threadSendCall
	threadReplies []threadReplyCall
	threadActions []threadActionCall
}

type threadSendCall struct {
	ChatID   string
	ThreadID string
	Text     string
	MsgID    string
}

type threadReplyCall struct {
	ChatID       string
	ThreadID     string
	ReplyToMsgID string
	Text         string
	MsgID        string
}

type threadActionCall struct {
	ChatID   string
	ThreadID string
	Action   string
}

func newThreadAwareFakeChannel(name string) *threadAwareFakeChannel {
	return &threadAwareFakeChannel{
		name:      name,
		started:   make(chan struct{}),
		nextMsgID: 3000,
	}
}

func (f *threadAwareFakeChannel) Name() string { return f.name }

func (f *threadAwareFakeChannel) Run(ctx context.Context, inbox chan<- InboundEvent) error {
	f.mu.Lock()
	f.inbox = inbox
	f.mu.Unlock()
	close(f.started)
	<-ctx.Done()
	return nil
}

func (f *threadAwareFakeChannel) Send(_ context.Context, chatID, text string) (string, error) {
	return f.SendThread(context.Background(), chatID, "", text)
}

func (f *threadAwareFakeChannel) SendThread(_ context.Context, chatID, threadID, text string) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	id := f.nextID()
	f.threadSends = append(f.threadSends, threadSendCall{ChatID: chatID, ThreadID: threadID, Text: text, MsgID: id})
	return id, nil
}

func (f *threadAwareFakeChannel) SendReply(ctx context.Context, chatID, replyToMsgID, text string) (string, error) {
	return f.SendThreadReply(ctx, chatID, "", replyToMsgID, text)
}

func (f *threadAwareFakeChannel) SendThreadReply(_ context.Context, chatID, threadID, replyToMsgID, text string) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	id := f.nextID()
	f.threadReplies = append(f.threadReplies, threadReplyCall{ChatID: chatID, ThreadID: threadID, ReplyToMsgID: replyToMsgID, Text: text, MsgID: id})
	return id, nil
}

func (f *threadAwareFakeChannel) SendPlaceholder(ctx context.Context, chatID string) (string, error) {
	return f.SendThreadPlaceholder(ctx, chatID, "")
}

func (f *threadAwareFakeChannel) SendThreadPlaceholder(_ context.Context, chatID, threadID string) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	id := f.nextID()
	f.threadSends = append(f.threadSends, threadSendCall{ChatID: chatID, ThreadID: threadID, Text: "placeholder", MsgID: id})
	return id, nil
}

func (f *threadAwareFakeChannel) SendReplyPlaceholder(ctx context.Context, chatID, replyToMsgID string) (string, error) {
	return f.SendThreadReplyPlaceholder(ctx, chatID, "", replyToMsgID)
}

func (f *threadAwareFakeChannel) SendThreadReplyPlaceholder(_ context.Context, chatID, threadID, replyToMsgID string) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	id := f.nextID()
	f.threadReplies = append(f.threadReplies, threadReplyCall{ChatID: chatID, ThreadID: threadID, ReplyToMsgID: replyToMsgID, Text: "placeholder", MsgID: id})
	return id, nil
}

func (f *threadAwareFakeChannel) EditMessage(context.Context, string, string, string) error {
	return nil
}

func (f *threadAwareFakeChannel) SendChatAction(ctx context.Context, chatID, action string) error {
	return f.SendThreadChatAction(ctx, chatID, "", action)
}

func (f *threadAwareFakeChannel) SendThreadChatAction(_ context.Context, chatID, threadID, action string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.threadActions = append(f.threadActions, threadActionCall{ChatID: chatID, ThreadID: threadID, Action: action})
	return nil
}

func (f *threadAwareFakeChannel) nextID() string {
	id := f.nextMsgID
	f.nextMsgID++
	return strconv.Itoa(id)
}

func (f *threadAwareFakeChannel) threadSendSnapshot() []threadSendCall {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]threadSendCall, len(f.threadSends))
	copy(out, f.threadSends)
	return out
}

func (f *threadAwareFakeChannel) threadReplySnapshot() []threadReplyCall {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]threadReplyCall, len(f.threadReplies))
	copy(out, f.threadReplies)
	return out
}

func (f *threadAwareFakeChannel) threadActionSnapshot() []threadActionCall {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]threadActionCall, len(f.threadActions))
	copy(out, f.threadActions)
	return out
}

func pinThreadedTurn(m *Manager, threadID string) {
	pinThreadedTurnInChat(m, "-10042", threadID)
}

func pinThreadedTurnInChat(m *Manager, chatID, threadID string) {
	m.pinTurn("telegram", chatID, "msg-1")
	m.setPinnedTurnSession("telegram:"+chatID+":"+threadID, "sess-thread", SessionSource{
		Platform: "telegram",
		ChatID:   chatID,
		ThreadID: threadID,
	})
}

func TestThreadAwareFinalPageUsesPinnedTurnThread(t *testing.T) {
	ch := newThreadAwareFakeChannel("telegram")
	m := NewManagerWithSubmitter(ManagerConfig{}, nil, slog.Default())
	if err := m.Register(ch); err != nil {
		t.Fatalf("Register: %v", err)
	}
	pinThreadedTurn(m, "777")

	var co *coalescer
	var coCancel context.CancelFunc
	m.dispatchFrame(context.Background(), kernel.RenderFrame{
		Phase: kernel.PhaseIdle,
		History: []hermes.Message{
			{Role: "assistant", Content: "threaded final answer"},
		},
	}, &co, &coCancel)

	got := ch.threadSendSnapshot()
	if len(got) != 1 {
		t.Fatalf("thread sends = %+v, want one final page", got)
	}
	if got[0].ChatID != "-10042" || got[0].ThreadID != "777" || !strings.Contains(got[0].Text, "threaded final answer") {
		t.Fatalf("thread send = %+v, want chat -10042 thread 777 final text", got[0])
	}
}

func TestThreadAwareReplySendUsesPinnedTurnThread(t *testing.T) {
	ch := newThreadAwareFakeChannel("telegram")
	m := NewManagerWithSubmitter(ManagerConfig{}, nil, slog.Default())

	if _, err := m.sendWithHooksReplyThread(context.Background(), ch, "42", "777", "msg-1", "threaded reply"); err != nil {
		t.Fatalf("sendWithHooksReplyThread: %v", err)
	}

	got := ch.threadReplySnapshot()
	if len(got) != 1 {
		t.Fatalf("thread replies = %+v, want one reply", got)
	}
	if got[0] != (threadReplyCall{ChatID: "42", ThreadID: "777", ReplyToMsgID: "msg-1", Text: "threaded reply", MsgID: "3000"}) {
		t.Fatalf("thread reply = %+v, want threaded reply", got[0])
	}
}

func TestThreadAwareToolProgressUsesPinnedTurnThread(t *testing.T) {
	ch := newThreadAwareFakeChannel("telegram")
	m := NewManagerWithSubmitter(ManagerConfig{}, nil, slog.Default())
	if err := m.Register(ch); err != nil {
		t.Fatalf("Register: %v", err)
	}
	pinThreadedTurn(m, "888")

	var co *coalescer
	var coCancel context.CancelFunc
	m.dispatchFrame(context.Background(), kernel.RenderFrame{
		Phase: kernel.PhaseStreaming,
		SoulEvents: []kernel.SoulEntry{
			{At: time.Now(), Text: "tool: terminal: go test ./..."},
		},
	}, &co, &coCancel)

	got := ch.threadSendSnapshot()
	if len(got) != 1 {
		t.Fatalf("thread sends = %+v, want one tool-progress send", got)
	}
	if got[0].ChatID != "-10042" || got[0].ThreadID != "888" || !strings.Contains(got[0].Text, "⚙️ [runtime] Running test suite") {
		t.Fatalf("thread send = %+v, want threaded tool progress", got[0])
	}
}

func TestThreadAwareTypingActionUsesPinnedTurnThread(t *testing.T) {
	now := time.Date(2026, 5, 7, 10, 0, 0, 0, time.UTC)
	ch := newThreadAwareFakeChannel("telegram")
	m := NewManagerWithSubmitter(ManagerConfig{
		Now: func() time.Time { return now },
	}, nil, slog.Default())
	if err := m.Register(ch); err != nil {
		t.Fatalf("Register: %v", err)
	}
	pinThreadedTurn(m, "999")

	var co *coalescer
	var coCancel context.CancelFunc
	defer func() {
		if coCancel != nil {
			coCancel()
		}
	}()
	m.dispatchFrame(context.Background(), kernel.RenderFrame{
		Phase:     kernel.PhaseStreaming,
		DraftText: "working",
	}, &co, &coCancel)

	got := ch.threadActionSnapshot()
	if len(got) != 1 {
		t.Fatalf("thread actions = %+v, want one typing action", got)
	}
	if got[0] != (threadActionCall{ChatID: "-10042", ThreadID: "999", Action: "typing"}) {
		t.Fatalf("thread action = %+v, want chat -10042 thread 999 typing", got[0])
	}
}

func TestThreadAwareDMTopicNoAnchorDropsThread(t *testing.T) {
	ch := newThreadAwareFakeChannel("telegram")
	m := NewManagerWithSubmitter(ManagerConfig{}, nil, slog.Default())

	if _, err := m.sendWithHooksReplyThread(context.Background(), ch, "42", "20197", "", "dm topic without anchor"); err != nil {
		t.Fatalf("sendWithHooksReplyThread: %v", err)
	}

	got := ch.threadSendSnapshot()
	if len(got) != 1 {
		t.Fatalf("thread sends = %+v, want one degraded plain send", got)
	}
	if got[0].ChatID != "42" || got[0].ThreadID != "" || got[0].Text != "dm topic without anchor" {
		t.Fatalf("thread send = %+v, want chat 42 without thread id", got[0])
	}
	if replies := ch.threadReplySnapshot(); len(replies) != 0 {
		t.Fatalf("thread replies = %+v, want no reply without anchor", replies)
	}
}

func TestThreadAwareDMTopicFinalPagesReplyAnchorEveryPage(t *testing.T) {
	ch := newThreadAwareFakeChannel("telegram")
	m := NewManagerWithSubmitter(ManagerConfig{}, nil, slog.Default())

	m.sendFinalPages(context.Background(), ch, "42", "20197", "462", []string{"first page", "second page"})

	got := ch.threadReplySnapshot()
	if len(got) != 2 {
		t.Fatalf("thread replies = %+v, want every page anchored as reply", got)
	}
	for i, call := range got {
		if call.ChatID != "42" || call.ThreadID != "20197" || call.ReplyToMsgID != "462" {
			t.Fatalf("reply[%d] = %+v, want chat 42 thread 20197 reply 462", i, call)
		}
	}
	if sends := ch.threadSendSnapshot(); len(sends) != 0 {
		t.Fatalf("thread sends = %+v, want no unanchored follow-up pages", sends)
	}
}

func TestThreadAwareDMTopicTypingActionSkipped(t *testing.T) {
	now := time.Date(2026, 5, 9, 10, 0, 0, 0, time.UTC)
	ch := newThreadAwareFakeChannel("telegram")
	m := NewManagerWithSubmitter(ManagerConfig{
		Now: func() time.Time { return now },
	}, nil, slog.Default())
	if err := m.Register(ch); err != nil {
		t.Fatalf("Register: %v", err)
	}
	pinThreadedTurnInChat(m, "42", "20197")

	var co *coalescer
	var coCancel context.CancelFunc
	defer func() {
		if coCancel != nil {
			coCancel()
		}
	}()
	m.dispatchFrame(context.Background(), kernel.RenderFrame{
		Phase:     kernel.PhaseStreaming,
		DraftText: "working",
	}, &co, &coCancel)

	if got := ch.threadActionSnapshot(); len(got) != 0 {
		t.Fatalf("thread actions = %+v, want DM topic typing action skipped", got)
	}
}
