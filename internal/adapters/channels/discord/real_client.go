package discord

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/bwmarrin/discordgo"
)

type realSession struct {
	s *discordgo.Session
}

var _ discordSession = (*realSession)(nil)
var _ discordCommandRegistrar = (*realSession)(nil)
var _ discordInteractionResponder = (*realSession)(nil)
var _ discordThreadStarter = (*realSession)(nil)

func NewRealSession(token string) (discordSession, error) {
	if token == "" {
		return nil, fmt.Errorf("discord: empty bot token")
	}
	s, err := discordgo.New("Bot " + token)
	if err != nil {
		return nil, fmt.Errorf("discord: new session: %w", err)
	}
	s.Identify.Intents = discordgo.IntentsGuilds |
		discordgo.IntentsGuildMessages |
		discordgo.IntentsDirectMessages |
		discordgo.IntentsMessageContent
	return &realSession{s: s}, nil
}

func (r *realSession) Open() error  { return r.s.Open() }
func (r *realSession) Close() error { return r.s.Close() }

func (r *realSession) AddHandler(handler interface{}) func() {
	return r.s.AddHandler(handler)
}

func (r *realSession) CurrentUserID() string {
	if r.s.State == nil || r.s.State.User == nil {
		return ""
	}
	return strings.TrimSpace(r.s.State.User.ID)
}

func (r *realSession) ApplicationCommandBulkOverwrite(appID, guildID string, commands []*discordgo.ApplicationCommand, options ...discordgo.RequestOption) ([]*discordgo.ApplicationCommand, error) {
	return r.s.ApplicationCommandBulkOverwrite(appID, guildID, commands, options...)
}

func (r *realSession) InteractionRespond(interaction *discordgo.Interaction, resp *discordgo.InteractionResponse, options ...discordgo.RequestOption) error {
	return r.s.InteractionRespond(interaction, resp, options...)
}

func (r *realSession) FollowupMessageCreate(interaction *discordgo.Interaction, wait bool, data *discordgo.WebhookParams, options ...discordgo.RequestOption) (*discordgo.Message, error) {
	return r.s.FollowupMessageCreate(interaction, wait, data, options...)
}

func (r *realSession) ThreadStartComplex(channelID string, data *discordgo.ThreadStart, options ...discordgo.RequestOption) (*discordgo.Channel, error) {
	return r.s.ThreadStartComplex(channelID, data, options...)
}

func (r *realSession) MessageThreadStartComplex(channelID, messageID string, data *discordgo.ThreadStart, options ...discordgo.RequestOption) (*discordgo.Channel, error) {
	return r.s.MessageThreadStartComplex(channelID, messageID, data, options...)
}

func (r *realSession) ChannelMessageSend(channelID, content string) (*discordgo.Message, error) {
	return r.s.ChannelMessageSend(channelID, content)
}

func (r *realSession) ChannelMessageSendComplex(channelID string, data *discordgo.MessageSend) (*discordgo.Message, error) {
	return r.s.ChannelMessageSendComplex(channelID, data)
}

func (r *realSession) ChannelMessageEdit(channelID, messageID, content string) (*discordgo.Message, error) {
	return r.s.ChannelMessageEdit(channelID, messageID, content)
}

func (r *realSession) MessageReactionAdd(channelID, messageID, emoji string) error {
	return r.s.MessageReactionAdd(channelID, messageID, emoji)
}

func (r *realSession) MessageReactionRemoveMe(channelID, messageID, emoji string) error {
	return r.s.MessageReactionRemove(channelID, messageID, emoji, "@me")
}

func (r *realSession) ReadAttachment(ctx context.Context, attachment *discordgo.MessageAttachment) ([]byte, error) {
	if attachment == nil || strings.TrimSpace(attachment.URL) == "" {
		return nil, errDiscordAttachmentReadUnavailable
	}
	if !discordTrustedAttachmentHost(attachment.URL) {
		return nil, errDiscordAttachmentReadUnavailable
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, attachment.URL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", r.s.Token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("discord attachment authenticated read: HTTP %d", resp.StatusCode)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, discordMaxAttachmentBytes+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > discordMaxAttachmentBytes {
		return nil, errDiscordAttachmentTooLarge
	}
	return data, nil
}
