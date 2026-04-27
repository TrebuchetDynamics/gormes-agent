package gateway

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

// Steer evidence strings are stable degraded-mode reasons surfaced before any
// future queue or running-agent dispatch path can handle /steer.
type SteerEvidence string

const (
	SteerEvidenceUsage              SteerEvidence = "steer_usage"
	SteerEvidencePayloadUnsupported SteerEvidence = "steer_payload_unsupported"
)

const (
	SteerPreviewMaxRunes = 80

	steerCommandName             = "/steer"
	steerPreviewTruncationMarker = "..."
)

// SteerPayloadMetadata carries synthetic media counts for the pure parser.
// Platform adapters keep their own attachment details out of this slice.
type SteerPayloadMetadata struct {
	ImageCount      int
	AttachmentCount int
}

// SteerCommand is the parsed shape of a /steer invocation.
type SteerCommand struct {
	Guidance string
	Preview  string
	Evidence SteerEvidence
}

// ParseSteerCommand turns raw /steer slash text plus payload metadata into a
// pure parser result. It performs no queueing, session mutation, or dispatch.
func ParseSteerCommand(raw string, payload SteerPayloadMetadata) SteerCommand {
	guidance, ok := steerCommandGuidance(raw)
	if !ok {
		return SteerCommand{Evidence: SteerEvidenceUsage}
	}
	if payload.ImageCount > 0 || payload.AttachmentCount > 0 {
		return SteerCommand{Evidence: SteerEvidencePayloadUnsupported}
	}
	return SteerCommand{
		Guidance: guidance,
		Preview:  SteerPreview(guidance),
	}
}

// SteerPreview returns deterministic, bounded guidance text for acknowledgments
// and evidence. Truncation is marked with an ASCII suffix.
func SteerPreview(guidance string) string {
	guidance = strings.TrimSpace(guidance)
	if utf8.RuneCountInString(guidance) <= SteerPreviewMaxRunes {
		return guidance
	}

	limit := SteerPreviewMaxRunes - utf8.RuneCountInString(steerPreviewTruncationMarker)
	if limit <= 0 {
		return truncateRunes(steerPreviewTruncationMarker, SteerPreviewMaxRunes)
	}
	return truncateRunes(guidance, limit) + steerPreviewTruncationMarker
}

func steerCommandGuidance(raw string) (string, bool) {
	body := strings.TrimSpace(raw)
	if body == "" {
		return "", false
	}

	command, rest := splitCommandToken(body)
	if !strings.EqualFold(command, steerCommandName) {
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
