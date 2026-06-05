package discord

import "github.com/TrebuchetDynamics/gormes-agent/internal/adapters/channels/discord/interactions"

const (
	DiscordInteractionAllowed        = interactions.DiscordInteractionAllowed
	DiscordInteractionMissingChannel = interactions.DiscordInteractionMissingChannel
	DiscordInteractionChannelDenied  = interactions.DiscordInteractionChannelDenied
	DiscordInteractionChannelIgnored = interactions.DiscordInteractionChannelIgnored
	DiscordInteractionMissingUser    = interactions.DiscordInteractionMissingUser
	DiscordInteractionUserDenied     = interactions.DiscordInteractionUserDenied
)

type DiscordInteractionContext = interactions.DiscordInteractionContext
type DiscordInteractionPolicy = interactions.DiscordInteractionPolicy
type DiscordInteractionAuthResult = interactions.DiscordInteractionAuthResult

func EvaluateInteractionAuthorization(ctx DiscordInteractionContext, policy DiscordInteractionPolicy) DiscordInteractionAuthResult {
	return interactions.EvaluateInteractionAuthorization(ctx, policy)
}

func AuthorizeComponent(ctx DiscordInteractionContext, allowedUserIDs, allowedRoleIDs []string) bool {
	return interactions.AuthorizeComponent(ctx, allowedUserIDs, allowedRoleIDs)
}

func AuthorizedSkillAutocomplete(ctx DiscordInteractionContext, policy DiscordInteractionPolicy, names []string, current string) []string {
	return interactions.AuthorizedSkillAutocomplete(ctx, policy, names, current)
}
