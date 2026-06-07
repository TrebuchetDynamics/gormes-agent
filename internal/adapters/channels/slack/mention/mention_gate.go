package mention

import (
	"fmt"
	"os"
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/adapters/channels/internal/channelutil"
)

const SlackMentionPolicyUnavailable = "slack_mention_policy_unavailable"

type MentionPolicyConfig struct {
	RequireMention       any
	StrictMention        any
	FreeResponseChannels any
	LookupEnv            func(string) string
}

type MentionPolicy struct {
	RequireMention       bool
	StrictMention        bool
	FreeResponseChannels map[string]struct{}
	Evidence             []MentionPolicyEvidence
}

type MentionPolicyEvidence struct {
	Code   string
	Source string
	Reason string
}

type MentionGateInput struct {
	ChannelID        string
	UserID           string
	BotUserID        string
	Text             string
	Timestamp        string
	ThreadTS         string
	ChatType         string
	ActiveSession    bool
	ThreadMentioned  bool
	ReplyToBotThread bool
}

type MentionGateDecision struct {
	Process        bool
	Text           string
	RememberThread bool
	Evidence       []MentionPolicyEvidence
}

func ResolveMentionPolicy(cfg MentionPolicyConfig) MentionPolicy {
	lookup := cfg.LookupEnv
	if lookup == nil {
		lookup = os.Getenv
	}
	policy := MentionPolicy{
		RequireMention:       true,
		FreeResponseChannels: map[string]struct{}{},
	}

	requireRaw, requireSource, requireSet := mentionPolicyRawValue(cfg.RequireMention, lookup, "GORMES_SLACK_REQUIRE_MENTION", "SLACK_REQUIRE_MENTION")
	if requireSet {
		value, ok := parseMentionBool(requireRaw)
		if ok {
			policy.RequireMention = value
		} else {
			policy.Evidence = append(policy.Evidence, mentionPolicyEvidence(requireSource, "require_mention must be one of true,false,1,0,yes,no,on,off"))
		}
	}

	strictRaw, strictSource, strictSet := mentionPolicyRawValue(cfg.StrictMention, lookup, "GORMES_SLACK_STRICT_MENTION", "SLACK_STRICT_MENTION")
	if strictSet {
		value, ok := parseMentionBool(strictRaw)
		if ok {
			policy.StrictMention = value
		} else {
			policy.Evidence = append(policy.Evidence, mentionPolicyEvidence(strictSource, "strict_mention must be one of true,false,1,0,yes,no,on,off"))
		}
	}

	freeRaw, freeSource, freeSet := mentionPolicyRawValue(cfg.FreeResponseChannels, lookup, "GORMES_SLACK_FREE_RESPONSE_CHANNELS", "SLACK_FREE_RESPONSE_CHANNELS")
	if freeSet {
		channels, ok := parseFreeResponseChannels(freeRaw)
		if ok {
			policy.FreeResponseChannels = channels
		} else {
			policy.Evidence = append(policy.Evidence, mentionPolicyEvidence(freeSource, "free_response_channels must be a CSV string, scalar, or list"))
		}
	}

	return policy
}

func (p MentionPolicy) ChannelAllowsFreeResponse(channelID string) bool {
	_, ok := p.FreeResponseChannels[strings.TrimSpace(channelID)]
	return ok
}

func EvaluateMentionGate(policy MentionPolicy, input MentionGateInput) MentionGateDecision {
	text := strings.TrimSpace(input.Text)
	decision := MentionGateDecision{Text: text, Evidence: policy.Evidence}
	botUserID := strings.TrimSpace(input.BotUserID)
	if botUserID == "" || IsDM(input.ChannelID, input.ChatType) {
		decision.Process = true
		return decision
	}
	if policy.ChannelAllowsFreeResponse(input.ChannelID) || !policy.RequireMention {
		decision.Process = true
		return decision
	}

	mention := "<@" + botUserID + ">"
	isMentioned := strings.Contains(text, mention)
	threadReply := strings.TrimSpace(input.ThreadTS) != "" && strings.TrimSpace(input.ThreadTS) != strings.TrimSpace(input.Timestamp)
	if policy.StrictMention && !isMentioned {
		return decision
	}
	if !isMentioned {
		if threadReply && (input.ActiveSession || input.ThreadMentioned || input.ReplyToBotThread) {
			decision.Process = true
		}
		return decision
	}

	decision.Process = true
	decision.Text = strings.TrimSpace(strings.ReplaceAll(text, mention, ""))
	if strings.TrimSpace(input.ThreadTS) != "" && !policy.StrictMention {
		decision.RememberThread = true
	}
	return decision
}

func mentionPolicyRawValue(configured any, lookup func(string) string, names ...string) (any, string, bool) {
	if configured != nil {
		return configured, "config", true
	}
	for _, name := range names {
		if value := strings.TrimSpace(lookup(name)); value != "" {
			return value, name, true
		}
	}
	return nil, "", false
}

func parseMentionBool(raw any) (bool, bool) {
	switch value := raw.(type) {
	case bool:
		return value, true
	case string:
		switch normalizeMentionBoolString(value) {
		case "1", "true", "yes", "on":
			return true, true
		case "0", "false", "no", "off":
			return false, true
		default:
			return false, false
		}
	case int:
		if value == 0 {
			return false, true
		}
		if value == 1 {
			return true, true
		}
	case int64:
		if value == 0 {
			return false, true
		}
		if value == 1 {
			return true, true
		}
	}
	return false, false
}

func parseFreeResponseChannels(raw any) (map[string]struct{}, bool) {
	out := map[string]struct{}{}
	switch value := raw.(type) {
	case nil:
		return out, true
	case string:
		addFreeResponseCSV(out, value)
	case []string:
		for _, item := range value {
			addFreeResponseChannel(out, item)
		}
	case []any:
		for _, item := range value {
			addFreeResponseChannel(out, fmt.Sprint(item))
		}
	case int, int64, float64:
		addFreeResponseChannel(out, fmt.Sprint(itemWithoutDecimal(value)))
	case bool:
		return out, false
	default:
		addFreeResponseChannel(out, fmt.Sprint(value))
	}
	return out, true
}

func itemWithoutDecimal(value any) any {
	if f, ok := value.(float64); ok && f == float64(int64(f)) {
		return int64(f)
	}
	return value
}

func addFreeResponseCSV(out map[string]struct{}, value string) {
	for _, part := range channelutil.SplitCommaList(value) {
		out[part] = struct{}{}
	}
}

func addFreeResponseChannel(out map[string]struct{}, value string) {
	trimmed := strings.TrimSpace(value)
	if trimmed != "" {
		out[trimmed] = struct{}{}
	}
}

func normalizeMentionBoolString(value string) string {
	trimmed := strings.TrimSpace(value)
	for len(trimmed) >= 2 {
		first := trimmed[0]
		last := trimmed[len(trimmed)-1]
		if (first != '"' && first != '\'') || first != last {
			break
		}
		trimmed = strings.TrimSpace(trimmed[1 : len(trimmed)-1])
	}
	return strings.ToLower(trimmed)
}

func IsDM(channelID, chatType string) bool {
	switch strings.ToLower(strings.TrimSpace(chatType)) {
	case "im", "mpim", "dm", "direct", "private":
		return true
	}
	return strings.HasPrefix(strings.TrimSpace(channelID), "D")
}

func mentionPolicyEvidence(source, reason string) MentionPolicyEvidence {
	return MentionPolicyEvidence{Code: SlackMentionPolicyUnavailable, Source: source, Reason: reason}
}
