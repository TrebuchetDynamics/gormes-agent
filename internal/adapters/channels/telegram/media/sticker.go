package media

import (
	"path/filepath"
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/adapters/channels/internal/channelutil"
)

func StickerFileName(filePath, fileUniqueID, fileID string) string {
	ext := strings.ToLower(filepath.Ext(strings.TrimSpace(filePath)))
	if ext == "" {
		ext = ".webp"
	}
	base := FirstNonEmpty(fileUniqueID, fileID, "sticker")
	return "sticker_" + SafeToken(base) + ext
}

func StickerFallbackDescription(emoji string) string {
	emoji = strings.TrimSpace(emoji)
	if emoji != "" {
		return "a sticker with emoji " + emoji
	}
	return "a sticker"
}

func FirstNonEmpty(values ...string) string { return channelutil.FirstNonEmpty(values...) }

func SafeToken(s string) string { return channelutil.SafeTokenDefault(s, "telegram") }
