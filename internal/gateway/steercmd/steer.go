package steercmd

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

// Evidence strings are stable degraded-mode reasons surfaced before any
// future queue or running-agent dispatch path can handle /steer.
type Evidence string

const (
	EvidenceUsage              Evidence = "steer_usage"
	EvidencePayloadUnsupported Evidence = "steer_payload_unsupported"
	EvidenceQueued             Evidence = "steer_queued"
	EvidenceInjected           Evidence = "steer_injected"
	EvidenceUnavailable        Evidence = "steer_unavailable"
	EvidencePreview            Evidence = "steer_preview"
)

const (
	PreviewMaxRunes = 80

	commandName             = "/steer"
	previewTruncationMarker = "..."
)

// PayloadMetadata carries synthetic media counts for the pure parser.
// Platform adapters keep their own attachment details out of this slice.
type PayloadMetadata struct {
	ImageCount      int
	AttachmentCount int
}

// Command is the parsed shape of a /steer invocation.
type Command struct {
	Guidance string
	Preview  string
	Evidence Evidence
}

// Parse turns raw /steer slash text plus payload metadata into a pure parser
// result. It performs no queueing, session mutation, or dispatch.
func Parse(raw string, payload PayloadMetadata) Command {
	guidance, ok := commandGuidance(raw)
	if !ok {
		return Command{Evidence: EvidenceUsage}
	}
	if payload.ImageCount > 0 || payload.AttachmentCount > 0 {
		return Command{Evidence: EvidencePayloadUnsupported}
	}
	return Command{
		Guidance: guidance,
		Preview:  Preview(guidance),
	}
}

// Preview returns deterministic, bounded guidance text for acknowledgments and
// evidence. Truncation is marked with an ASCII suffix.
func Preview(guidance string) string {
	guidance = strings.TrimSpace(guidance)
	if utf8.RuneCountInString(guidance) <= PreviewMaxRunes {
		return guidance
	}

	limit := PreviewMaxRunes - utf8.RuneCountInString(previewTruncationMarker)
	if limit <= 0 {
		return truncateRunes(previewTruncationMarker, PreviewMaxRunes)
	}
	return truncateRunes(guidance, limit) + previewTruncationMarker
}

func commandGuidance(raw string) (string, bool) {
	body := strings.TrimSpace(raw)
	if body == "" {
		return "", false
	}

	command, rest := splitCommandToken(body)
	if !strings.EqualFold(command, commandName) {
		return "", false
	}

	guidance := strings.TrimSpace(rest)
	if guidance == "" {
		return "", false
	}
	return guidance, true
}

func splitCommandToken(body string) (string, string) {
	for i, r := range body {
		if unicode.IsSpace(r) {
			return body[:i], body[i:]
		}
	}
	return body, ""
}

func truncateRunes(s string, limit int) string {
	if limit <= 0 {
		return ""
	}
	if utf8.RuneCountInString(s) <= limit {
		return s
	}
	out := make([]rune, 0, limit)
	for _, r := range s {
		if len(out) == limit {
			break
		}
		out = append(out, r)
	}
	return string(out)
}
