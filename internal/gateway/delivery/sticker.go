package delivery

import (
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway/delivery/stickers"
)

type StickerDescription = stickers.StickerDescription

func GetCachedStickerDescription(path, fileUniqueID string) (StickerDescription, bool, error) {
	return stickers.GetCachedStickerDescription(path, fileUniqueID)
}

func CacheStickerDescription(path, fileUniqueID, description, emoji, setName string, now time.Time) error {
	return stickers.CacheStickerDescription(path, fileUniqueID, description, emoji, setName, now)
}

func BuildStickerInjection(description, emoji, setName string) string {
	return stickers.BuildStickerInjection(description, emoji, setName)
}

func BuildAnimatedStickerInjection(emoji string) string {
	return stickers.BuildAnimatedStickerInjection(emoji)
}
