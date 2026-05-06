// Package discord adapts Discord bot traffic into the shared gateway chassis.
package discord

import (
	"context"

	"github.com/bwmarrin/discordgo"
)

// discordSession is the narrow surface of *discordgo.Session the adapter uses.
type discordSession interface {
	Open() error
	Close() error
	AddHandler(handler interface{}) func()
	ChannelMessageSend(channelID, content string) (*discordgo.Message, error)
	ChannelMessageSendComplex(channelID string, data *discordgo.MessageSend) (*discordgo.Message, error)
	ChannelMessageEdit(channelID, messageID, content string) (*discordgo.Message, error)
	MessageReactionAdd(channelID, messageID, emoji string) error
	MessageReactionRemoveMe(channelID, messageID, emoji string) error
	ReadAttachment(ctx context.Context, attachment *discordgo.MessageAttachment) ([]byte, error)
}
