package gateway

import (
	"path/filepath"
	"testing"
	"time"
)

func TestStickerCache_PutLookupAndPersistNormalizedKeys(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sticker_cache.json")

	cache, err := OpenStickerCache(path)
	if err != nil {
		t.Fatalf("OpenStickerCache: %v", err)
	}

	cachedAt := time.Date(2026, 4, 23, 8, 39, 9, 0, time.UTC)
	if err := cache.Put(StickerCacheRecord{
		Platform:  " Telegram ",
		ChatID:    " 42 ",
		Alias:     " Party Parrot ",
		StickerID: " sticker-file-99 ",
		CachedAt:  cachedAt,
	}); err != nil {
		t.Fatalf("Put telegram sticker: %v", err)
	}
	if err := cache.Put(StickerCacheRecord{
		Platform:  "discord",
		Alias:     "ThumbsUp",
		StickerID: "discord-sticker-1",
		CachedAt:  cachedAt.Add(time.Minute),
	}); err != nil {
		t.Fatalf("Put discord sticker: %v", err)
	}

	got, ok := cache.Lookup("telegram", "42", "party parrot")
	if !ok {
		t.Fatal("Lookup(telegram/42/party parrot) = ok=false, want true")
	}
	if got.Platform != "telegram" || got.ChatID != "42" || got.Alias != "Party Parrot" || got.StickerID != "sticker-file-99" {
		t.Fatalf("telegram sticker = %+v, want normalized record", got)
	}

	reopened, err := OpenStickerCache(path)
	if err != nil {
		t.Fatalf("OpenStickerCache(reopen): %v", err)
	}
	got, ok = reopened.Lookup("DISCORD", "", "thumbsup")
	if !ok {
		t.Fatal("Lookup(DISCORD/thumbsup) after reopen = ok=false, want true")
	}
	if got.StickerID != "discord-sticker-1" {
		t.Fatalf("discord sticker id = %q, want discord-sticker-1", got.StickerID)
	}

	snapshot := reopened.Snapshot()
	if len(snapshot) != 2 {
		t.Fatalf("Snapshot len = %d, want 2", len(snapshot))
	}
	if snapshot[0].Platform != "discord" || snapshot[1].Platform != "telegram" {
		t.Fatalf("Snapshot order = %+v, want discord before telegram", snapshot)
	}
}
