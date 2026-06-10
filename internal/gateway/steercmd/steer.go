package steercmd

import (
	"strings"
	"unicode/utf8"

	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway/commandline"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/redaction"
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

	commandName             = "steer"
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
	preview := previewLineValue(guidance)
	if utf8.RuneCountInString(preview) <= PreviewMaxRunes {
		return preview
	}

	limit := PreviewMaxRunes - utf8.RuneCountInString(previewTruncationMarker)
	if limit <= 0 {
		return truncateRunes(previewTruncationMarker, PreviewMaxRunes)
	}
	return truncateRunes(preview, limit) + previewTruncationMarker
}

func commandGuidance(raw string) (string, bool) {
	body := strings.TrimSpace(raw)
	if body == "" {
		return "", false
	}

	command, rest := commandline.Split(body)
	if !isSlashCommandToken(command) || commandline.Name(command) != commandName {
		return "", false
	}

	guidance := strings.TrimSpace(rest)
	if guidance == "" {
		return "", false
	}
	return guidance, true
}

func isSlashCommandToken(token string) bool {
	return strings.HasPrefix(token, "/") || strings.HasPrefix(token, "／")
}

func previewLineValue(value string) string {
	value = collapseRedactedSecretAssignments(redaction.RedactSecrets(value))
	replacer := strings.NewReplacer(
		"`", "'",
		"*", "'",
	)
	return strings.Join(strings.Fields(replacer.Replace(value)), " ")
}

func collapseRedactedSecretAssignments(value string) string {
	replacer := strings.NewReplacer(
		"api_key=[redacted]", "[redacted]",
		"api-key=[redacted]", "[redacted]",
		"api key=[redacted]", "[redacted]",
		"authorization=[redacted]", "[redacted]",
		"bearer=[redacted]", "[redacted]",
		"token=[redacted]", "[redacted]",
		"secret=[redacted]", "[redacted]",
		"password=[redacted]", "[redacted]",
	)
	fields := strings.Fields(replacer.Replace(value))
	out := make([]string, 0, len(fields))
	for i := 0; i < len(fields); i++ {
		field := fields[i]
		lower := strings.ToLower(field)
		nextRedacted := i+1 < len(fields) && strings.Contains(strings.ToLower(fields[i+1]), "[redacted]")
		if steerSecretField(lower) && (strings.Contains(lower, "[redacted]") || nextRedacted) {
			out = append(out, "[redacted]")
			if nextRedacted {
				i++
			}
			continue
		}
		out = append(out, field)
	}
	return strings.Join(out, " ")
}

func steerSecretField(value string) bool {
	return strings.Contains(value, "api_key") || strings.Contains(value, "api-key") || strings.Contains(value, "apikey") || strings.Contains(value, "authorization") || strings.Contains(value, "bearer") || strings.Contains(value, "token") || strings.Contains(value, "secret") || strings.Contains(value, "password")
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
