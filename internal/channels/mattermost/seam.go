package mattermost

import (
	"encoding/json"
	"strings"
	"sync"

	"github.com/TrebuchetDynamics/gormes-agent/internal/channels/threadtext"
	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway"
)

type GatingKind string

const (
	KindFree  GatingKind = "free"
	KindGated GatingKind = "gated"
)

type MentionGatingInputs struct {
	Kind            GatingKind
	FreeChannelIDs  map[string]bool
	AllowedChannels map[string]bool
}

type ProcessingHook func(msg threadtext.InboundMessage)

type ProcessingHooks struct {
	OnStart    ProcessingHook
	OnComplete ProcessingHook
	OnFailure  ProcessingHook
	OnCancel   ProcessingHook
}

type Seam struct {
	replyMode   threadtext.ReplyMode
	gatingInputs MentionGatingInputs
	hooks       ProcessingHooks
	botUserID   string
	cancelled   bool
	mu          sync.Mutex
	dedup       map[string]bool
}

type postedEvent struct {
	Event string       `json:"event"`
	Data  postedData   `json:"data"`
}

type postedData struct {
	Post        string `json:"post"`
	ChannelType string `json:"channel_type"`
	SenderName  string `json:"sender_name"`
}

type post struct {
	ID        string `json:"id"`
	UserID    string `json:"user_id"`
	Message   string `json:"message"`
	ChannelID string `json:"channel_id"`
	RootID    string `json:"root_id"`
	Type      string `json:"type"`
}

func NewSeam(
	replyMode threadtext.ReplyMode,
	gatingInputs MentionGatingInputs,
	botUserID string,
	hooks *ProcessingHooks,
) *Seam {
	s := &Seam{
		replyMode:    replyMode,
		gatingInputs: gatingInputs,
		botUserID:    botUserID,
		dedup:        make(map[string]bool),
	}
	if hooks != nil {
		s.hooks = *hooks
	}
	return s
}

func (s *Seam) ParsePostedEvent(rawJSON string) (gateway.InboundEvent, bool) {
	var ev postedEvent
	if err := json.Unmarshal([]byte(rawJSON), &ev); err != nil {
		return gateway.InboundEvent{}, false
	}
	if ev.Event != "posted" {
		return gateway.InboundEvent{}, false
	}
	if ev.Data.Post == "" {
		return gateway.InboundEvent{}, false
	}

	var p post
	if err := json.Unmarshal([]byte(ev.Data.Post), &p); err != nil {
		return gateway.InboundEvent{}, false
	}

	if p.UserID == s.botUserID || p.UserID == "" {
		return gateway.InboundEvent{}, false
	}

	if p.Type != "" {
		return gateway.InboundEvent{}, false
	}

	s.mu.Lock()
	if s.dedup[p.ID] {
		s.mu.Unlock()
		return gateway.InboundEvent{}, false
	}
	s.dedup[p.ID] = true
	s.mu.Unlock()

	if p.ID == "" {
		return gateway.InboundEvent{}, false
	}

	senderName := strings.TrimPrefix(ev.Data.SenderName, "@")
	if senderName == "" {
		senderName = p.UserID
	}

	if s.gatingInputs.Kind == KindGated && ev.Data.ChannelType != "D" {
		if len(s.gatingInputs.AllowedChannels) > 0 && !s.gatingInputs.AllowedChannels[p.ChannelID] {
			return gateway.InboundEvent{}, false
		}
		if !s.gatingInputs.FreeChannelIDs[p.ChannelID] && !s.containsBotMention(p.Message) {
			return gateway.InboundEvent{}, false
		}
	}

	messageText := p.Message
	if s.containsBotMention(messageText) {
		messageText = s.stripBotMention(messageText)
	}

	threadRootID := p.RootID

	msg := threadtext.InboundMessage{
		ChatID:       p.ChannelID,
		ChatName:     "",
		UserID:       p.UserID,
		UserName:     senderName,
		MessageID:    p.ID,
		Text:         messageText,
		ThreadID:     threadRootID,
		ThreadRootID: threadRootID,
	}

	if s.hooks.OnStart != nil {
		s.hooks.OnStart(msg)
	}

	evOut, ok := threadtext.NormalizeInbound("mattermost", msg)

	s.mu.Lock()
	cancelled := s.cancelled
	s.mu.Unlock()

	if !ok {
		if s.hooks.OnFailure != nil && !cancelled {
			s.hooks.OnFailure(msg)
		}
		return gateway.InboundEvent{}, false
	}

	if s.hooks.OnComplete != nil && !cancelled {
		s.hooks.OnComplete(msg)
	}

	return evOut, true
}

func (s *Seam) containsBotMention(text string) bool {
	lower := strings.ToLower(text)
	return strings.Contains(lower, "@"+strings.ToLower(s.botUserID))
}

func (s *Seam) stripBotMention(text string) string {
	pattern := "@" + s.botUserID
	result := text
	for {
		i := strings.Index(strings.ToLower(result), strings.ToLower(pattern))
		if i < 0 {
			break
		}
		before := result[:i]
		after := result[i+len(pattern):]
		if i > 0 && result[i-1] != ' ' {
			after = after
		}
		result = strings.TrimSpace(before + after)
	}
	return strings.TrimSpace(result)
}

func (s *Seam) ResolveReplyTarget(msg threadtext.InboundMessage) (threadtext.ReplyTarget, bool) {
	return threadtext.ResolveReplyTarget(msg, s.replyMode)
}

func (s *Seam) Cancel(lastMsg threadtext.InboundMessage) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cancelled = true
	if s.hooks.OnCancel != nil {
		s.hooks.OnCancel(lastMsg)
	}
}

func (s *Seam) Cancelled() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.cancelled
}
