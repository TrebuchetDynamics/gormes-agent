package gateway

import (
	"context"
	"strconv"
	"sync"
)

type fakeChannel struct {
	name    string
	inbox   chan<- InboundEvent
	started chan struct{}

	mu            sync.Mutex
	sent          []fakeSent
	edits         []fakeEdit
	media         []fakeMedia
	placeholders  []string
	reactions     []fakeReaction
	typingChats   []string
	nextMsgID     int
	sendErr       error
	editErr       error
	reactionErr   error
	reactionUndos int
	typingStops   int
}

type fakeSent struct{ ChatID, Text, MsgID string }
type fakeEdit struct{ ChatID, MsgID, Text string }
type fakeMedia struct {
	ChatID       string
	ReplyToMsgID string
	Media        OutboundMedia
	MsgID        string
}
type fakeReaction struct{ ChatID, MsgID string }

func newFakeChannel(name string) *fakeChannel {
	return &fakeChannel{
		name:      name,
		started:   make(chan struct{}),
		nextMsgID: 1000,
	}
}

type channelOnlyFake struct {
	name    string
	inbox   chan<- InboundEvent
	started chan struct{}

	mu        sync.Mutex
	sent      []fakeSent
	nextMsgID int
}

func newChannelOnlyFake(name string) *channelOnlyFake {
	return &channelOnlyFake{
		name:      name,
		started:   make(chan struct{}),
		nextMsgID: 2000,
	}
}

func (f *channelOnlyFake) Name() string { return f.name }

func (f *channelOnlyFake) Run(ctx context.Context, inbox chan<- InboundEvent) error {
	f.mu.Lock()
	f.inbox = inbox
	f.mu.Unlock()
	close(f.started)
	<-ctx.Done()
	return nil
}

func (f *channelOnlyFake) Send(_ context.Context, chatID, text string) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	id := strconv.Itoa(f.nextMsgID)
	f.nextMsgID++
	f.sent = append(f.sent, fakeSent{ChatID: chatID, Text: text, MsgID: id})
	return id, nil
}

func (f *channelOnlyFake) pushInbound(e InboundEvent) {
	<-f.started
	f.mu.Lock()
	in := f.inbox
	f.mu.Unlock()
	in <- e
}

func (f *channelOnlyFake) sentSnapshot() []fakeSent {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]fakeSent, len(f.sent))
	copy(out, f.sent)
	return out
}

func (f *fakeChannel) Name() string { return f.name }

func (f *fakeChannel) Run(ctx context.Context, inbox chan<- InboundEvent) error {
	f.mu.Lock()
	f.inbox = inbox
	f.mu.Unlock()
	close(f.started)
	<-ctx.Done()
	return nil
}

func (f *fakeChannel) Send(_ context.Context, chatID, text string) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.sendErr != nil {
		return "", f.sendErr
	}
	id := strconv.Itoa(f.nextMsgID)
	f.nextMsgID++
	f.sent = append(f.sent, fakeSent{ChatID: chatID, Text: text, MsgID: id})
	return id, nil
}

func (f *fakeChannel) EditMessage(_ context.Context, chatID, msgID, text string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.editErr != nil {
		return f.editErr
	}
	f.edits = append(f.edits, fakeEdit{ChatID: chatID, MsgID: msgID, Text: text})
	return nil
}

func (f *fakeChannel) SendMedia(_ context.Context, chatID, replyToMsgID string, media OutboundMedia) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.sendErr != nil {
		return "", f.sendErr
	}
	id := strconv.Itoa(f.nextMsgID)
	f.nextMsgID++
	f.media = append(f.media, fakeMedia{ChatID: chatID, ReplyToMsgID: replyToMsgID, Media: media, MsgID: id})
	return id, nil
}

func (f *fakeChannel) SendPlaceholder(_ context.Context, chatID string) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.sendErr != nil {
		return "", f.sendErr
	}
	id := strconv.Itoa(f.nextMsgID)
	f.nextMsgID++
	f.placeholders = append(f.placeholders, chatID)
	f.sent = append(f.sent, fakeSent{ChatID: chatID, Text: "⏳", MsgID: id})
	return id, nil
}

func (f *fakeChannel) StartTyping(_ context.Context, chatID string) (func(), error) {
	f.mu.Lock()
	f.typingChats = append(f.typingChats, chatID)
	f.mu.Unlock()
	return func() {
		f.mu.Lock()
		f.typingStops++
		f.mu.Unlock()
	}, nil
}

func (f *fakeChannel) ReactToMessage(_ context.Context, chatID, msgID string) (func(), error) {
	f.mu.Lock()
	f.reactions = append(f.reactions, fakeReaction{ChatID: chatID, MsgID: msgID})
	f.mu.Unlock()
	return func() {
		f.mu.Lock()
		f.reactionUndos++
		f.mu.Unlock()
	}, nil
}

func (f *fakeChannel) OnProcessingStart(ctx context.Context, chatID, msgID string) error {
	if f.reactionErr != nil {
		return f.reactionErr
	}
	_, err := f.ReactToMessage(ctx, chatID, msgID)
	return err
}

func (f *fakeChannel) OnProcessingComplete(_ context.Context, _, _ string, _ ProcessingOutcome) error {
	return f.reactionErr
}

func (f *fakeChannel) pushInbound(e InboundEvent) {
	<-f.started
	f.mu.Lock()
	in := f.inbox
	f.mu.Unlock()
	in <- e
}

func (f *fakeChannel) sentSnapshot() []fakeSent {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]fakeSent, len(f.sent))
	copy(out, f.sent)
	return out
}

func (f *fakeChannel) editsSnapshot() []fakeEdit {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]fakeEdit, len(f.edits))
	copy(out, f.edits)
	return out
}

func (f *fakeChannel) mediaSnapshot() []fakeMedia {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]fakeMedia, len(f.media))
	copy(out, f.media)
	return out
}
