package telegram

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	telegrammedia "github.com/TrebuchetDynamics/gormes-agent/internal/adapters/channels/telegram/media"
	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway"
)

const StickerVisionPrompt = "Describe this sticker in 1-2 sentences. Focus on what it depicts -- character, action, emotion. Be concise and objective."

type StickerVisionRequest struct {
	ImagePath    string
	Prompt       string
	FileUniqueID string
	Emoji        string
	SetName      string
}

type StickerVisionAnalyzer func(context.Context, StickerVisionRequest) (string, error)

func (b *Bot) telegramStickerMarker(ctx context.Context, sticker *tgbotapi.Sticker) string {
	if sticker == nil {
		return ""
	}
	emoji := strings.TrimSpace(sticker.Emoji)
	setName := strings.TrimSpace(sticker.SetName)
	if sticker.IsAnimated {
		return gateway.BuildAnimatedStickerInjection(emoji)
	}
	fileUniqueID := strings.TrimSpace(sticker.FileUniqueID)
	if fileUniqueID != "" {
		if cached, ok, err := gateway.GetCachedStickerDescription(b.telegramStickerCachePath(), fileUniqueID); err == nil && ok {
			return gateway.BuildStickerInjection(cached.Description, telegrammedia.FirstNonEmpty(cached.Emoji, emoji), telegrammedia.FirstNonEmpty(cached.SetName, setName))
		}
	}
	if b.cfg.StickerVisionAnalyzer == nil {
		return b.telegramStickerFallback(emoji, setName)
	}
	sourceID := strings.TrimSpace(sticker.FileID)
	if sourceID == "" {
		return b.telegramStickerFallback(emoji, setName)
	}
	file, err := b.telegramGetFile(sourceID)
	if err != nil {
		return b.telegramStickerFallback(emoji, setName)
	}
	data, err := b.telegramDownloadFile(ctx, file.FilePath)
	if err != nil {
		return b.telegramStickerFallback(emoji, setName)
	}
	imagePath, err := b.cacheTelegramBytes("stickers", telegramStickerFileName(file.FilePath, fileUniqueID, sourceID), data)
	if err != nil {
		return b.telegramStickerFallback(emoji, setName)
	}
	description, err := b.cfg.StickerVisionAnalyzer(ctx, StickerVisionRequest{
		ImagePath:    imagePath,
		Prompt:       StickerVisionPrompt,
		FileUniqueID: fileUniqueID,
		Emoji:        emoji,
		SetName:      setName,
	})
	description = strings.TrimSpace(description)
	if err != nil || description == "" {
		return b.telegramStickerFallback(emoji, setName)
	}
	if fileUniqueID != "" {
		if err := gateway.CacheStickerDescription(b.telegramStickerCachePath(), fileUniqueID, description, emoji, setName, time.Now()); err != nil {
			return b.telegramStickerFallback(emoji, setName)
		}
	}
	return gateway.BuildStickerInjection(description, emoji, setName)
}

func telegramStickerFileName(filePath, fileUniqueID, fileID string) string {
	return telegrammedia.StickerFileName(filePath, fileUniqueID, fileID)
}

func (b *Bot) telegramStickerFallback(emoji, setName string) string {
	return gateway.BuildStickerInjection(telegramStickerFallbackDescription(emoji), emoji, setName)
}

func (b *Bot) telegramStickerCachePath() string {
	if path := strings.TrimSpace(b.cfg.StickerCachePath); path != "" {
		return path
	}
	cacheDir, err := os.UserCacheDir()
	if err != nil || strings.TrimSpace(cacheDir) == "" {
		cacheDir = os.TempDir()
	}
	return filepath.Join(cacheDir, "gormes", "telegram", "sticker_cache.json")
}

func telegramStickerFallbackDescription(emoji string) string {
	return telegrammedia.StickerFallbackDescription(emoji)
}
