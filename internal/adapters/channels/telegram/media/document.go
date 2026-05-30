package media

import (
	"fmt"
	"mime"
	"path/filepath"
	"sort"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

const (
	MaxDocumentBytes       int64 = 20 * 1024 * 1024
	InlineTextDocumentSize int64 = 100 * 1024
)

var supportedDocumentExtensions = map[string]string{
	".cfg":  "text/plain",
	".csv":  "text/csv",
	".docx": "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
	".ini":  "text/plain",
	".json": "application/json",
	".log":  "text/plain",
	".md":   "text/markdown",
	".pdf":  "application/pdf",
	".pptx": "application/vnd.openxmlformats-officedocument.presentationml.presentation",
	".toml": "application/toml",
	".txt":  "text/plain",
	".xlsx": "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
	".xml":  "application/xml",
	".yaml": "application/yaml",
	".yml":  "application/yaml",
	".zip":  "application/zip",
}

var supportedVideoExtensions = map[string]string{
	".m4v":  "video/mp4",
	".mov":  "video/quicktime",
	".mp4":  "video/mp4",
	".webm": "video/webm",
}

var mimeExtensionFallbacks = map[string]string{
	"application/json":   ".json",
	"application/pdf":    ".pdf",
	"application/toml":   ".toml",
	"application/xml":    ".xml",
	"application/yaml":   ".yaml",
	"application/zip":    ".zip",
	"text/csv":           ".csv",
	"text/markdown":      ".md",
	"text/plain":         ".txt",
	"video/mp4":          ".mp4",
	"video/quicktime":    ".mov",
	"video/webm":         ".webm",
	"x-application/yaml": ".yaml",
}

func DocumentMediaTypeForExtension(ext string) (string, bool) {
	mediaType, ok := supportedDocumentExtensions[strings.ToLower(strings.TrimSpace(ext))]
	return mediaType, ok
}

func VideoMediaTypeForExtension(ext string) (string, bool) {
	mediaType, ok := supportedVideoExtensions[strings.ToLower(strings.TrimSpace(ext))]
	return mediaType, ok
}

func CleanMediaType(mediaType string) string {
	if mediaType = strings.TrimSpace(mediaType); mediaType == "" {
		return ""
	}
	if semi := strings.Index(mediaType, ";"); semi >= 0 {
		mediaType = mediaType[:semi]
	}
	return strings.ToLower(strings.TrimSpace(mediaType))
}

func InferExtension(fileName, mediaType string) string {
	if ext := strings.ToLower(filepath.Ext(strings.TrimSpace(fileName))); ext != "" {
		return ext
	}
	mediaType = CleanMediaType(mediaType)
	if ext := mimeExtensionFallbacks[mediaType]; ext != "" {
		return ext
	}
	extensions, _ := mime.ExtensionsByType(mediaType)
	for _, ext := range extensions {
		ext = strings.ToLower(ext)
		if _, ok := supportedDocumentExtensions[ext]; ok {
			return ext
		}
		if _, ok := supportedVideoExtensions[ext]; ok {
			return ext
		}
	}
	return ""
}

func DisplayFileName(fileName, ext, fallbackBase string) string {
	fileName = SafeFileName(fileName)
	if fileName != "" && fileName != fallbackBase {
		return fileName
	}
	ext = strings.ToLower(strings.TrimSpace(ext))
	if ext == "" {
		return fallbackBase
	}
	return fallbackBase + ext
}

func SafeFileName(fileName string) string {
	fileName = filepath.Base(strings.TrimSpace(fileName))
	var out strings.Builder
	for _, r := range fileName {
		switch {
		case r == 0 || r < 32 || r == 127:
			out.WriteByte('_')
		case r == '/' || r == '\\':
			out.WriteByte('_')
		default:
			out.WriteRune(r)
		}
	}
	cleaned := strings.Trim(out.String(), " .")
	if cleaned == "" || cleaned == "." || cleaned == ".." {
		return ""
	}
	if len(cleaned) <= 160 {
		return cleaned
	}
	ext := filepath.Ext(cleaned)
	stem := strings.TrimSuffix(cleaned, ext)
	if len(ext) > 32 {
		ext = ""
	}
	if len(stem) > 128 {
		stem = stem[:128]
	}
	return stem + ext
}

func ShouldInlineTextDocument(ext string, size int64) bool {
	if size <= 0 || size > InlineTextDocumentSize {
		return false
	}
	switch strings.ToLower(ext) {
	case ".md", ".txt":
		return true
	default:
		return false
	}
}

func UnsupportedDocumentMarker(fileName, mediaType, reason string) string {
	var parts []string
	if fileName = strings.TrimSpace(fileName); fileName != "" {
		parts = append(parts, "file="+SafeFileName(fileName))
	}
	if mediaType = CleanMediaType(mediaType); mediaType != "" {
		parts = append(parts, "mime="+mediaType)
	}
	if reason = strings.TrimSpace(reason); reason != "" {
		parts = append(parts, reason)
	}
	parts = append(parts, "supported types: "+SupportedTypesList())
	return "[Unsupported Telegram document type: " + strings.Join(parts, "; ") + "]"
}

func DocumentSizeMarker(kind, fileName, mediaType string, size int64) string {
	var parts []string
	if fileName = SafeFileName(fileName); fileName != "" {
		parts = append(parts, "file="+fileName)
	}
	if mediaType = CleanMediaType(mediaType); mediaType != "" {
		parts = append(parts, "mime="+mediaType)
	}
	if size <= 0 {
		parts = append(parts, "size could not be verified")
	} else {
		parts = append(parts, fmt.Sprintf("too large: size=%d bytes", size))
	}
	parts = append(parts, "maximum=20 MB")
	return fmt.Sprintf("[Telegram %s rejected: %s]", kind, strings.Join(parts, "; "))
}

func CachedAttachmentMarker(kind, fileName, mediaType string, size int64) string {
	var parts []string
	if fileName = SafeFileName(fileName); fileName != "" {
		parts = append(parts, "file="+fileName)
	}
	if mediaType = CleanMediaType(mediaType); mediaType != "" {
		parts = append(parts, "mime="+mediaType)
	}
	if size > 0 {
		parts = append(parts, fmt.Sprintf("size=%d bytes", size))
	}
	if len(parts) == 0 {
		return fmt.Sprintf("[Telegram %s message attached]", kind)
	}
	return fmt.Sprintf("[Telegram %s message attached: %s]", kind, strings.Join(parts, ", "))
}

func SupportedTypesList() string {
	extensions := make([]string, 0, len(supportedDocumentExtensions)+len(supportedVideoExtensions))
	for ext := range supportedDocumentExtensions {
		extensions = append(extensions, ext)
	}
	for ext := range supportedVideoExtensions {
		extensions = append(extensions, ext)
	}
	sort.Strings(extensions)
	return strings.Join(extensions, ", ")
}

func LargestPhoto(photos []tgbotapi.PhotoSize) (tgbotapi.PhotoSize, bool) {
	if len(photos) == 0 {
		return tgbotapi.PhotoSize{}, false
	}
	best := photos[0]
	bestScore := PhotoScore(best)
	for _, photo := range photos[1:] {
		score := PhotoScore(photo)
		if score > bestScore {
			best = photo
			bestScore = score
		}
	}
	return best, true
}

func PhotoScore(photo tgbotapi.PhotoSize) int {
	score := photo.Width * photo.Height
	if score == 0 {
		score = photo.FileSize
	}
	return score
}

func PhotoExtension(filePath string) string {
	switch strings.ToLower(filepath.Ext(filePath)) {
	case ".gif":
		return ".gif"
	case ".jpeg":
		return ".jpeg"
	case ".jpg":
		return ".jpg"
	case ".png":
		return ".png"
	case ".webp":
		return ".webp"
	default:
		return ".jpg"
	}
}

func PhotoMediaType(ext string) string {
	switch strings.ToLower(ext) {
	case ".gif":
		return "image/gif"
	case ".jpeg", ".jpg":
		return "image/jpeg"
	case ".png":
		return "image/png"
	case ".webp":
		return "image/webp"
	default:
		return "image/jpeg"
	}
}
