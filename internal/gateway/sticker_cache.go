package gateway

import (
	"time"

	gatewaydelivery "github.com/TrebuchetDynamics/gormes-agent/internal/gateway/delivery"
)

type StickerDescription = gatewaydelivery.StickerDescription

func GetCachedStickerDescription(path, fileUniqueID string) (StickerDescription, bool, error) {
	return gatewaydelivery.GetCachedStickerDescription(path, fileUniqueID)
}

func CacheStickerDescription(path, fileUniqueID, description, emoji, setName string, now time.Time) error {
	return gatewaydelivery.CacheStickerDescription(path, fileUniqueID, description, emoji, setName, now)
}

func BuildStickerInjection(description, emoji, setName string) string {
	return gatewaydelivery.BuildStickerInjection(description, emoji, setName)
}

func BuildAnimatedStickerInjection(emoji string) string {
	return gatewaydelivery.BuildAnimatedStickerInjection(emoji)
}
