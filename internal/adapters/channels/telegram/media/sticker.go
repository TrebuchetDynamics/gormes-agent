package media

import (
	"path/filepath"
	"strings"
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

func FirstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func SafeToken(s string) string {
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
