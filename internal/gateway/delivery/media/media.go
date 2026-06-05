package media

import (
	"path/filepath"
	"regexp"
	"strings"
)

type MediaKind string

const (
	MediaKindAudio    MediaKind = "audio"
	MediaKindDocument MediaKind = "document"
	MediaKindImage    MediaKind = "image"
	MediaKindVideo    MediaKind = "video"
)

type Media struct {
	Path     string
	AsVoice  bool
	Kind     MediaKind
	ThreadID string
}

type MediaEvidence string

const (
	MediaEvidenceExtracted MediaEvidence = "media_extracted"
	MediaEvidenceIgnored   MediaEvidence = "media_ignored"
)

type MediaEvidenceRecord struct {
	Code   MediaEvidence
	Target string
	Detail string
}

type MediaContent struct {
	Text     string
	Media    []Media
	Evidence []MediaEvidenceRecord
}

var mediaTagRE = regexp.MustCompile(`(?:\[\[audio_as_voice\]\]\s*)?(?:\[MEDIA:([^\]\r\n]+)\]|MEDIA:([^\s\]]+))`)
var mediaBlankGapRE = regexp.MustCompile(`\n{2,}`)

// PrepareMediaContent extracts Hermes MEDIA tags from final assistant text so
// channels can deliver files natively without leaking tag syntax to the
// operator-visible message.
func PrepareMediaContent(finalText string) MediaContent {
	var out MediaContent
	cleaned := mediaTagRE.ReplaceAllStringFunc(finalText, func(tag string) string {
		candidate, ok := parseMediaTag(tag)
		if !ok {
			return tag
		}
		mediaPath, ok := CleanMediaPath(candidate.RawPath)
		if !ok {
			out.Evidence = append(out.Evidence, MediaEvidenceRecord{
				Code:   MediaEvidenceIgnored,
				Target: "[redacted]",
				Detail: "unsafe media path redacted",
			})
			return "[MEDIA:redacted]"
		}
		out.Media = append(out.Media, Media{
			Path:    mediaPath,
			AsVoice: candidate.AsVoice,
			Kind:    MediaKindForPath(mediaPath),
		})
		out.Evidence = append(out.Evidence, MediaEvidenceRecord{
			Code:   MediaEvidenceExtracted,
			Target: mediaPath,
		})
		return ""
	})
	if len(out.Media) > 0 {
		cleaned = mediaBlankGapRE.ReplaceAllString(cleaned, "\n")
	}
	out.Text = trimMediaText(cleaned)
	return out
}

type mediaTagCandidate struct {
	RawPath string
	AsVoice bool
}

func parseMediaTag(tag string) (mediaTagCandidate, bool) {
	matches := mediaTagRE.FindStringSubmatch(tag)
	if len(matches) != 3 {
		return mediaTagCandidate{}, false
	}
	rawPath := firstNonEmpty(matches[1], matches[2])
	if strings.TrimSpace(rawPath) == "" {
		return mediaTagCandidate{}, false
	}
	return mediaTagCandidate{RawPath: rawPath, AsVoice: strings.Contains(tag, "[[audio_as_voice]]")}, true
}

func CleanMediaPath(raw string) (string, bool) {
	value := strings.TrimSpace(raw)
	if value == "" || strings.ContainsRune(value, 0) {
		return "", false
	}
	value = filepath.Clean(value)
	if value == "." || value == ".." {
		return "", false
	}
	if !SupportedMediaExt(filepath.Ext(value)) {
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

func SupportedMediaExt(ext string) bool {
	return mediaKindForExt(ext) != ""
}

// ClassifyMedia returns the explicit media kind or infers it from the local
// file extension for older call sites that only populated Path.
func ClassifyMedia(media Media) MediaKind {
	if media.Kind != "" {
		return media.Kind
	}
	return MediaKindForPath(media.Path)
}

func MediaKindForPath(path string) MediaKind {
	return mediaKindForExt(filepath.Ext(path))
}

func mediaKindForExt(ext string) MediaKind {
	switch strings.ToLower(strings.TrimSpace(ext)) {
	case ".mp3", ".ogg", ".opus", ".wav", ".m4a", ".aac", ".flac":
		return MediaKindAudio
	case ".png", ".jpg", ".jpeg", ".webp":
		return MediaKindImage
	case ".mp4":
		return MediaKindVideo
	case ".pdf", ".csv", ".txt", ".zip":
		return MediaKindDocument
	default:
		return ""
	}
}

func trimMediaText(text string) string {
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		lines[i] = strings.TrimRight(line, " \t")
	}
	return strings.TrimSpace(strings.Join(lines, "\n"))
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
