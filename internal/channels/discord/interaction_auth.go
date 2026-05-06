package discord

import "strings"

const (
	DiscordInteractionAllowed        = "discord_interaction_allowed"
	DiscordInteractionMissingChannel = "discord_interaction_missing_channel"
	DiscordInteractionChannelDenied  = "discord_interaction_channel_denied"
	DiscordInteractionChannelIgnored = "discord_interaction_channel_ignored"
	DiscordInteractionMissingUser    = "discord_interaction_missing_user"
	DiscordInteractionUserDenied     = "discord_interaction_user_denied"
)

type DiscordInteractionContext struct {
	UserID          string
	RoleIDs         []string
	ChannelID       string
	ParentChannelID string
	IsDM            bool
}

type DiscordInteractionPolicy struct {
	AllowedUserIDs    []string
	AllowedRoleIDs    []string
	AllowedChannelIDs []string
	IgnoredChannelIDs []string
}

type DiscordInteractionAuthResult struct {
	Allowed bool
	Code    string
	Reason  string
}

func EvaluateInteractionAuthorization(ctx DiscordInteractionContext, policy DiscordInteractionPolicy) DiscordInteractionAuthResult {
	if !ctx.IsDM {
		channels := interactionChannelSet(ctx)
		allowedChannels := stringSet(policy.AllowedChannelIDs)
		if len(allowedChannels) > 0 && !allowedChannels["*"] {
			if len(channels) == 0 {
				return interactionDenied(DiscordInteractionMissingChannel, "channel id missing with channel allowlist configured")
			}
			if !setsIntersect(channels, allowedChannels) {
				return interactionDenied(DiscordInteractionChannelDenied, "channel not in allowed channels")
			}
		}

		ignoredChannels := stringSet(policy.IgnoredChannelIDs)
		if len(ignoredChannels) > 0 && (ignoredChannels["*"] || setsIntersect(channels, ignoredChannels)) {
			return interactionDenied(DiscordInteractionChannelIgnored, "channel in ignored channels")
		}
	}

	if !AuthorizeComponent(ctx, policy.AllowedUserIDs, policy.AllowedRoleIDs) {
		if strings.TrimSpace(ctx.UserID) == "" && (len(stringSet(policy.AllowedUserIDs)) > 0 || len(stringSet(policy.AllowedRoleIDs)) > 0) {
			return interactionDenied(DiscordInteractionMissingUser, "missing interaction user with allowlist configured")
		}
		return interactionDenied(DiscordInteractionUserDenied, "user not in allowed users or roles")
	}
	return DiscordInteractionAuthResult{Allowed: true, Code: DiscordInteractionAllowed}
}

func AuthorizeComponent(ctx DiscordInteractionContext, allowedUserIDs, allowedRoleIDs []string) bool {
	users := stringSet(allowedUserIDs)
	roles := stringSet(allowedRoleIDs)
	if len(users) == 0 && len(roles) == 0 {
		return true
	}

	userID := strings.TrimSpace(ctx.UserID)
	if userID == "" {
		return false
	}
	if users[userID] {
		return true
	}
	if len(roles) == 0 {
		return false
	}
	for _, roleID := range ctx.RoleIDs {
		if roles[strings.TrimSpace(roleID)] {
			return true
		}
	}
	return false
}

func AuthorizedSkillAutocomplete(ctx DiscordInteractionContext, policy DiscordInteractionPolicy, names []string, current string) []string {
	if !EvaluateInteractionAuthorization(ctx, policy).Allowed {
		return nil
	}
	needle := strings.ToLower(strings.TrimSpace(current))
	out := make([]string, 0, len(names))
	for _, name := range names {
		trimmed := strings.TrimSpace(name)
		if trimmed == "" {
			continue
		}
		if needle == "" || strings.Contains(strings.ToLower(trimmed), needle) {
			out = append(out, trimmed)
		}
	}
	return out
}

func interactionDenied(code, reason string) DiscordInteractionAuthResult {
	return DiscordInteractionAuthResult{Allowed: false, Code: code, Reason: reason}
}

func interactionChannelSet(ctx DiscordInteractionContext) map[string]bool {
	values := []string{ctx.ChannelID, ctx.ParentChannelID}
	return stringSet(values)
}

func stringSet(values []string) map[string]bool {
	set := make(map[string]bool, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed != "" {
			set[trimmed] = true
		}
	}
	return set
}

func setsIntersect(a, b map[string]bool) bool {
	for value := range a {
		if b[value] {
			return true
		}
	}
	return false
}
