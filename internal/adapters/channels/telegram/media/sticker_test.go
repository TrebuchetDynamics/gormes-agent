package media

import "testing"

func TestStickerFileNameUsesStableSafeIdentifier(t *testing.T) {
	if got := StickerFileName("stickers/raw.webp", "unique id", "file/id"); got != "sticker_unique_id.webp" {
		t.Fatalf("StickerFileName = %q, want sticker_unique_id.webp", got)
	}
	if got := StickerFileName("", "", "file/id"); got != "sticker_file_id.webp" {
		t.Fatalf("StickerFileName fallback = %q, want sticker_file_id.webp", got)
	}
}

func TestStickerFallbackDescription(t *testing.T) {
	if got := StickerFallbackDescription(" 😀 "); got != "a sticker with emoji 😀" {
		t.Fatalf("description = %q", got)
	}
	if got := StickerFallbackDescription(" "); got != "a sticker" {
		t.Fatalf("blank description = %q", got)
	}
}

func TestFirstNonEmptyAndSafeToken(t *testing.T) {
	if got := FirstNonEmpty(" ", " second "); got != "second" {
		t.Fatalf("FirstNonEmpty = %q, want second", got)
	}
	if got := SafeToken(" /// "); got != "telegram" {
		t.Fatalf("SafeToken = %q, want telegram", got)
	}
}
