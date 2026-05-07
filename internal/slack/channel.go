package slack

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"sync"

	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway"
)

// Channel adapts the Slack Client seam onto the shared gateway manager.
type Channel struct {
	client Client
	log    *slog.Logger
	cfg    ChannelConfig

	selfUserID string

	mu               sync.RWMutex
	threadByChannel  map[string]string
	mentionedThreads map[string]struct{}
	threadContext    *ThreadContextCache
}

type ChannelConfig struct {
	RequireMention       any
	StrictMention        any
	FreeResponseChannels any
	ChannelSkillBindings any
	ChannelPrompts       any
	LookupEnv            func(string) string
}

var (
	_ gateway.Channel     = (*Channel)(nil)
	_ gateway.MediaSender = (*Channel)(nil)
)

func NewChannel(client Client, log *slog.Logger, cfgs ...ChannelConfig) *Channel {
	if log == nil {
		log = slog.Default()
	}
	cfg := ChannelConfig{RequireMention: false, LookupEnv: os.Getenv}
	if len(cfgs) > 0 {
		cfg = cfgs[0]
		if cfg.LookupEnv == nil {
			cfg.LookupEnv = os.Getenv
		}
	}
	return &Channel{
		client:           client,
		log:              log,
		cfg:              cfg,
		threadByChannel:  map[string]string{},
		mentionedThreads: map[string]struct{}{},
		threadContext:    newThreadContextCache(""),
	}
}

func (c *Channel) Name() string { return "slack" }

func (c *Channel) Run(ctx context.Context, inbox chan<- gateway.InboundEvent) error {
	selfID, err := c.client.AuthTest(ctx)
	if err != nil {
		return err
	}
	c.selfUserID = selfID
	c.threadContext.SetSelfUserID(selfID)

	return c.client.Run(ctx, func(e Event) {
		c.handleEvent(ctx, inbox, e)
	})
}

func (c *Channel) handleEvent(ctx context.Context, inbox chan<- gateway.InboundEvent, e Event) {
	if err := c.client.Ack(e.RequestID); err != nil {
		c.log.Warn("slack ack failed", "request_id", e.RequestID, "err", err)
		return
	}

	ev, ok := c.toInboundEvent(e)
	if !ok {
		return
	}
	select {
	case inbox <- ev:
	case <-ctx.Done():
	}
}

func (c *Channel) toInboundEvent(e Event) (gateway.InboundEvent, bool) {
	channelID := strings.TrimSpace(e.ChannelID)
	userID := strings.TrimSpace(e.UserID)
	if channelID == "" || userID == "" || userID == c.selfUserID {
		return gateway.InboundEvent{}, false
	}
	if ignoreSubtype(e.SubType) {
		return gateway.InboundEvent{}, false
	}

	threadTS := strings.TrimSpace(e.ThreadTS)
	c.rememberThread(channelID, threadTS)
	ts := strings.TrimSpace(e.Timestamp)
	replyToText := ""
	if isSlackThreadReply(threadTS, ts) {
		if len(e.ThreadReplies) > 0 {
			replyToText = c.threadContext.Store(channelID, threadTS, e.TeamID, e.ThreadReplies).ParentText
		} else {
			replyToText = c.threadContext.ParentText(channelID, threadTS, e.TeamID)
		}
	}

	kind, body := gateway.ParseInboundText(strings.TrimSpace(e.Text))
	if kind == gateway.EventSubmit {
		policy := ResolveMentionPolicy(MentionPolicyConfig{
			RequireMention:       c.cfg.RequireMention,
			StrictMention:        c.cfg.StrictMention,
			FreeResponseChannels: c.cfg.FreeResponseChannels,
			LookupEnv:            c.cfg.LookupEnv,
		})
		decision := EvaluateMentionGate(policy, MentionGateInput{
			ChannelID:       channelID,
			UserID:          userID,
			BotUserID:       c.selfUserID,
			Text:            e.Text,
			Timestamp:       ts,
			ThreadTS:        threadTS,
			ChatType:        e.ChatType,
			ThreadMentioned: c.threadMentioned(threadTS),
		})
		for _, evidence := range decision.Evidence {
			c.log.Warn(evidence.Code, "source", evidence.Source, "reason", evidence.Reason)
		}
		if !decision.Process {
			return gateway.InboundEvent{}, false
		}
		kind, body = gateway.ParseInboundText(decision.Text)
		if decision.RememberThread {
			c.rememberMentionedThread(threadTS)
		}
	}
	if kind == gateway.EventSubmit {
		var evidence []SlackRichTextEvidence
		body, evidence = augmentInboundText(body, e.Blocks, e.Attachments)
		for _, ev := range evidence {
			c.log.Warn(slackRichTextUnavailableCode, "source", ev.Source, "reason", ev.Reason)
		}
	}
	return gateway.InboundEvent{
		Platform:    "slack",
		AccountID:   strings.TrimSpace(e.TeamID),
		ChatID:      channelID,
		ChatType:    slackChatType(e.ChannelID, e.ChatType),
		UserID:      userID,
		ThreadID:    threadTS,
		MsgID:       ts,
		MessageID:   ts,
		ReplyToText: replyToText,
		Kind:        kind,
		Text:        body,
		AutoSkills:  gateway.ResolveChannelSkills(c.cfg.ChannelSkillBindings, channelID, ""),
		ChannelPrompt: gateway.ResolveChannelPrompt(
			c.cfg.ChannelPrompts,
			channelID,
			"",
		),
	}, true
}

func (c *Channel) Send(ctx context.Context, chatID, text string) (string, error) {
	return c.client.PostMessage(ctx, chatID, c.threadForChannel(chatID), text)
}

func (c *Channel) SendMedia(ctx context.Context, chatID, replyToMsgID string, media gateway.OutboundMedia) (string, error) {
	mediaPath := strings.TrimSpace(media.Path)
	if mediaPath == "" {
		return "", fmt.Errorf("slack: media path is required")
	}
	threadTS := strings.TrimSpace(media.ThreadID)
	if threadTS == "" {
		threadTS = strings.TrimSpace(replyToMsgID)
	}
	if threadTS == "" {
		threadTS = c.threadForChannel(chatID)
	}
	return c.client.UploadFile(ctx, chatID, threadTS, mediaPath)
}

func (c *Channel) rememberThread(channelID, threadTS string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if threadTS == "" {
		delete(c.threadByChannel, channelID)
		return
	}
	c.threadByChannel[channelID] = threadTS
}

func (c *Channel) threadForChannel(channelID string) string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.threadByChannel[channelID]
}

func (c *Channel) rememberMentionedThread(threadTS string) {
	threadTS = strings.TrimSpace(threadTS)
	if threadTS == "" {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.mentionedThreads[threadTS] = struct{}{}
}

func (c *Channel) threadMentioned(threadTS string) bool {
	threadTS = strings.TrimSpace(threadTS)
	if threadTS == "" {
		return false
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	_, ok := c.mentionedThreads[threadTS]
	return ok
}

func isSlackThreadReply(threadTS, ts string) bool {
	return strings.TrimSpace(threadTS) != "" && strings.TrimSpace(threadTS) != strings.TrimSpace(ts)
}

func slackChatType(channelID, chatType string) string {
	if slackMentionGateIsDM(channelID, chatType) {
		return "dm"
	}
	return strings.TrimSpace(chatType)
}
