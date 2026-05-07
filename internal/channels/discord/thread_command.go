package discord

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/bwmarrin/discordgo"

	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway"
)

const discordThreadNameLimit = 80

var discordMentionTokenRE = regexp.MustCompile(`<@[!&]?\d+>|<#\d+>`)

func (b *Bot) handleThreadCreateSlash(ctx context.Context, inbox chan<- gateway.InboundEvent, session discordSession, i *discordgo.InteractionCreate, data discordgo.ApplicationCommandInteractionData) {
	if !b.authorizeInteractionOrRespond(session, i, "thread") {
		return
	}
	_ = b.deferEphemeral(session, i.Interaction)

	name := sanitizeDiscordThreadName(discordOptionString(data, "name"))
	message := strings.TrimSpace(discordOptionString(data, "message"))
	archive := normalizeDiscordArchiveDuration(discordOptionInt(data, "auto_archive_duration", 1440))

	thread, err := b.createDiscordThread(session, strings.TrimSpace(i.ChannelID), name, message, archive)
	if err != nil {
		_ = b.followupEphemeral(session, i.Interaction, "Failed to create thread: "+err.Error())
		return
	}
	b.rememberThread(thread)
	if ev, err := b.participatedThreads.Mark(thread.ID); err != nil {
		b.log.Warn("discord thread participation tracker write failed", "evidence", ev.Code)
	}
	_ = b.followupEphemeral(session, i.Interaction, "Created thread <#"+thread.ID+">")

	if message == "" {
		return
	}
	ev := b.inboundEventFromInteraction(i, message)
	ev.Kind = gateway.EventSubmit
	ev.Text = message
	ev.ChatID = strings.TrimSpace(thread.ParentID)
	if ev.ChatID == "" {
		ev.ChatID = strings.TrimSpace(i.ChannelID)
	}
	ev.ChatName = strings.TrimSpace(thread.Name)
	ev.ThreadID = strings.TrimSpace(thread.ID)
	ev.ParentChatID = ev.ChatID
	if ev.GuildID == "" {
		ev.GuildID = strings.TrimSpace(thread.GuildID)
	}
	b.enqueueInteraction(ctx, inbox, ev)
}

func (b *Bot) createDiscordThread(session discordSession, channelID, name, message string, archive int) (*discordgo.Channel, error) {
	starter, ok := session.(discordThreadStarter)
	if !ok {
		return nil, fmt.Errorf("discord_thread_create_unavailable")
	}
	data := &discordgo.ThreadStart{
		Name:                name,
		AutoArchiveDuration: archive,
		Type:                discordgo.ChannelTypeGuildPublicThread,
	}
	thread, directErr := starter.ThreadStartComplex(channelID, data)
	if directErr == nil && thread != nil {
		if strings.TrimSpace(thread.ParentID) == "" {
			thread.ParentID = channelID
		}
		if strings.TrimSpace(thread.Name) == "" {
			thread.Name = name
		}
		return thread, nil
	}

	seedText := strings.TrimSpace(message)
	if seedText == "" {
		seedText = "Thread created by Gormes: **" + name + "**"
	}
	seed, sendErr := session.ChannelMessageSendComplex(channelID, &discordgo.MessageSend{
		Content:         seedText,
		AllowedMentions: BuildAllowedMentionsFromEnv(),
	})
	if sendErr != nil {
		return nil, sendErr
	}
	if seed == nil || strings.TrimSpace(seed.ID) == "" {
		return nil, fmt.Errorf("seed message missing id")
	}
	thread, fallbackErr := starter.MessageThreadStartComplex(channelID, seed.ID, data)
	if fallbackErr != nil {
		return nil, fallbackErr
	}
	if thread == nil {
		return nil, fmt.Errorf("thread create returned nil")
	}
	if strings.TrimSpace(thread.ParentID) == "" {
		thread.ParentID = channelID
	}
	if strings.TrimSpace(thread.Name) == "" {
		thread.Name = name
	}
	return thread, nil
}

func sanitizeDiscordThreadName(name string) string {
	name = discordMentionTokenRE.ReplaceAllString(strings.TrimSpace(name), "")
	name = strings.Join(strings.Fields(name), " ")
	if name == "" {
		name = "Gormes"
	}
	if utf8.RuneCountInString(name) <= discordThreadNameLimit {
		return name
	}
	runes := []rune(name)
	if discordThreadNameLimit <= 3 {
		return string(runes[:discordThreadNameLimit])
	}
	return strings.TrimSpace(string(runes[:discordThreadNameLimit-3])) + "..."
}

func normalizeDiscordArchiveDuration(value int) int {
	switch value {
	case 60, 1440, 4320, 10080:
		return value
	default:
		return 1440
	}
}
