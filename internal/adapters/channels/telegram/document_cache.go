package telegram

import (
	"context"
	"fmt"
	"mime"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway"
)

const (
	telegramMaxDocumentBytes       int64 = 20 * 1024 * 1024
	telegramInlineTextDocumentSize int64 = 100 * 1024
	telegramDefaultMediaBatchDelay       = 300 * time.Millisecond
)

var telegramSupportedDocumentExtensions = map[string]string{
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

var telegramSupportedVideoExtensions = map[string]string{
	".m4v":  "video/mp4",
	".mov":  "video/quicktime",
	".mp4":  "video/mp4",
	".webm": "video/webm",
}

var telegramMIMEExtensionFallbacks = map[string]string{
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

type telegramPhotoBatchEntry struct {
	event gateway.InboundEvent
	timer *time.Timer
	seq   uint64
}

func (b *Bot) telegramDocumentAttachment(ctx context.Context, doc *tgbotapi.Document) (string, string, *gateway.Attachment) {
	if doc == nil {
		return "", "", nil
	}
	sourceID := strings.TrimSpace(doc.FileID)
	fileName := strings.TrimSpace(doc.FileName)
	mediaType := cleanTelegramMediaType(doc.MimeType)
	ext := telegramInferExtension(fileName, mediaType)
	if ext == "" {
		return "", telegramUnsupportedDocumentMarker(fileName, mediaType, "missing filename or MIME type"), nil
	}

	kind := "document"
	if inferred, ok := telegramSupportedVideoExtensions[ext]; ok {
		kind = "video"
		if mediaType == "" {
			mediaType = inferred
		}
	} else if inferred, ok := telegramSupportedDocumentExtensions[ext]; ok {
		if mediaType == "" {
			mediaType = inferred
		}
	} else {
		return "", telegramUnsupportedDocumentMarker(fileName, mediaType, fmt.Sprintf("unsupported extension %q", ext)), nil
	}

	if doc.FileSize <= 0 {
		return "", telegramDocumentSizeMarker(kind, fileName, mediaType, 0), nil
	}
	if int64(doc.FileSize) > telegramMaxDocumentBytes {
		return "", telegramDocumentSizeMarker(kind, fileName, mediaType, int64(doc.FileSize)), nil
	}

	displayName := telegramDisplayFileName(fileName, ext, "document")
	marker, attachment := b.cacheTelegramFileID(ctx, kind, sourceID, displayName, mediaType, int64(doc.FileSize))
	if attachment == nil {
		return "", marker, nil
	}
	if kind == "video" {
		return "", marker, attachment
	}

	var prefix string
	if telegramShouldInlineTextDocument(ext, attachment.SizeBytes) {
		if data, err := os.ReadFile(attachment.URL); err == nil && utf8.Valid(data) {
			prefix = fmt.Sprintf("[Content of %s]:\n%s", displayName, string(data))
		}
	}
	return prefix, marker, attachment
}

func (b *Bot) telegramVideoMessageAttachment(ctx context.Context, video *tgbotapi.Video) (string, *gateway.Attachment) {
	if video == nil {
		return "", nil
	}
	sourceID := strings.TrimSpace(video.FileID)
	mediaType := cleanTelegramMediaType(video.MimeType)
	ext := telegramInferExtension(video.FileName, mediaType)
	if ext == "" {
		ext = ".mp4"
	}
	if inferred, ok := telegramSupportedVideoExtensions[ext]; ok && mediaType == "" {
		mediaType = inferred
	}
	if mediaType == "" {
		mediaType = "video/mp4"
	}
	if video.FileSize <= 0 {
		return telegramDocumentSizeMarker("video", video.FileName, mediaType, 0), nil
	}
	if int64(video.FileSize) > telegramMaxDocumentBytes {
		return telegramDocumentSizeMarker("video", video.FileName, mediaType, int64(video.FileSize)), nil
	}
	fileName := telegramDisplayFileName(video.FileName, ext, "video")
	return b.cacheTelegramFileID(ctx, "video", sourceID, fileName, mediaType, int64(video.FileSize))
}

func (b *Bot) telegramPhotoMessageAttachment(ctx context.Context, photos []tgbotapi.PhotoSize) (string, *gateway.Attachment) {
	photo, ok := telegramLargestPhoto(photos)
	if !ok {
		return "", nil
	}
	sourceID := strings.TrimSpace(photo.FileID)
	if sourceID == "" {
		return "[Telegram photo unavailable: missing file_id]", nil
	}
	file, err := b.telegramGetFile(sourceID)
	if err != nil {
		return "[Telegram photo unavailable: getFile failed]", nil
	}
	data, err := b.telegramDownloadFile(ctx, file.FilePath)
	if err != nil {
		return "[Telegram photo unavailable: download failed]", nil
	}
	ext := telegramPhotoExtension(file.FilePath)
	fileName := telegramDisplayFileName(filepath.Base(file.FilePath), ext, "photo")
	if fileName == "photo"+ext {
		fileName = "photo_" + telegramSafeToken(sourceID) + ext
	}
	path, err := b.cacheTelegramBytes("photos", fileName, data)
	if err != nil {
		return "[Telegram photo unavailable: cache write failed]", nil
	}
	return "", &gateway.Attachment{
		Kind:      "photo",
		URL:       path,
		MediaType: telegramPhotoMediaType(ext),
		FileName:  fileName,
		SourceID:  sourceID,
		SizeBytes: int64(len(data)),
	}
}

func (b *Bot) cacheTelegramFileID(ctx context.Context, kind, sourceID, fileName, mediaType string, declaredSize int64) (string, *gateway.Attachment) {
	if sourceID == "" {
		return fmt.Sprintf("[Telegram %s unavailable: missing file_id]", kind), nil
	}
	file, err := b.telegramGetFile(sourceID)
	if err != nil {
		return fmt.Sprintf("[Telegram %s unavailable: getFile failed]", kind), nil
	}
	data, err := b.telegramDownloadFile(ctx, file.FilePath)
	if err != nil {
		return fmt.Sprintf("[Telegram %s unavailable: download failed]", kind), nil
	}
	if int64(len(data)) > telegramMaxDocumentBytes {
		return telegramDocumentSizeMarker(kind, fileName, mediaType, int64(len(data))), nil
	}
	if mediaType == "" {
		mediaType = "application/octet-stream"
	}
	dir := kind + "s"
	if kind == "document" {
		dir = "documents"
	}
	path, err := b.cacheTelegramBytes(dir, fileName, data)
	if err != nil {
		return fmt.Sprintf("[Telegram %s unavailable: cache write failed]", kind), nil
	}
	size := int64(len(data))
	attachment := &gateway.Attachment{
		Kind:      kind,
		URL:       path,
		MediaType: mediaType,
		FileName:  fileName,
		SourceID:  sourceID,
		SizeBytes: size,
	}
	return telegramCachedAttachmentMarker(kind, fileName, mediaType, size), attachment
}

func (b *Bot) telegramGetFile(fileID string) (tgbotapi.File, error) {
	if b.client == nil {
		return tgbotapi.File{}, fmt.Errorf("telegram client unavailable")
	}
	return b.client.GetFile(tgbotapi.FileConfig{FileID: fileID})
}

func (b *Bot) telegramDownloadFile(ctx context.Context, filePath string) ([]byte, error) {
	if b.client == nil {
		return nil, fmt.Errorf("telegram client unavailable")
	}
	return b.client.DownloadFile(ctx, filePath)
}

func (b *Bot) cacheTelegramBytes(category, fileName string, data []byte) (string, error) {
	root := strings.TrimSpace(b.cfg.AttachmentCacheDir)
	if root == "" {
		cacheDir, err := os.UserCacheDir()
		if err != nil || strings.TrimSpace(cacheDir) == "" {
			cacheDir = os.TempDir()
		}
		root = filepath.Join(cacheDir, "gormes", "telegram")
	}
	dir := filepath.Join(root, telegramSafeToken(category))
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", err
	}
	path := filepath.Join(dir, fmt.Sprintf("%d_%s", time.Now().UnixNano(), telegramSafeFileName(fileName)))
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return "", err
	}
	return filepath.Clean(path), nil
}

func cleanTelegramMediaType(mediaType string) string {
	if mediaType = strings.TrimSpace(mediaType); mediaType == "" {
		return ""
	}
	if semi := strings.Index(mediaType, ";"); semi >= 0 {
		mediaType = mediaType[:semi]
	}
	return strings.ToLower(strings.TrimSpace(mediaType))
}

func telegramInferExtension(fileName, mediaType string) string {
	if ext := strings.ToLower(filepath.Ext(strings.TrimSpace(fileName))); ext != "" {
		return ext
	}
	mediaType = cleanTelegramMediaType(mediaType)
	if ext := telegramMIMEExtensionFallbacks[mediaType]; ext != "" {
		return ext
	}
	extensions, _ := mime.ExtensionsByType(mediaType)
	for _, ext := range extensions {
		ext = strings.ToLower(ext)
		if _, ok := telegramSupportedDocumentExtensions[ext]; ok {
			return ext
		}
		if _, ok := telegramSupportedVideoExtensions[ext]; ok {
			return ext
		}
	}
	return ""
}

func telegramDisplayFileName(fileName, ext, fallbackBase string) string {
	fileName = telegramSafeFileName(fileName)
	if fileName != "" && fileName != fallbackBase {
		return fileName
	}
	ext = strings.ToLower(strings.TrimSpace(ext))
	if ext == "" {
		return fallbackBase
	}
	return fallbackBase + ext
}

func telegramSafeFileName(fileName string) string {
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

func telegramSafeToken(s string) string {
	s = strings.TrimSpace(s)
	var out strings.Builder
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z':
			out.WriteRune(r)
		case r >= 'A' && r <= 'Z':
			out.WriteRune(r)
		case r >= '0' && r <= '9':
			out.WriteRune(r)
		case r == '-' || r == '_' || r == '.':
			out.WriteRune(r)
		default:
			out.WriteByte('_')
		}
	}
	cleaned := strings.Trim(out.String(), "._-")
	if cleaned == "" {
		return "telegram"
	}
	if len(cleaned) > 64 {
		return cleaned[:64]
	}
	return cleaned
}

func telegramShouldInlineTextDocument(ext string, size int64) bool {
	if size <= 0 || size > telegramInlineTextDocumentSize {
		return false
	}
	switch strings.ToLower(ext) {
	case ".md", ".txt":
		return true
	default:
		return false
	}
}

func telegramUnsupportedDocumentMarker(fileName, mediaType, reason string) string {
	var parts []string
	if fileName = strings.TrimSpace(fileName); fileName != "" {
		parts = append(parts, "file="+telegramSafeFileName(fileName))
	}
	if mediaType = cleanTelegramMediaType(mediaType); mediaType != "" {
		parts = append(parts, "mime="+mediaType)
	}
	if reason = strings.TrimSpace(reason); reason != "" {
		parts = append(parts, reason)
	}
	parts = append(parts, "supported types: "+telegramSupportedTypesList())
	return "[Unsupported Telegram document type: " + strings.Join(parts, "; ") + "]"
}

func telegramDocumentSizeMarker(kind, fileName, mediaType string, size int64) string {
	var parts []string
	if fileName = telegramSafeFileName(fileName); fileName != "" {
		parts = append(parts, "file="+fileName)
	}
	if mediaType = cleanTelegramMediaType(mediaType); mediaType != "" {
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

func telegramCachedAttachmentMarker(kind, fileName, mediaType string, size int64) string {
	var parts []string
	if fileName = telegramSafeFileName(fileName); fileName != "" {
		parts = append(parts, "file="+fileName)
	}
	if mediaType = cleanTelegramMediaType(mediaType); mediaType != "" {
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

func telegramSupportedTypesList() string {
	extensions := make([]string, 0, len(telegramSupportedDocumentExtensions)+len(telegramSupportedVideoExtensions))
	for ext := range telegramSupportedDocumentExtensions {
		extensions = append(extensions, ext)
	}
	for ext := range telegramSupportedVideoExtensions {
		extensions = append(extensions, ext)
	}
	sort.Strings(extensions)
	return strings.Join(extensions, ", ")
}

func telegramLargestPhoto(photos []tgbotapi.PhotoSize) (tgbotapi.PhotoSize, bool) {
	if len(photos) == 0 {
		return tgbotapi.PhotoSize{}, false
	}
	best := photos[0]
	bestScore := telegramPhotoScore(best)
	for _, photo := range photos[1:] {
		score := telegramPhotoScore(photo)
		if score > bestScore {
			best = photo
			bestScore = score
		}
	}
	return best, true
}

func telegramPhotoScore(photo tgbotapi.PhotoSize) int {
	score := photo.Width * photo.Height
	if score == 0 {
		score = photo.FileSize
	}
	return score
}

func telegramPhotoExtension(filePath string) string {
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

func telegramPhotoMediaType(ext string) string {
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
