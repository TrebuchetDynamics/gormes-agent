package messaging

import (
	"os"
	"strings"

	"github.com/bwmarrin/discordgo"
)

const (
	discordAllowMentionEveryone    = "DISCORD_ALLOW_MENTION_EVERYONE"
	discordAllowMentionRoles       = "DISCORD_ALLOW_MENTION_ROLES"
	discordAllowMentionUsers       = "DISCORD_ALLOW_MENTION_USERS"
	discordAllowMentionRepliedUser = "DISCORD_ALLOW_MENTION_REPLIED_USER"
)

// BuildAllowedMentionsFromEnv mirrors Hermes' safe Discord defaults: user and
// reply mentions stay enabled, while everyone/here and role pings require an
// explicit operator opt-in.
func BuildAllowedMentionsFromEnv() *discordgo.MessageAllowedMentions {
	parse := make([]discordgo.AllowedMentionType, 0, 3)
	if envBool(discordAllowMentionEveryone, false) {
		parse = append(parse, discordgo.AllowedMentionTypeEveryone)
	}
	if envBool(discordAllowMentionRoles, false) {
		parse = append(parse, discordgo.AllowedMentionTypeRoles)
	}
	if envBool(discordAllowMentionUsers, true) {
		parse = append(parse, discordgo.AllowedMentionTypeUsers)
	}
	return &discordgo.MessageAllowedMentions{
		Parse:       parse,
		RepliedUser: envBool(discordAllowMentionRepliedUser, true),
	}
}

func envBool(name string, fallback bool) bool {
	raw := strings.TrimSpace(strings.ToLower(os.Getenv(name)))
	switch raw {
	case "":
		return fallback
	case "true", "1", "yes", "on":
		return true
	case "false", "0", "no", "off":
		return false
	default:
		return fallback
	}
}
