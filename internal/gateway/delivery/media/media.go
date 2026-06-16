package media

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/redaction"
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
	var cleaned strings.Builder
	last := 0
	for _, loc := range mediaTagRE.FindAllStringIndex(finalText, -1) {
		start, end := loc[0], loc[1]
		if start < last {
			continue
		}
		if !mediaTagBoundaryOK(finalText, start) {
			continue
		}
		cleaned.WriteString(finalText[last:start])
		tag := finalText[start:end]
		candidate, ok := parseMediaTag(tag)
		if !ok {
			cleaned.WriteString(tag)
			last = end
			continue
		}
		mediaPath, suffix, ok := cleanMediaTagPath(candidate.RawPath)
		if !ok {
			out.Evidence = append(out.Evidence, MediaEvidenceRecord{
				Code:   MediaEvidenceIgnored,
				Target: "[redacted]",
				Detail: "unsafe media path redacted",
			})
			cleaned.WriteString("[MEDIA:redacted]")
			last = end
			continue
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
		cleaned.WriteString(suffix)
		last = end
	}
	cleaned.WriteString(finalText[last:])
	text := cleaned.String()
	if len(out.Media) > 0 {
		text = mediaBlankGapRE.ReplaceAllString(text, "\n")
	}
	out.Text = trimMediaText(text)
	return out
}

func mediaTagBoundaryOK(text string, start int) bool {
	if start <= 0 || start > len(text) {
		return start == 0
	}
	prev, _ := utf8.DecodeLastRuneInString(text[:start])
	return !(unicode.IsLetter(prev) || unicode.IsNumber(prev) || prev == '_')
}

type mediaTagCandidate struct {
	RawPath string
	AsVoice bool
}

func cleanMediaTagPath(raw string) (path, suffix string, ok bool) {
	if path, ok := CleanMediaPath(raw); ok {
		return path, "", true
	}
	trimmed := strings.TrimRight(raw, ".,!?;:)}")
	if trimmed == raw || strings.TrimSpace(trimmed) == "" {
		return "", "", false
	}
	if path, ok := CleanMediaPath(trimmed); ok {
		return path, raw[len(trimmed):], true
	}
	return "", "", false
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
	if value == "" || strings.ContainsRune(value, 0) || hasMediaPathControl(value) || hasMediaPathScheme(value) {
		return "", false
	}
	value = strings.ReplaceAll(value, "\\", "/")
	if redaction.CheckSensitivePath(value).Blocked {
		return "", false
	}
	for _, segment := range strings.Split(value, "/") {
		if segment == ".." {
			return "", false
		}
		if sensitiveMediaPathComponent(segment) {
			return "", false
		}
	}
	value = filepath.Clean(value)
	if value == "." || value == ".." {
		return "", false
	}
	if !SupportedMediaExt(filepath.Ext(value)) {
		return "", false
	}
	info, err := os.Lstat(value)
	if err != nil || !info.Mode().IsRegular() {
		return "", false
	}
	return value, true
}

func sensitiveMediaPathComponent(segment string) bool {
	switch strings.ToLower(strings.TrimSpace(segment)) {
	case ".ssh", ".aws", ".gcloud", ".azure", ".kube":
		return true
	default:
		return false
	}
}

func hasMediaPathControl(value string) bool {
	for _, r := range value {
		if unicode.IsControl(r) || hiddenMediaPathFormattingRune(r) {
			return true
		}
	}
	return false
}

func hiddenMediaPathFormattingRune(r rune) bool {
	switch {
	case r >= 0x200b && r <= 0x200f:
		return true
	case r >= 0x2028 && r <= 0x202e:
		return true
	case r >= 0x2060 && r <= 0x2069:
		return true
	case r == 0xfeff || r == 0xfffc:
		return true
	case r >= 0xfff9 && r <= 0xfffb:
		return true
	default:
		return false
	}
}

func hasMediaPathScheme(value string) bool {
	colon := strings.Index(value, ":")
	if colon < 0 {
		return false
	}
	if colon == 1 && len(value) >= 3 && ((value[0] >= 'A' && value[0] <= 'Z') || (value[0] >= 'a' && value[0] <= 'z')) && (value[2] == '/' || value[2] == '\\') {
		return false
	}
	sep := strings.IndexAny(value, `/\\`)
	return sep < 0 || colon < sep
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
