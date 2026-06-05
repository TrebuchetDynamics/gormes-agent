package discord

import "github.com/TrebuchetDynamics/gormes-agent/internal/tools/discord/limits"

const (
	LimitMinimum = limits.LimitMinimum
	LimitMaximum = limits.LimitMaximum

	SearchMembersDefaultLimit = limits.SearchMembersDefaultLimit
	FetchMessagesDefaultLimit = limits.FetchMessagesDefaultLimit
)

type DiscordLimitEvidence = limits.Evidence

const (
	DiscordLimitEvidenceProvided  = limits.EvidenceProvided
	DiscordLimitEvidenceClamped   = limits.EvidenceClamped
	DiscordLimitEvidenceDefaulted = limits.EvidenceDefaulted
)

type DiscordLimitNormalization = limits.Normalization

// NormalizeDiscordLimit coerces the model-provided limit argument for Discord
// actions that expose bounded result limits.
func NormalizeDiscordLimit(action string, arguments map[string]any) DiscordLimitNormalization {
	return limits.Normalize(action, arguments)
}
