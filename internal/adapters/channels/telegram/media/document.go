package media

import (
	"fmt"
	"mime"
	"path/filepath"
	"sort"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/TrebuchetDynamics/gormes-agent/internal/adapters/channels/internal/channelutil"
)

const (
	MaxDocumentBytes       int64 = 20 * 1024 * 1024
	InlineTextDocumentSize int64 = 100 * 1024
)

var supportedVideoExtensions = map[string]string{
	".m4v":  "video/mp4",
	".mov":  "video/quicktime",
	".mp4":  "video/mp4",
	".webm": "video/webm",
}

var videoMIMEExtensionFallbacks = map[string]string{
	"video/mp4":       ".mp4",
	"video/quicktime": ".mov",
	"video/webm":      ".webm",
}

func DocumentMediaTypeForExtension(ext string) (string, bool) {
	return channelutil.DocumentMediaTypeForExtension(ext)
}

func VideoMediaTypeForExtension(ext string) (string, bool) {
	mediaType, ok := supportedVideoExtensions[strings.ToLower(strings.TrimSpace(ext))]
	return mediaType, ok
}

func CleanMediaType(mediaType string) string { return channelutil.CleanMediaType(mediaType) }

func InferExtension(fileName, mediaType string) string {
	if ext := strings.ToLower(filepath.Ext(strings.TrimSpace(fileName))); ext != "" {
		return ext
	}
	mediaType = CleanMediaType(mediaType)
	if ext := channelutil.MIMEExtensionFallback(mediaType); ext != "" {
		return ext
	}
	if ext := videoMIMEExtensionFallbacks[mediaType]; ext != "" {
		return ext
	}
	extensions, _ := mime.ExtensionsByType(mediaType)
	for _, ext := range extensions {
		ext = strings.ToLower(ext)
		if _, ok := channelutil.DocumentMediaTypeForExtension(ext); ok {
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

func SafeFileName(fileName string) string { return channelutil.SafeFileName(fileName) }

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
	extensions := channelutil.DocumentExtensions()
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
	if ext := strings.ToLower(filepath.Ext(filePath)); channelutil.ImageExtensionSupported(ext) {
		return ext
	}
	return ".jpg"
}

func PhotoMediaType(ext string) string { return channelutil.ImageMediaTypeForExtension(ext) }
