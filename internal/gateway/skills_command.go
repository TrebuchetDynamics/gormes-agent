// Package gateway is the channel-agnostic messaging chassis for Gormes.
package gateway

import (
	"context"
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway/skillscmd"
)

// SkillsCommandOptions injects read-only seams for /skills command tests and
// non-network gateway/TUI callers. Zero values preserve the installed-skill
// list/inspect defaults and use an empty hub provider set for browse/search.
type SkillsCommandOptions = skillscmd.SkillsCommandOptions

// HandleSkillsCommand parses and executes /skills subcommands.
// Returns the text output to render in the channel.
func HandleSkillsCommand(body string) string { return skillscmd.HandleSkillsCommand(body) }

// HandleSkillsCommandWithOptions parses and executes /skills subcommands using
// explicit read-only dependencies. It is the shared gateway/TUI seam for
// installed list/inspect, hub browse/search, and mutating-action unavailable
// evidence.
func HandleSkillsCommandWithOptions(ctx context.Context, body string, opts SkillsCommandOptions) string {
	return skillscmd.HandleSkillsCommandWithOptions(ctx, body, opts)
}

type skillsPlainSender interface {
	SendPlain(ctx context.Context, chatID, text string) (string, error)
}

type skillsPlainReplySender interface {
	SendPlainReply(ctx context.Context, chatID, replyToMsgID, text string) (string, error)
}

func (m *Manager) handleSkillsCommand(ctx context.Context, ch Channel, ev InboundEvent) {
	_, _ = m.sendSkillsCommandReply(ctx, ch, ev.ChatID, ev.MsgID, HandleSkillsCommandWithOptions(ctx, ev.Text, m.cfg.SkillsCommandOptions))
}

func (m *Manager) sendSkillsCommandReply(ctx context.Context, ch Channel, chatID, replyToMsgID, text string) (string, error) {
	if ch != nil && strings.HasPrefix(ch.Name(), "telegram") {
		if replyToMsgID != "" {
			if sender, ok := ch.(skillsPlainReplySender); ok {
				return m.sendPlainSkillsWithHooks(ctx, ch, chatID, replyToMsgID, text, func(sendCtx context.Context) (string, error) {
					return sender.SendPlainReply(sendCtx, chatID, replyToMsgID, text)
				})
			}
		}
		if sender, ok := ch.(skillsPlainSender); ok {
			return m.sendPlainSkillsWithHooks(ctx, ch, chatID, replyToMsgID, text, func(sendCtx context.Context) (string, error) {
				return sender.SendPlain(sendCtx, chatID, text)
			})
		}
	}
	return m.sendWithHooksReply(ctx, ch, chatID, replyToMsgID, text)
}

func (m *Manager) sendPlainSkillsWithHooks(ctx context.Context, ch Channel, chatID, replyToMsgID, text string, send func(context.Context) (string, error)) (string, error) {
	if ch == nil {
		return "", nil
	}
	ev := HookEvent{
		Point:            HookBeforeSend,
		Platform:         ch.Name(),
		ChatID:           chatID,
		ReplyToMessageID: replyToMsgID,
		Text:             text,
	}
	m.fireHook(ctx, ev)
	msgID, err := send(ctx)
	if err != nil {
		m.writeRuntimeStatus(context.Background(), RuntimeStatusUpdate{
			Platform:      ch.Name(),
			PlatformState: PlatformStateFailed,
			ErrorMessage:  err.Error(),
		})
		m.fireHook(ctx, HookEvent{
			Point:            HookOnError,
			Platform:         ch.Name(),
			ChatID:           chatID,
			ReplyToMessageID: replyToMsgID,
			Text:             text,
			Err:              err,
		})
		return "", err
	}
	m.writeRuntimeStatus(context.Background(), RuntimeStatusUpdate{
		Platform:      ch.Name(),
		PlatformState: PlatformStateRunning,
	})
	m.fireHook(ctx, HookEvent{
		Point:            HookAfterSend,
		Platform:         ch.Name(),
		ChatID:           chatID,
		MsgID:            msgID,
		ReplyToMessageID: replyToMsgID,
		Text:             text,
	})
	return msgID, nil
}
