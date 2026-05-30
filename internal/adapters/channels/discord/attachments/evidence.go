package attachments

import (
	"errors"
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"
)

var (
	ErrReadUnavailable = errors.New("discord attachment read unavailable")
	ErrBlockedSSRF     = errors.New("discord attachment blocked by SSRF guard")
	ErrTooLarge        = errors.New("discord attachment too large")
)

// SourceID returns the stable gateway source ID for a Discord attachment.
func SourceID(att *discordgo.MessageAttachment) string {
	if att == nil {
		return ""
	}
	if id := strings.TrimSpace(att.ID); id != "" {
		return id
	}
	return SafeFileName(att.Filename)
}

// Evidence builds the redacted user-visible marker for a skipped attachment.
func Evidence(defaultCode string, att *discordgo.MessageAttachment, reason string) string {
	var parts []string
	if defaultCode = strings.TrimSpace(defaultCode); defaultCode == "" {
		defaultCode = "discord_attachment_unavailable"
	}
	parts = append(parts, "code="+defaultCode)
	if att != nil {
		if fileName := SafeFileName(att.Filename); fileName != "" {
			parts = append(parts, "file="+fileName)
		}
		if mediaType := CleanMediaType(att.ContentType); mediaType != "" {
			parts = append(parts, "mime="+mediaType)
		}
		if att.Size > 0 {
			parts = append(parts, fmt.Sprintf("size=%d bytes", att.Size))
		}
	}
	if reason = strings.TrimSpace(reason); reason != "" {
		parts = append(parts, "reason="+reason)
	}
	return "[Discord attachment skipped: " + strings.Join(parts, "; ") + "]"
}

// EvidenceReason maps internal attachment read/classification failures to redacted evidence text.
func EvidenceReason(err error) string {
	switch {
	case err == nil:
		return ""
	case errors.Is(err, ErrBlockedSSRF):
		return "blocked by SSRF guard"
	case errors.Is(err, ErrTooLarge):
		return "attachment exceeds 32 MB"
	case errors.Is(err, ErrInvalidPayload):
		return "invalid or HTML/error payload"
	default:
		return "download unavailable"
	}
}
