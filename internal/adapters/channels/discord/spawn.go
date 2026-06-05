package discord

import (
	"context"
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"
)

func (b *Bot) CreateThread(ctx context.Context, channelID, name string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	starter, ok := b.session.(discordThreadStarter)
	if !ok {
		return "", fmt.Errorf("discord_thread_create_unavailable")
	}
	channelID = strings.TrimSpace(channelID)
	if channelID == "" {
		return "", fmt.Errorf("discord: CreateThread requires non-empty channel_id")
	}
	threadName := sanitizeDiscordThreadName(name)
	thread, err := starter.ThreadStartComplex(channelID, &discordgo.ThreadStart{
		Name:                threadName,
		AutoArchiveDuration: 1440,
		Type:                discordgo.ChannelTypeGuildPublicThread,
	})
	if err != nil {
		return "", err
	}
	if thread == nil || strings.TrimSpace(thread.ID) == "" {
		return "", fmt.Errorf("discord: thread create returned nil")
	}
	if strings.TrimSpace(thread.ParentID) == "" {
		thread.ParentID = channelID
	}
	if strings.TrimSpace(thread.Name) == "" {
		thread.Name = threadName
	}
	b.rememberThread(thread)
	if ev, err := b.participatedThreads.Mark(thread.ID); err != nil {
		b.log.Warn("discord thread participation tracker write failed", "evidence", ev.Code)
	}
	return strings.TrimSpace(thread.ID), nil
}

func (b *Bot) SendThread(ctx context.Context, chatID, threadID, text string) (string, error) {
	threadID = strings.TrimSpace(threadID)
	if threadID == "" {
		return b.Send(ctx, chatID, text)
	}
	return b.Send(ctx, threadID, text)
}
