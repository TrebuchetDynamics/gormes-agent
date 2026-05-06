package hermes

import (
	"reflect"
	"strconv"
	"strings"
)

const (
	ManualCompressionSessionSplitCode     = "manual_compression_session_split"
	ManualCompressionSessionUnchangedCode = "manual_compression_session_unchanged"
)

type ManualCompressionFeedback struct {
	Noop      bool
	Headline  string
	TokenLine string
	Note      string
}

type ManualCompressionSessionEvidence struct {
	Code              string `json:"code"`
	OldSessionID      string `json:"old_session_id,omitempty"`
	NewSessionID      string `json:"new_session_id,omitempty"`
	PendingTitleReset bool   `json:"pending_title_reset,omitempty"`
	Message           string `json:"message"`
}

func SummarizeManualCompression(before, after []Message, beforeTokens, afterTokens int) ManualCompressionFeedback {
	beforeCount := len(before)
	afterCount := len(after)
	noop := reflect.DeepEqual(before, after)

	out := ManualCompressionFeedback{Noop: noop}
	if noop {
		out.Headline = "No changes from compression: " + strconv.Itoa(beforeCount) + " messages"
		if afterTokens == beforeTokens {
			out.TokenLine = "Approx request size: ~" + commaInt(beforeTokens) + " tokens (unchanged)"
		} else {
			out.TokenLine = "Approx request size: ~" + commaInt(beforeTokens) + " -> ~" + commaInt(afterTokens) + " tokens"
		}
		return out
	}

	out.Headline = "Compressed: " + strconv.Itoa(beforeCount) + " -> " + strconv.Itoa(afterCount) + " messages"
	out.TokenLine = "Approx request size: ~" + commaInt(beforeTokens) + " -> ~" + commaInt(afterTokens) + " tokens"
	if afterCount < beforeCount && afterTokens > beforeTokens {
		out.Note = "Note: fewer messages can still raise this estimate when compression rewrites the transcript into denser summaries."
	}
	return out
}

func ParseManualCompressionFocus(command string) string {
	raw := strings.TrimSpace(command)
	if raw == "" {
		return ""
	}
	parts := strings.SplitN(raw, " ", 2)
	if len(parts) < 2 {
		return ""
	}
	head := strings.TrimPrefix(strings.ToLower(strings.TrimSpace(parts[0])), "/")
	if head != "compress" {
		return ""
	}
	return strings.TrimSpace(parts[1])
}

func ManualCompressionSessionSplit(oldSessionID, newSessionID string) ManualCompressionSessionEvidence {
	oldSessionID = strings.TrimSpace(oldSessionID)
	newSessionID = strings.TrimSpace(newSessionID)
	if oldSessionID == "" || newSessionID == "" || oldSessionID == newSessionID {
		return ManualCompressionSessionEvidence{
			Code:    ManualCompressionSessionUnchangedCode,
			Message: "manual compression did not change the active session",
		}
	}
	return ManualCompressionSessionEvidence{
		Code:              ManualCompressionSessionSplitCode,
		OldSessionID:      oldSessionID,
		NewSessionID:      newSessionID,
		PendingTitleReset: true,
		Message:           "manual compression created a continuation session and reset pending title state",
	}
}

func commaInt(n int) string {
	if n == 0 {
		return "0"
	}
	sign := ""
	if n < 0 {
		sign = "-"
		n = -n
	}
	raw := strconv.Itoa(n)
	if len(raw) <= 3 {
		return sign + raw
	}
	var b strings.Builder
	b.Grow(len(raw) + (len(raw)-1)/3 + len(sign))
	b.WriteString(sign)
	prefix := len(raw) % 3
	if prefix == 0 {
		prefix = 3
	}
	b.WriteString(raw[:prefix])
	for i := prefix; i < len(raw); i += 3 {
		b.WriteByte(',')
		b.WriteString(raw[i : i+3])
	}
	return b.String()
}
