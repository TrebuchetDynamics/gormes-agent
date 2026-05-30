package discord

import (
	"github.com/TrebuchetDynamics/gormes-agent/internal/adapters/channels/discord/messaging"
	"github.com/bwmarrin/discordgo"
)

// BuildAllowedMentionsFromEnv mirrors Hermes' safe Discord defaults: user and
// reply mentions stay enabled, while everyone/here and role pings require an
// explicit operator opt-in.
func BuildAllowedMentionsFromEnv() *discordgo.MessageAllowedMentions {
	return messaging.BuildAllowedMentionsFromEnv()
}
