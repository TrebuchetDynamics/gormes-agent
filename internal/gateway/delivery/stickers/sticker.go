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
	key, ok := stickerCacheKey(fileUniqueID)
	if !ok {
		return StickerDescription{}, false, nil
	}
	cache, err := loadStickerCache(path)
	if err != nil {
		return StickerDescription{}, false, err
	}
	entry, ok := cache[key]
	return entry, ok, nil
}

func CacheStickerDescription(path, fileUniqueID, description, emoji, setName string, now time.Time) error {
	key, ok := stickerCacheKey(fileUniqueID)
	if !ok {
		return nil
	}
	cache, err := loadStickerCacheForWrite(path)
	if err != nil {
		return err
	}
	cache[key] = StickerDescription{
		Description: strings.TrimSpace(description),
		Emoji:       strings.TrimSpace(emoji),
		SetName:     strings.TrimSpace(setName),
		CachedAt:    now.Unix(),
	}
	return saveStickerCache(path, cache)
}

func stickerCacheKey(fileUniqueID string) (string, bool) {
	key := strings.TrimSpace(fileUniqueID)
	return key, key != ""
}

func BuildStickerInjection(description, emoji, setName string) string {
	description = sanitizeStickerPromptField(description)
	emoji = sanitizeStickerPromptField(emoji)
	setName = sanitizeStickerPromptField(setName)
	context := ""
	if setName != "" && emoji != "" {
		context = " " + emoji + ` from "` + setName + `"`
	} else if emoji != "" {
		context = " " + emoji
	}
	return `[The user sent a sticker` + context + `~ It shows: "` + description + `" (=^.w.^=)]`
}

func BuildAnimatedStickerInjection(emoji string) string {
	emoji = sanitizeStickerPromptField(emoji)
	if emoji != "" {
		return `[The user sent an animated sticker ` + emoji + `~ I can't see animated ones yet, but the emoji suggests: ` + emoji + `]`
	}
	return "[The user sent an animated sticker~ I can't see animated ones yet]"
}

func sanitizeStickerPromptField(value string) string {
	replacer := strings.NewReplacer(
		"\"", "'",
		"`", "'",
		"[", "(",
		"]", ")",
	)
	return strings.Join(strings.Fields(replacer.Replace(value)), " ")
}

func loadStickerCache(path string) (map[string]StickerDescription, error) {
	cache, err := readStickerCache(path)
	if err != nil {
		if errors.Is(err, jsonfile.ErrEmpty) || !jsonfile.IsReadError(err) {
			return map[string]StickerDescription{}, nil
		}
		return nil, err
	}
	return cache, nil
}

func readStickerCache(path string) (map[string]StickerDescription, error) {
	if strings.TrimSpace(path) == "" {
		return map[string]StickerDescription{}, nil
	}
	var cache map[string]StickerDescription
	exists, err := jsonfile.Read(context.Background(), path, &cache, "sticker cache")
	if err != nil {
		return nil, err
	}
	if !exists || cache == nil {
		return map[string]StickerDescription{}, nil
	}
	return cache, nil
}

func loadStickerCacheForWrite(path string) (map[string]StickerDescription, error) {
	cache, err := readStickerCache(path)
	if err != nil {
		if errors.Is(err, jsonfile.ErrEmpty) {
			return map[string]StickerDescription{}, nil
		}
		return nil, err
	}
	return cache, nil
}

func saveStickerCache(path string, cache map[string]StickerDescription) error {
	if strings.TrimSpace(path) == "" {
		return nil
	}
	return jsonfile.WriteAtomicWithOptions(context.Background(), path, cache, "sticker cache", jsonfile.WriteOptions{
		DirMode:    0o700,
		FileMode:   0o600,
		TmpPattern: ".sticker-cache-*.tmp",
	})
}
