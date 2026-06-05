package discord

import (
	"context"
	"errors"
	"io"
	"sync"
	"testing"
	"time"

	"github.com/bwmarrin/discordgo"
)

var errMockDiscordAttachmentReadUnavailable = errors.New("mock discord attachment read unavailable")

type mockSession struct {
	mu                       sync.Mutex
	opened                   bool
	openOnce                 sync.Once
	openCh                   chan struct{}
	closed                   bool
	handlers                 []interface{}
	sent                     []mockSent
	complexSent              []mockComplexSent
	edits                    []mockEdit
	reactionsAdded           []mockReaction
	reactionsRemove          []mockReaction
	applicationCommands      []*discordgo.ApplicationCommand
	interactionResponses     []*discordgo.InteractionResponse
	followups                []*discordgo.WebhookParams
	threadStartCalls         []mockThreadStart
	messageThreadStarts      []mockMessageThreadStart
	nextMsgID                int
	currentUserID            string
	openErr                  error
	sendErr                  error
	sendErrWhenReference     error
	editErr                  error
	reactionErr              error
	commandErr               error
	interactionErr           error
	threadStartErr           error
	messageThreadErr         error
	threadStartResult        *discordgo.Channel
	messageThreadStartResult *discordgo.Channel
	attachmentBytes          map[string][]byte
	attachmentErr            error
	attachmentReads          int
}

type mockSent struct{ ChannelID, Content, MsgID string }
type mockThreadStart struct {
	ChannelID string
	Data      *discordgo.ThreadStart
}
type mockMessageThreadStart struct {
	ChannelID string
	MessageID string
	Data      *discordgo.ThreadStart
}
type mockComplexSent struct {
	ChannelID string
	MsgID     string
	Data      *discordgo.MessageSend
	FileNames []string
	FileBytes [][]byte
}
type mockEdit struct{ ChannelID, MsgID, Content string }
type mockReaction struct{ ChannelID, MsgID, Emoji string }

func newMockSession() *mockSession {
	return &mockSession{
		nextMsgID:       1000,
		attachmentBytes: map[string][]byte{},
		openCh:          make(chan struct{}),
	}
}

func (m *mockSession) Open() error {
	m.mu.Lock()
	m.opened = true
	err := m.openErr
	m.mu.Unlock()
	m.openOnce.Do(func() { close(m.openCh) })
	return err
}

// waitOpen blocks until Bot.Run reaches session.Open, after every AddHandler
// call has returned. Use this in tests instead of time.Sleep to deterministically
// synchronize handler registration with mock event delivery.
func (m *mockSession) waitOpen(t *testing.T) {
	t.Helper()
	select {
	case <-m.openCh:
	case <-time.After(2 * time.Second):
		t.Fatal("discord mock: timed out waiting for session.Open()")
	}
}

func (m *mockSession) Close() error {
	m.mu.Lock()
	m.closed = true
	m.mu.Unlock()
	return nil
}

func (m *mockSession) AddHandler(handler interface{}) func() {
	m.mu.Lock()
	m.handlers = append(m.handlers, handler)
	m.mu.Unlock()
	return func() {}
}

func (m *mockSession) CurrentUserID() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.currentUserID
}

func (m *mockSession) ApplicationCommandBulkOverwrite(_ string, _ string, commands []*discordgo.ApplicationCommand, _ ...discordgo.RequestOption) ([]*discordgo.ApplicationCommand, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.commandErr != nil {
		return nil, m.commandErr
	}
	m.applicationCommands = append([]*discordgo.ApplicationCommand(nil), commands...)
	return commands, nil
}

func (m *mockSession) InteractionRespond(_ *discordgo.Interaction, resp *discordgo.InteractionResponse, _ ...discordgo.RequestOption) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.interactionErr != nil {
		return m.interactionErr
	}
	m.interactionResponses = append(m.interactionResponses, resp)
	return nil
}

func (m *mockSession) FollowupMessageCreate(_ *discordgo.Interaction, _ bool, data *discordgo.WebhookParams, _ ...discordgo.RequestOption) (*discordgo.Message, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.interactionErr != nil {
		return nil, m.interactionErr
	}
	m.followups = append(m.followups, data)
	resp := &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
	}
	if data != nil {
		resp.Data = &discordgo.InteractionResponseData{Content: data.Content, Flags: data.Flags}
	}
	m.interactionResponses = append(m.interactionResponses, resp)
	id := nextID(&m.nextMsgID)
	return &discordgo.Message{ID: id}, nil
}

func (m *mockSession) ThreadStartComplex(channelID string, data *discordgo.ThreadStart, _ ...discordgo.RequestOption) (*discordgo.Channel, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.threadStartCalls = append(m.threadStartCalls, mockThreadStart{ChannelID: channelID, Data: data})
	if m.threadStartErr != nil {
		return nil, m.threadStartErr
	}
	if m.threadStartResult != nil {
		return m.threadStartResult, nil
	}
	return &discordgo.Channel{ID: nextID(&m.nextMsgID), ParentID: channelID, Name: data.Name, Type: discordgo.ChannelTypeGuildPublicThread}, nil
}

func (m *mockSession) MessageThreadStartComplex(channelID, messageID string, data *discordgo.ThreadStart, _ ...discordgo.RequestOption) (*discordgo.Channel, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.messageThreadStarts = append(m.messageThreadStarts, mockMessageThreadStart{ChannelID: channelID, MessageID: messageID, Data: data})
	if m.messageThreadErr != nil {
		return nil, m.messageThreadErr
	}
	if m.messageThreadStartResult != nil {
		return m.messageThreadStartResult, nil
	}
	return &discordgo.Channel{ID: nextID(&m.nextMsgID), ParentID: channelID, Name: data.Name, Type: discordgo.ChannelTypeGuildPublicThread}, nil
}

func (m *mockSession) ChannelMessageSend(channelID, content string) (*discordgo.Message, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.sendErr != nil {
		return nil, m.sendErr
	}
	id := nextID(&m.nextMsgID)
	m.sent = append(m.sent, mockSent{ChannelID: channelID, Content: content, MsgID: id})
	return &discordgo.Message{ID: id, ChannelID: channelID, Content: content}, nil
}

func (m *mockSession) ChannelMessageSendComplex(channelID string, data *discordgo.MessageSend) (*discordgo.Message, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.sendErr != nil {
		return nil, m.sendErr
	}
	if data != nil && data.Reference != nil && m.sendErrWhenReference != nil {
		return nil, m.sendErrWhenReference
	}
	id := nextID(&m.nextMsgID)
	sent := mockComplexSent{ChannelID: channelID, MsgID: id, Data: data}
	if data != nil {
		for _, file := range data.Files {
			if file == nil {
				continue
			}
			sent.FileNames = append(sent.FileNames, file.Name)
			if file.Reader != nil {
				body, _ := io.ReadAll(file.Reader)
				sent.FileBytes = append(sent.FileBytes, body)
			}
		}
	}
	m.complexSent = append(m.complexSent, sent)
	return &discordgo.Message{ID: id, ChannelID: channelID}, nil
}

func (m *mockSession) ChannelMessageEdit(channelID, messageID, content string) (*discordgo.Message, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.editErr != nil {
		return nil, m.editErr
	}
	m.edits = append(m.edits, mockEdit{ChannelID: channelID, MsgID: messageID, Content: content})
	return &discordgo.Message{ID: messageID, ChannelID: channelID, Content: content}, nil
}

func (m *mockSession) MessageReactionAdd(channelID, messageID, emoji string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.reactionErr != nil {
		return m.reactionErr
	}
	m.reactionsAdded = append(m.reactionsAdded, mockReaction{channelID, messageID, emoji})
	return nil
}

func (m *mockSession) MessageReactionRemoveMe(channelID, messageID, emoji string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.reactionsRemove = append(m.reactionsRemove, mockReaction{channelID, messageID, emoji})
	return nil
}

func (m *mockSession) ReadAttachment(_ context.Context, attachment *discordgo.MessageAttachment) ([]byte, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.attachmentReads++
	if m.attachmentErr != nil {
		return nil, m.attachmentErr
	}
	if attachment == nil {
		return nil, errMockDiscordAttachmentReadUnavailable
	}
	for _, key := range []string{attachment.ID, attachment.URL, attachment.Filename} {
		if data, ok := m.attachmentBytes[key]; ok {
			return append([]byte(nil), data...), nil
		}
	}
	return nil, errMockDiscordAttachmentReadUnavailable
}

func (m *mockSession) deliver(msg *discordgo.MessageCreate) bool {
	m.mu.Lock()
	handlers := append([]interface{}{}, m.handlers...)
	m.mu.Unlock()
	for _, h := range handlers {
		if fn, ok := h.(func(*discordgo.Session, *discordgo.MessageCreate)); ok {
			fn(nil, msg)
			return true
		}
	}
	return false
}

func (m *mockSession) deliverThreadCreate(thread *discordgo.ThreadCreate) bool {
	m.mu.Lock()
	handlers := append([]interface{}{}, m.handlers...)
	m.mu.Unlock()
	for _, h := range handlers {
		if fn, ok := h.(func(*discordgo.Session, *discordgo.ThreadCreate)); ok {
			fn(nil, thread)
			return true
		}
	}
	return false
}

func (m *mockSession) deliverThreadUpdate(thread *discordgo.ThreadUpdate) bool {
	m.mu.Lock()
	handlers := append([]interface{}{}, m.handlers...)
	m.mu.Unlock()
	for _, h := range handlers {
		if fn, ok := h.(func(*discordgo.Session, *discordgo.ThreadUpdate)); ok {
			fn(nil, thread)
			return true
		}
	}
	return false
}

func (m *mockSession) deliverThreadDelete(thread *discordgo.ThreadDelete) bool {
	m.mu.Lock()
	handlers := append([]interface{}{}, m.handlers...)
	m.mu.Unlock()
	for _, h := range handlers {
		if fn, ok := h.(func(*discordgo.Session, *discordgo.ThreadDelete)); ok {
			fn(nil, thread)
			return true
		}
	}
	return false
}

func (m *mockSession) applicationCommandsSnapshot() []*discordgo.ApplicationCommand {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]*discordgo.ApplicationCommand, len(m.applicationCommands))
	copy(out, m.applicationCommands)
	return out
}

func (m *mockSession) interactionResponsesSnapshot() []*discordgo.InteractionResponse {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]*discordgo.InteractionResponse, len(m.interactionResponses))
	copy(out, m.interactionResponses)
	return out
}

func (m *mockSession) threadStartCallsSnapshot() []mockThreadStart {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]mockThreadStart, len(m.threadStartCalls))
	copy(out, m.threadStartCalls)
	return out
}

func (m *mockSession) messageThreadStartCallsSnapshot() []mockMessageThreadStart {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]mockMessageThreadStart, len(m.messageThreadStarts))
	copy(out, m.messageThreadStarts)
	return out
}

func (m *mockSession) sentSnapshot() []mockSent {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]mockSent, len(m.sent))
	copy(out, m.sent)
	return out
}

func (m *mockSession) complexSnapshot() []mockComplexSent {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]mockComplexSent, len(m.complexSent))
	copy(out, m.complexSent)
	return out
}

func (m *mockSession) editsSnapshot() []mockEdit {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]mockEdit, len(m.edits))
	copy(out, m.edits)
	return out
}

func (m *mockSession) reactionsAddedSnapshot() []mockReaction {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]mockReaction, len(m.reactionsAdded))
	copy(out, m.reactionsAdded)
	return out
}

func (m *mockSession) closedSnapshot() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.closed
}

func (m *mockSession) attachmentReadCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.attachmentReads
}

func nextID(n *int) string {
	id := *n
	*n++
	return intToString(id)
}

func intToString(n int) string {
	if n == 0 {
		return "0"
	}
	buf := make([]byte, 0, 20)
	for n > 0 {
		buf = append([]byte{byte('0' + n%10)}, buf...)
		n /= 10
	}
	return string(buf)
}
