package stickers

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestStickerCache_MissingAndCorruptReturnMiss(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "missing.json")
	if got, ok, err := GetCachedStickerDescription(missing, "uid-1"); err != nil || ok || got.Description != "" {
		t.Fatalf("missing cache = %+v ok=%v err=%v, want miss without error", got, ok, err)
	}

	corrupt := filepath.Join(t.TempDir(), "sticker_cache.json")
	if err := os.WriteFile(corrupt, []byte(`{bad json`), 0o600); err != nil {
		t.Fatalf("write corrupt cache: %v", err)
	}
	if got, ok, err := GetCachedStickerDescription(corrupt, "uid-1"); err != nil || ok || got.Description != "" {
		t.Fatalf("corrupt cache = %+v ok=%v err=%v, want miss without error", got, ok, err)
	}
}

func TestStickerCache_StoreLookupAndOverwrite(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sticker_cache.json")
	now := time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)

	if err := CacheStickerDescription(path, "uid-1", "A happy dog", "dog", "Dogs", now); err != nil {
		t.Fatalf("CacheStickerDescription: %v", err)
	}
	got, ok, err := GetCachedStickerDescription(path, "uid-1")
	if err != nil || !ok {
		t.Fatalf("GetCachedStickerDescription ok=%v err=%v, want hit", ok, err)
	}
	if got.Description != "A happy dog" || got.Emoji != "dog" || got.SetName != "Dogs" || got.CachedAt != now.Unix() {
		t.Fatalf("cached entry = %+v, want stored metadata", got)
	}

	if err := CacheStickerDescription(path, "uid-1", "A calm dog", "", "", now.Add(time.Hour)); err != nil {
		t.Fatalf("overwrite CacheStickerDescription: %v", err)
	}
	got, ok, err = GetCachedStickerDescription(path, "uid-1")
	if err != nil || !ok || got.Description != "A calm dog" || got.SetName != "" {
		t.Fatalf("overwritten entry = %+v ok=%v err=%v, want new description", got, ok, err)
	}
}

func TestBuildStickerInjection(t *testing.T) {
	tests := []struct {
		name        string
		description string
		emoji       string
		setName     string
		want        string
	}{
		{name: "plain", description: "A waving character", want: `[The user sent a sticker~ It shows: "A waving character" (=^.w.^=)]`},
		{name: "emoji", description: "A character", emoji: "wave", want: `[The user sent a sticker wave~ It shows: "A character" (=^.w.^=)]`},
		{name: "emoji and set", description: "A character", emoji: "wave", setName: "MyPack", want: `[The user sent a sticker wave from "MyPack"~ It shows: "A character" (=^.w.^=)]`},
		{name: "set without emoji", description: "A character", setName: "MyPack", want: `[The user sent a sticker~ It shows: "A character" (=^.w.^=)]`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := BuildStickerInjection(tt.description, tt.emoji, tt.setName); got != tt.want {
				t.Fatalf("BuildStickerInjection = %q, want %q", got, tt.want)
			}
		})
	}

	if got := BuildAnimatedStickerInjection("wave"); got != `[The user sent an animated sticker wave~ I can't see animated ones yet, but the emoji suggests: wave]` {
		t.Fatalf("BuildAnimatedStickerInjection = %q", got)
	}
}
