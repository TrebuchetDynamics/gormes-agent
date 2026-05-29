package discord

import "strings"

const (
	DiscordAdmissionAllowed        = "discord_admission_allowed"
	DiscordAdmissionOwnMessage     = "discord_own_message"
	DiscordAdmissionBotDenied      = "discord_bot_denied"
	DiscordAdmissionAllowedChannel = "discord_channel_denied"
	DiscordAdmissionIgnoredChannel = "discord_channel_ignored"
	DiscordAdmissionMentionMissing = "discord_mention_missing"
)

type AdmissionPolicy struct {
	AllowedChannelIDs     []string
	IgnoredChannelIDs     []string
	FreeResponseChannels  []string
	NoThreadChannelIDs    []string
	RequireMention        bool
	AutoThread            bool
	AllowBots             string
	KnownThreadBypass     bool
	ParticipatedThreadIDs []string
}

type AdmissionContext struct {
	ChannelID          string
	ParentChannelID    string
	GuildID            string
	AuthorID           string
	AuthorBot          bool
	SelfUserID         string
	Mentioned          bool
	ParticipatedThread bool
	IsDM               bool
	IsThread           bool
	IsReply            bool
}

type AdmissionResult struct {
	Allowed          bool
	Reason           string
	ShouldAutoThread bool
}

func EvaluateAdmission(policy AdmissionPolicy, ctx AdmissionContext) AdmissionResult {
	if trim(ctx.AuthorID) != "" && trim(ctx.SelfUserID) != "" && trim(ctx.AuthorID) == trim(ctx.SelfUserID) {
		return admissionDenied(DiscordAdmissionOwnMessage)
	}
	if ctx.AuthorBot {
		switch strings.ToLower(trim(policy.AllowBots)) {
		case "", "none":
			return admissionDenied(DiscordAdmissionBotDenied)
		case "mentions":
			if !ctx.Mentioned {
				return admissionDenied(DiscordAdmissionBotDenied)
			}
		case "all":
		default:
			return admissionDenied(DiscordAdmissionBotDenied)
		}
	}

	channels := admissionChannelSet(ctx)
	allowed := normalizeStringSet(policy.AllowedChannelIDs)
	if len(allowed) > 0 && !containsWildcard(allowed) && !admissionSetsIntersect(channels, allowed) {
		return admissionDenied(DiscordAdmissionAllowedChannel)
	}
	ignored := normalizeStringSet(policy.IgnoredChannelIDs)
	if containsWildcard(ignored) || admissionSetsIntersect(channels, ignored) {
		return admissionDenied(DiscordAdmissionIgnoredChannel)
	}

	free := normalizeStringSet(policy.FreeResponseChannels)
	isFree := containsWildcard(free) || admissionSetsIntersect(channels, free)
	participated := ctx.IsThread && (ctx.ParticipatedThread || participatedThread(policy, ctx))
	if !ctx.IsDM && policy.RequireMention && !isFree && !participated && !ctx.Mentioned {
		return admissionDenied(DiscordAdmissionMentionMissing)
	}

	noThread := normalizeStringSet(policy.NoThreadChannelIDs)
	skipThread := containsWildcard(noThread) || admissionSetsIntersect(channels, noThread)
	return AdmissionResult{
		Allowed: true,
		Reason:  DiscordAdmissionAllowed,
		ShouldAutoThread: policy.AutoThread &&
			!ctx.IsDM &&
			!ctx.IsThread &&
			!ctx.IsReply &&
			ctx.Mentioned &&
			!isFree &&
			!skipThread,
	}
}

func participatedThread(policy AdmissionPolicy, ctx AdmissionContext) bool {
	if !policy.KnownThreadBypass {
		return false
	}
	threads := normalizeStringSet(policy.ParticipatedThreadIDs)
	return threads[trim(ctx.ChannelID)] || threads[trim(ctx.ParentChannelID)]
}

func admissionDenied(reason string) AdmissionResult {
	return AdmissionResult{Allowed: false, Reason: reason}
}

func admissionChannelSet(ctx AdmissionContext) map[string]bool {
	out := map[string]bool{}
	if id := trim(ctx.ChannelID); id != "" {
		out[id] = true
	}
	if id := trim(ctx.ParentChannelID); id != "" {
		out[id] = true
	}
	return out
}

func normalizeStringSet(values []string) map[string]bool {
	out := make(map[string]bool, len(values))
	for _, value := range values {
		if v := trim(value); v != "" {
			out[v] = true
		}
	}
	return out
}

func containsWildcard(values map[string]bool) bool {
	return values["*"]
}

func admissionSetsIntersect(a, b map[string]bool) bool {
	for value := range a {
		if b[value] {
			return true
		}
	}
	return false
}

func trim(s string) string {
	return strings.TrimSpace(s)
}
