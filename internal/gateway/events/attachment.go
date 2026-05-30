package events

import (
	"strconv"
	"strings"
)

// Attachment is the channel-neutral media descriptor attached to an inbound
// event. SourceID preserves the platform-side media identifier so failures can
// still be diagnosed even when URL resolution fails.
type Attachment struct {
	Kind      string `json:"kind"`
	URL       string `json:"url,omitempty"`
	MediaType string `json:"mediaType,omitempty"`
	FileName  string `json:"fileName,omitempty"`
	SourceID  string `json:"sourceId,omitempty"`
	SizeBytes int64  `json:"sizeBytes,omitempty"`
	Error     string `json:"error,omitempty"`
}

// SubmitText builds the channel-neutral kernel submit body from message text,
// reply context, and normalized attachments.
func SubmitText(text string, replyToText string, attachments []Attachment) string {
	text = strings.TrimSpace(text)
	if reply := strings.TrimSpace(replyToText); reply != "" {
		prefix := `[Replying to: "` + truncateRunes(reply, 500) + `"]`
		if text == "" {
			text = prefix
		} else {
			text = prefix + "\n\n" + text
		}
	}
	if len(attachments) == 0 {
		return text
	}

	attachmentLines := make([]string, 0, len(attachments))
	for _, att := range attachments {
		if line := att.submitLine(); line != "" {
			attachmentLines = append(attachmentLines, line)
		}
	}
	if len(attachmentLines) == 0 {
		return text
	}
	audioHintLines := audioTranscriptionHintLines(attachments)

	lines := make([]string, 0, len(attachmentLines)+len(audioHintLines)+5)
	if text != "" {
		lines = append(lines, text, "")
	}
	lines = append(lines, "Attachments:")
	lines = append(lines, attachmentLines...)
	if len(audioHintLines) > 0 {
		lines = append(lines, "", "Audio transcription:")
		lines = append(lines, audioHintLines...)
	}
	return strings.TrimSpace(strings.Join(lines, "\n"))
}

func truncateRunes(s string, limit int) string {
	if limit <= 0 {
		return ""
	}
	for i := range s {
		if limit == 0 {
			return s[:i]
		}
		limit--
	}
	return s
}

func (a Attachment) submitLine() string {
	kind := strings.TrimSpace(a.Kind)
	if kind == "" {
		kind = "attachment"
	}
	label := kind
	if fileName := strings.TrimSpace(a.FileName); fileName != "" {
		label += " " + fileName
	}

	target := strings.TrimSpace(a.URL)
	if target == "" {
		target = strings.TrimSpace(a.SourceID)
	}
	if target == "" {
		return ""
	}

	var meta []string
	if mediaType := strings.TrimSpace(a.MediaType); mediaType != "" {
		meta = append(meta, "mediaType="+mediaType)
	}
	if sourceID := strings.TrimSpace(a.SourceID); sourceID != "" {
		meta = append(meta, "sourceId="+sourceID)
	}
	if a.SizeBytes > 0 {
		meta = append(meta, "sizeBytes="+strconv.FormatInt(a.SizeBytes, 10))
	}
	if errText := strings.TrimSpace(a.Error); errText != "" {
		meta = append(meta, "error="+errText)
	}

	line := "- " + label + ": " + target
	if len(meta) > 0 {
		line += " (" + strings.Join(meta, ", ") + ")"
	}
	return line
}

func audioTranscriptionHintLines(attachments []Attachment) []string {
	lines := make([]string, 0, len(attachments))
	for _, att := range attachments {
		if line := att.audioTranscriptionHintLine(); line != "" {
			lines = append(lines, line)
		}
	}
	return lines
}

func (a Attachment) audioTranscriptionHintLine() string {
	if !a.isAudioAttachment() {
		return ""
	}
	audioPath := strings.TrimSpace(a.URL)
	if audioPath == "" || !isLocalAttachmentPath(audioPath) {
		return ""
	}

	var meta []string
	if kind := strings.TrimSpace(a.Kind); kind != "" {
		meta = append(meta, "kind="+kind)
	}
	if fileName := strings.TrimSpace(a.FileName); fileName != "" {
		meta = append(meta, "fileName="+fileName)
	}
	if mediaType := strings.TrimSpace(a.MediaType); mediaType != "" {
		meta = append(meta, "mediaType="+mediaType)
	}

	line := "- transcribe_audio audio_path=" + strconv.Quote(audioPath)
	if len(meta) > 0 {
		line += " (" + strings.Join(meta, ", ") + ")"
	}
	return line
}

func (a Attachment) isAudioAttachment() bool {
	switch strings.ToLower(strings.TrimSpace(a.Kind)) {
	case "audio", "voice", "voice_message", "voice_note":
		return true
	}
	return strings.HasPrefix(strings.ToLower(strings.TrimSpace(a.MediaType)), "audio/")
}

func isLocalAttachmentPath(target string) bool {
	target = strings.TrimSpace(target)
	if target == "" {
		return false
	}
	lower := strings.ToLower(target)
	if strings.Contains(lower, "://") || strings.HasPrefix(lower, "data:") || strings.HasPrefix(lower, "transcript:") {
		return false
	}
	if strings.HasPrefix(target, "/") || strings.HasPrefix(target, "./") || strings.HasPrefix(target, "../") || strings.HasPrefix(target, "~") {
		return true
	}
	if len(target) >= 3 && target[1] == ':' && (target[2] == '\\' || target[2] == '/') {
		return true
	}
	return strings.ContainsAny(target, `/\`)
}
