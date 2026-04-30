package gateway

import (
	"path/filepath"
	"regexp"
	"strings"
)

type MediaDeliveryEvidence string

const (
	MediaDeliveryEvidenceExtracted MediaDeliveryEvidence = "media_extracted"
	MediaDeliveryEvidenceIgnored   MediaDeliveryEvidence = "media_ignored"
)

type MediaDeliveryEvidenceRecord struct {
	Code   MediaDeliveryEvidence
	Target string
	Detail string
}

type MediaDeliveryContent struct {
	Text     string
	Media    []OutboundMedia
	Evidence []MediaDeliveryEvidenceRecord
}

var mediaDeliveryTagRE = regexp.MustCompile(`(?s)(?:\[\[audio_as_voice\]\]\s*)?(?:\[MEDIA:([^\]\s]+)\]|MEDIA:([^\s\]]+))`)
var mediaDeliveryBlankGapRE = regexp.MustCompile(`\n{2,}`)

// PrepareMediaDeliveryContent extracts Hermes MEDIA tags from final assistant
// text so channels can deliver files natively without leaking tag syntax to
// the operator-visible message.
func PrepareMediaDeliveryContent(finalText string) MediaDeliveryContent {
	var out MediaDeliveryContent
	cleaned := mediaDeliveryTagRE.ReplaceAllStringFunc(finalText, func(tag string) string {
		matches := mediaDeliveryTagRE.FindStringSubmatch(tag)
		if len(matches) != 3 {
			return tag
		}
		rawPath := firstNonEmptyString(matches[1], matches[2])
		mediaPath, ok := cleanOutboundMediaPath(rawPath)
		if !ok {
			out.Evidence = append(out.Evidence, MediaDeliveryEvidenceRecord{
				Code:   MediaDeliveryEvidenceIgnored,
				Target: "[redacted]",
				Detail: "unsafe media path redacted",
			})
			return "[MEDIA:redacted]"
		}
		out.Media = append(out.Media, OutboundMedia{
			Path:    mediaPath,
			AsVoice: strings.Contains(tag, "[[audio_as_voice]]"),
		})
		out.Evidence = append(out.Evidence, MediaDeliveryEvidenceRecord{
			Code:   MediaDeliveryEvidenceExtracted,
			Target: mediaPath,
		})
		return ""
	})
	if len(out.Media) > 0 {
		cleaned = mediaDeliveryBlankGapRE.ReplaceAllString(cleaned, "\n")
	}
	out.Text = trimMediaDeliveryText(cleaned)
	return out
}

func cleanOutboundMediaPath(raw string) (string, bool) {
	value := strings.TrimSpace(raw)
	if value == "" || strings.ContainsRune(value, 0) {
		return "", false
	}
	value = filepath.Clean(value)
	if value == "." || value == ".." {
		return "", false
	}
	if !supportedOutboundMediaExt(filepath.Ext(value)) {
		return "", false
	}
	if !filepath.IsAbs(value) {
		for _, segment := range strings.Split(filepath.ToSlash(value), "/") {
			if segment == ".." {
				return "", false
			}
		}
	}
	return value, true
}

func supportedOutboundMediaExt(ext string) bool {
	switch strings.ToLower(strings.TrimSpace(ext)) {
	case ".mp3", ".ogg", ".opus", ".wav", ".m4a", ".aac", ".flac":
		return true
	default:
		return false
	}
}

func trimMediaDeliveryText(text string) string {
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		lines[i] = strings.TrimRight(line, " \t")
	}
	return strings.TrimSpace(strings.Join(lines, "\n"))
}
