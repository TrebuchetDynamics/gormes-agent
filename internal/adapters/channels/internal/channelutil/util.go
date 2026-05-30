// Package channelutil provides reusable utility functions shared across
// channel adapter implementations. Extraction target: toSet and NormalizedPolicy
// are each duplicated in 3+ channel subpackages.
package channelutil

import (
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/kernel"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tools/trace"
)

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

// CompactStrings trims values and returns non-empty entries in their original
// order. It intentionally preserves duplicates for configuration lists where
// later validation may care about repeated values.
func CompactStrings(values []string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}

// UniqueStrings trims values and returns the first occurrence of each non-empty
// entry in original order.
func UniqueStrings(values []string) []string {
	out := make([]string, 0, len(values))
	seen := map[string]struct{}{}
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		out = append(out, trimmed)
	}
	return out
}

// FormatToolTrace joins SoulEntry texts and formats them as a tool trace block.
// Behavior is identical to the duplicated function found in discord/legacy/render.go
// and slack/render.go. Extracted to share across channel renderers.
func FormatToolTrace(events []kernel.SoulEntry) string {
	texts := make([]string, 0, len(events))
	for _, event := range events {
		texts = append(texts, event.Text)
	}
	return trace.FormatBlock(texts)
}

// TruncateRunes truncates a string to at most max runes and appends "..."
// when truncation occurs. Returns empty when max is <= 0. Behavior is
// identical to the duplicated function found in slack/approval_buttons.go
// and telegram/callbacks/approval.go.
func TruncateRunes(value string, max int) string {
	if max <= 0 {
		return ""
	}
	runes := []rune(value)
	if len(runes) <= max {
		return value
	}
	if max <= 3 {
		return string(runes[:max])
	}
	return string(runes[:max-3]) + "..."
}
