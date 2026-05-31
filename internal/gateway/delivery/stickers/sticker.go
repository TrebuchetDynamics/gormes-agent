package stickers

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"
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
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]StickerDescription{}, nil
		}
		return nil, err
	}
	var cache map[string]StickerDescription
	if err := json.Unmarshal(raw, &cache); err != nil || cache == nil {
		return map[string]StickerDescription{}, nil
	}
	return cache, nil
}

func saveStickerCache(path string, cache map[string]StickerDescription) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(cache, "", "  ")
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".sticker-cache-*.tmp")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer func() { _ = os.Remove(tmpName) }()
	if _, err := tmp.Write(raw); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Chmod(tmpName, 0o600); err != nil {
		return err
	}
	return os.Rename(tmpName, path)
}
