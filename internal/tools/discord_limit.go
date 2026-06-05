package tools

import "github.com/TrebuchetDynamics/gormes-agent/internal/tools/discord"

const (
	discordLimitMinimum = discord.LimitMinimum
	discordLimitMaximum = discord.LimitMaximum

	DiscordLimitEvidenceProvided  = discord.DiscordLimitEvidenceProvided
	DiscordLimitEvidenceClamped   = discord.DiscordLimitEvidenceClamped
	DiscordLimitEvidenceDefaulted = discord.DiscordLimitEvidenceDefaulted
)

type DiscordLimitEvidence = discord.DiscordLimitEvidence
type DiscordLimitNormalization = discord.DiscordLimitNormalization

func NormalizeDiscordLimit(action string, arguments map[string]any) DiscordLimitNormalization {
	return discord.NormalizeDiscordLimit(action, arguments)
}
