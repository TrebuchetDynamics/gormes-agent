package telegram

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	telegrammedia "github.com/TrebuchetDynamics/gormes-agent/internal/adapters/channels/telegram/media"
	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway"
)

const (
	telegramMaxDocumentBytes       = telegrammedia.MaxDocumentBytes
	telegramInlineTextDocumentSize = telegrammedia.InlineTextDocumentSize
	telegramDefaultMediaBatchDelay = 300 * time.Millisecond
)

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
	if inferred, ok := telegrammedia.VideoMediaTypeForExtension(ext); ok {
		kind = "video"
		if mediaType == "" {
			mediaType = inferred
		}
	} else if inferred, ok := telegrammedia.DocumentMediaTypeForExtension(ext); ok {
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
	if inferred, ok := telegrammedia.VideoMediaTypeForExtension(ext); ok && mediaType == "" {
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
	return telegrammedia.CleanMediaType(mediaType)
}

func telegramInferExtension(fileName, mediaType string) string {
	return telegrammedia.InferExtension(fileName, mediaType)
}

func telegramDisplayFileName(fileName, ext, fallbackBase string) string {
	return telegrammedia.DisplayFileName(fileName, ext, fallbackBase)
}

func telegramSafeFileName(fileName string) string {
	return telegrammedia.SafeFileName(fileName)
}

func telegramSafeToken(s string) string {
	return telegrammedia.SafeToken(s)
}

func telegramShouldInlineTextDocument(ext string, size int64) bool {
	return telegrammedia.ShouldInlineTextDocument(ext, size)
}

func telegramUnsupportedDocumentMarker(fileName, mediaType, reason string) string {
	return telegrammedia.UnsupportedDocumentMarker(fileName, mediaType, reason)
}

func telegramDocumentSizeMarker(kind, fileName, mediaType string, size int64) string {
	return telegrammedia.DocumentSizeMarker(kind, fileName, mediaType, size)
}

func telegramCachedAttachmentMarker(kind, fileName, mediaType string, size int64) string {
	return telegrammedia.CachedAttachmentMarker(kind, fileName, mediaType, size)
}

func telegramSupportedTypesList() string {
	return telegrammedia.SupportedTypesList()
}

func telegramLargestPhoto(photos []tgbotapi.PhotoSize) (tgbotapi.PhotoSize, bool) {
	return telegrammedia.LargestPhoto(photos)
}

func telegramPhotoScore(photo tgbotapi.PhotoSize) int {
	return telegrammedia.PhotoScore(photo)
}

func telegramPhotoExtension(filePath string) string {
	return telegrammedia.PhotoExtension(filePath)
}

func telegramPhotoMediaType(ext string) string {
	return telegrammedia.PhotoMediaType(ext)
}
