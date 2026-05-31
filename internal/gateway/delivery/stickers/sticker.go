package stickers

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway/jsonfile"
)

type StickerDescription struct {
	Description string `json:"description"`
	Emoji       string `json:"emoji"`
	SetName     string `json:"set_name"`
	CachedAt    int64  `json:"cached_at"`
}

func GetCachedStickerDescription(path, fileUniqueID string) (StickerDescription, bool, error) {
	cache, err := loadStickerCache(path)
	if err != nil {
		return StickerDescription{}, false, err
	}
	entry, ok := cache[strings.TrimSpace(fileUniqueID)]
	return entry, ok, nil
}

func CacheStickerDescription(path, fileUniqueID, description, emoji, setName string, now time.Time) error {
	cache, err := loadStickerCache(path)
	if err != nil {
		return err
	}
	cache[strings.TrimSpace(fileUniqueID)] = StickerDescription{
		Description: strings.TrimSpace(description),
		Emoji:       strings.TrimSpace(emoji),
		SetName:     strings.TrimSpace(setName),
		CachedAt:    now.Unix(),
	}
	return saveStickerCache(path, cache)
}

func BuildStickerInjection(description, emoji, setName string) string {
	description = strings.TrimSpace(description)
	emoji = strings.TrimSpace(emoji)
	setName = strings.TrimSpace(setName)
	context := ""
	if setName != "" && emoji != "" {
		context = " " + emoji + ` from "` + setName + `"`
	} else if emoji != "" {
		context = " " + emoji
	}
	return `[The user sent a sticker` + context + `~ It shows: "` + description + `" (=^.w.^=)]`
}

func BuildAnimatedStickerInjection(emoji string) string {
	emoji = strings.TrimSpace(emoji)
	if emoji != "" {
		return `[The user sent an animated sticker ` + emoji + `~ I can't see animated ones yet, but the emoji suggests: ` + emoji + `]`
	}
	return "[The user sent an animated sticker~ I can't see animated ones yet]"
}

func loadStickerCache(path string) (map[string]StickerDescription, error) {
	var cache map[string]StickerDescription
	exists, err := jsonfile.Read(context.Background(), path, &cache, "sticker cache")
	if err != nil {
		if errors.Is(err, jsonfile.ErrEmpty) || !jsonfile.IsReadError(err) {
			return map[string]StickerDescription{}, nil
		}
		return nil, err
	}
	if !exists || cache == nil {
		return map[string]StickerDescription{}, nil
	}
	return cache, nil
}

func saveStickerCache(path string, cache map[string]StickerDescription) error {
	return jsonfile.WriteAtomicWithOptions(context.Background(), path, cache, "sticker cache", jsonfile.WriteOptions{
		DirMode:    0o700,
		FileMode:   0o600,
		TmpPattern: ".sticker-cache-*.tmp",
	})
}
