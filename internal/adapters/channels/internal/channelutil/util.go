// Package channelutil provides reusable utility functions shared across
// channel adapter implementations. Extraction target: toSet and NormalizedPolicy
// are each duplicated in 3+ channel subpackages.
package channelutil

import "strings"

// ToSet converts a string slice to a set map, trimming whitespace and
// skipping empty entries. Behavior is identical to the duplicated helper
// found in wecom, weixin, qqbot, feishu, and homeassistant.
func ToSet(values []string) map[string]struct{} {
	out := make(map[string]struct{}, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		out[value] = struct{}{}
	}
	return out
}

// NormalizedPolicy lowercases and trims a policy string. Returns "open" when
// policy is empty. Behavior matches the duplicated local helper in wecom,
// qqbot, and feishu.
func NormalizedPolicy(policy string) string {
	policy = strings.TrimSpace(strings.ToLower(policy))
	if policy == "" {
		return "open"
	}
	return policy
}

// StripLeadingMentions removes leading @-mention tokens from text. Used by
// qqbot and feishu to normalize group mentions before parsing.
func StripLeadingMentions(text string) string {
	fields := strings.Fields(strings.TrimSpace(text))
	for len(fields) > 0 {
		if !strings.HasPrefix(fields[0], "@") {
			break
		}
		fields = fields[1:]
	}
	return strings.TrimSpace(strings.Join(fields, " "))
}

// FirstNonEmpty returns the first non-empty (after trimming whitespace)
// string from values, with trailing/leading whitespace removed. Returns ""
// when all values are empty. Used by 8+ channel subpackages.
func FirstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}
