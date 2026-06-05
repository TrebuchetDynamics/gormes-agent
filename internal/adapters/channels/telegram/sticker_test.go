package telegram

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway"
)

func TestTelegramSticker_CacheHitInjectsDescriptionWithoutDownloadOrVision(t *testing.T) {
	cachePath := filepath.Join(t.TempDir(), "sticker_cache.json")
	if err := gateway.CacheStickerDescription(cachePath, "unique-cat", "A cat waving from a box", "cat", "Cats", time.Unix(100, 0)); err != nil {
		t.Fatalf("CacheStickerDescription: %v", err)
	}

	visionCalls := 0
	client := newMockClient()
	bot := New(Config{
		StickerCachePath: cachePath,
		StickerVisionAnalyzer: func(context.Context, StickerVisionRequest) (string, error) {
			visionCalls++
			return "should not be called", nil
		},
	}, client, nil)

	text, attachments := bot.telegramInboundTextAndAttachments(context.Background(), &tgbotapi.Message{
		Sticker: &tgbotapi.Sticker{
			FileID:       "file-cat",
			FileUniqueID: "unique-cat",
			Emoji:        "fallback-emoji",
			SetName:      "FallbackSet",
		},
	})

	want := `[The user sent a sticker cat from "Cats"~ It shows: "A cat waving from a box" (=^.w.^=)]`
	if text != want {
		t.Fatalf("text = %q, want %q", text, want)
	}
	if len(attachments) != 0 {
		t.Fatalf("attachments = %+v, want none", attachments)
	}
	if visionCalls != 0 {
		t.Fatalf("vision calls = %d, want 0", visionCalls)
	}
	if client.downloadCalls != 0 {
		t.Fatalf("download calls = %d, want 0", client.downloadCalls)
	}
}

func TestTelegramSticker_CacheMissDownloadsAnalyzesAndCachesStaticSticker(t *testing.T) {
	cachePath := filepath.Join(t.TempDir(), "sticker_cache.json")
	attachmentDir := t.TempDir()
	client := newMockClient()
	client.telegramFiles["file-cat"] = tgbotapi.File{FileID: "file-cat", FilePath: "stickers/cat.webp"}
	client.downloads["stickers/cat.webp"] = []byte("fake-webp-bytes")

	var requests []StickerVisionRequest
	bot := New(Config{
		AttachmentCacheDir: attachmentDir,
		StickerCachePath:   cachePath,
		StickerVisionAnalyzer: func(_ context.Context, req StickerVisionRequest) (string, error) {
			requests = append(requests, req)
			raw, err := os.ReadFile(req.ImagePath)
			if err != nil {
				t.Fatalf("vision request ImagePath unreadable: %v", err)
			}
			if string(raw) != "fake-webp-bytes" {
				t.Fatalf("cached sticker bytes = %q, want fake-webp-bytes", string(raw))
			}
			return "A tuxedo cat waving from a cardboard box", nil
		},
	}, client, nil)

	text, attachments := bot.telegramInboundTextAndAttachments(context.Background(), &tgbotapi.Message{
		Sticker: &tgbotapi.Sticker{
			FileID:       "file-cat",
			FileUniqueID: "unique-new-cat",
			Emoji:        "cat",
			SetName:      "Cats",
		},
	})

	want := `[The user sent a sticker cat from "Cats"~ It shows: "A tuxedo cat waving from a cardboard box" (=^.w.^=)]`
	if text != want {
		t.Fatalf("text = %q, want %q", text, want)
	}
	if len(attachments) != 0 {
		t.Fatalf("attachments = %+v, want none", attachments)
	}
	if len(requests) != 1 {
		t.Fatalf("vision requests = %d, want 1", len(requests))
	}
	req := requests[0]
	if req.Prompt != StickerVisionPrompt {
		t.Fatalf("prompt = %q, want sticker prompt", req.Prompt)
	}
	if req.FileUniqueID != "unique-new-cat" || req.Emoji != "cat" || req.SetName != "Cats" {
		t.Fatalf("request metadata = %+v", req)
	}
	if !strings.HasSuffix(req.ImagePath, ".webp") {
		t.Fatalf("image path = %q, want .webp", req.ImagePath)
	}
	if !strings.HasPrefix(req.ImagePath, filepath.Clean(attachmentDir)) {
		t.Fatalf("image path = %q, want under attachment cache %q", req.ImagePath, attachmentDir)
	}
	if client.downloadCalls != 1 {
		t.Fatalf("download calls = %d, want 1", client.downloadCalls)
	}
	cached, ok, err := gateway.GetCachedStickerDescription(cachePath, "unique-new-cat")
	if err != nil || !ok {
		t.Fatalf("GetCachedStickerDescription ok=%v err=%v, want hit", ok, err)
	}
	if cached.Description != "A tuxedo cat waving from a cardboard box" || cached.Emoji != "cat" || cached.SetName != "Cats" {
		t.Fatalf("cached = %+v", cached)
	}
}

func TestTelegramSticker_AnimatedStickerInjectsPlaceholderWithoutDownloadOrVision(t *testing.T) {
	visionCalls := 0
	client := newMockClient()
	bot := New(Config{
		StickerCachePath: filepath.Join(t.TempDir(), "sticker_cache.json"),
		StickerVisionAnalyzer: func(context.Context, StickerVisionRequest) (string, error) {
			visionCalls++
			return "should not be called", nil
		},
	}, client, nil)

	text, attachments := bot.telegramInboundTextAndAttachments(context.Background(), &tgbotapi.Message{
		Sticker: &tgbotapi.Sticker{
			FileID:       "animated-file",
			FileUniqueID: "animated-unique",
			IsAnimated:   true,
			Emoji:        "party",
			SetName:      "PartyPack",
		},
	})

	want := `[The user sent an animated sticker party~ I can't see animated ones yet, but the emoji suggests: party]`
	if text != want {
		t.Fatalf("text = %q, want %q", text, want)
	}
	if len(attachments) != 0 {
		t.Fatalf("attachments = %+v, want none", attachments)
	}
	if visionCalls != 0 {
		t.Fatalf("vision calls = %d, want 0", visionCalls)
	}
	if client.downloadCalls != 0 {
		t.Fatalf("download calls = %d, want 0", client.downloadCalls)
	}
}

func TestTelegramSticker_DownloadOrVisionFailureFallsBackToPlaceholder(t *testing.T) {
	t.Run("download", func(t *testing.T) {
		client := newMockClient()
		client.telegramFiles["file-cat"] = tgbotapi.File{FileID: "file-cat", FilePath: "stickers/cat.webp"}
		client.downloadErr = errors.New("telegram token https://api.telegram.org/botSECRET/file/bad.webp")
		bot := New(Config{
			StickerCachePath: filepath.Join(t.TempDir(), "sticker_cache.json"),
			StickerVisionAnalyzer: func(context.Context, StickerVisionRequest) (string, error) {
				t.Fatal("vision analyzer should not run after download failure")
				return "", nil
			},
		}, client, nil)

		text, attachments := bot.telegramInboundTextAndAttachments(context.Background(), &tgbotapi.Message{
			Sticker: &tgbotapi.Sticker{FileID: "file-cat", FileUniqueID: "unique-cat", Emoji: "cat", SetName: "Cats"},
		})

		want := `[The user sent a sticker cat from "Cats"~ It shows: "a sticker with emoji cat" (=^.w.^=)]`
		if text != want {
			t.Fatalf("text = %q, want %q", text, want)
		}
		if strings.Contains(text, "SECRET") || strings.Contains(text, "api.telegram.org") {
			t.Fatalf("fallback leaked token-bearing error: %q", text)
		}
		if len(attachments) != 0 {
			t.Fatalf("attachments = %+v, want none", attachments)
		}
		if _, ok, err := gateway.GetCachedStickerDescription(bot.telegramStickerCachePath(), "unique-cat"); err != nil || ok {
			t.Fatalf("cache ok=%v err=%v, want miss", ok, err)
		}
	})

	t.Run("vision", func(t *testing.T) {
		client := newMockClient()
		client.telegramFiles["file-dog"] = tgbotapi.File{FileID: "file-dog", FilePath: "stickers/dog.webp"}
		client.downloads["stickers/dog.webp"] = []byte("fake-webp")
		bot := New(Config{
			AttachmentCacheDir: t.TempDir(),
			StickerCachePath:   filepath.Join(t.TempDir(), "sticker_cache.json"),
			StickerVisionAnalyzer: func(context.Context, StickerVisionRequest) (string, error) {
				return "", errors.New("vision provider failed with /tmp/private/sticker.webp")
			},
		}, client, nil)

		text, attachments := bot.telegramInboundTextAndAttachments(context.Background(), &tgbotapi.Message{
			Sticker: &tgbotapi.Sticker{FileID: "file-dog", FileUniqueID: "unique-dog", Emoji: "dog", SetName: "Dogs"},
		})

		want := `[The user sent a sticker dog from "Dogs"~ It shows: "a sticker with emoji dog" (=^.w.^=)]`
		if text != want {
			t.Fatalf("text = %q, want %q", text, want)
		}
		if strings.Contains(text, "/tmp/private") {
			t.Fatalf("fallback leaked local path: %q", text)
		}
		if len(attachments) != 0 {
			t.Fatalf("attachments = %+v, want none", attachments)
		}
		if _, ok, err := gateway.GetCachedStickerDescription(bot.telegramStickerCachePath(), "unique-dog"); err != nil || ok {
			t.Fatalf("cache ok=%v err=%v, want miss", ok, err)
		}
	})
}
