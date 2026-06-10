package stickers

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway/jsonfile"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/redaction"
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
	if !ok || strings.TrimSpace(entry.Description) == "" || entry.CachedAt <= 0 {
		return StickerDescription{}, false, nil
	}
	return entry, true, nil
}

func CacheStickerDescription(path, fileUniqueID, description, emoji, setName string, now time.Time) error {
	key, ok := stickerCacheKey(fileUniqueID)
	if !ok || strings.TrimSpace(description) == "" || now.IsZero() || now.Unix() <= 0 {
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
	if key == "" || hasStickerControl(key) {
		return "", false
	}
	return key, true
}

func hasStickerControl(value string) bool {
	for _, r := range value {
		if r < 0x20 || r == 0x7f || (r >= 0x80 && r <= 0x9f) || hiddenStickerFormattingRune(r) {
			return true
		}
	}
	return false
}

func hiddenStickerFormattingRune(r rune) bool {
	switch {
	case r >= 0x200b && r <= 0x200f:
		return true
	case r >= 0x2028 && r <= 0x202e:
		return true
	case r >= 0x2060 && r <= 0x2069:
		return true
	case r == 0xfeff || r == 0xfffc:
		return true
	case r >= 0xfff9 && r <= 0xfffb:
		return true
	default:
		return false
	}
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
	value = redaction.RedactSecrets(value)
	replacer := strings.NewReplacer(
		"\"", "'",
		"`", "'",
		"[", "(",
		"]", ")",
	)
	var b strings.Builder
	for _, r := range replacer.Replace(value) {
		if r < 0x20 || r == 0x7f || (r >= 0x80 && r <= 0x9f) {
			b.WriteByte(' ')
			continue
		}
		b.WriteRune(r)
	}
	fields := strings.Fields(b.String())
	out := make([]string, 0, len(fields))
	for i := 0; i < len(fields); i++ {
		field := fields[i]
		lower := strings.ToLower(field)
		nextRedacted := i+1 < len(fields) && isStickerRedactionMarker(fields[i+1])
		nextSecretThenRedacted := i+2 < len(fields) && secretLikeStickerField(strings.ToLower(fields[i+1])) && isStickerRedactionMarker(fields[i+2])
		if secretLikeStickerField(lower) && (strings.Contains(lower, "(redacted)") || strings.Contains(lower, "=") || nextRedacted || nextSecretThenRedacted) {
			out = append(out, "[redacted]")
			if nextSecretThenRedacted {
				i += 2
			} else if nextRedacted {
				i++
			}
			continue
		}
		out = append(out, field)
	}
	return strings.Join(out, " ")
}

func isStickerRedactionMarker(value string) bool {
	return strings.Contains(strings.ToLower(value), "(redacted)") || strings.Contains(strings.ToLower(value), "[redacted]")
}

func secretLikeStickerField(value string) bool {
	return strings.Contains(value, "api_key") || strings.Contains(value, "api-key") || strings.Contains(value, "apikey") || strings.Contains(value, "authorization") || strings.Contains(value, "bearer") || strings.Contains(value, "token") || strings.Contains(value, "secret") || strings.Contains(value, "password")
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
