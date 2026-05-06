package telegram

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway"
)

func TestTelegramPhotoBatchCoalescesBurst(t *testing.T) {
	ev := runTelegramPhotoBatch(t, "", 21*time.Millisecond)
	if ev.Text != "first caption" {
		t.Fatalf("Text = %q, want first caption", ev.Text)
	}
	if ev.MsgID != "11" {
		t.Fatalf("MsgID = %q, want first message id", ev.MsgID)
	}
	assertTelegramPhotoAttachments(t, ev, 2)
}

func TestTelegramPhotoBatchCoalescesMediaGroupAlbum(t *testing.T) {
	ev := runTelegramPhotoBatch(t, "album-1", 21*time.Millisecond)
	if ev.Text != "first caption" {
		t.Fatalf("Text = %q, want first caption", ev.Text)
	}
	assertTelegramPhotoAttachments(t, ev, 2)
}

func TestTelegramPhotoBatchDisconnectCancelsPendingFlush(t *testing.T) {
	mc := newMockClient()
	cacheDir := t.TempDir()
	mc.telegramFiles["photo-1"] = tgbotapi.File{FileID: "photo-1", FilePath: "photos/one.jpg"}
	mc.downloads["photos/one.jpg"] = []byte("photo one")
	b := New(Config{AttachmentCacheDir: cacheDir, MediaBatchDelay: time.Hour}, mc, nil)
	inbox := make(chan gateway.InboundEvent, 2)

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		_ = b.Run(ctx, inbox)
		close(done)
	}()
	defer func() {
		cancel()
		<-done
	}()

	mc.pushPhotoUpdate(42, 11, "first caption", "", []tgbotapi.PhotoSize{
		{FileID: "photo-1-small", Width: 64, Height: 64, FileSize: 3},
		{FileID: "photo-1", Width: 640, Height: 480, FileSize: 9},
	})
	waitForTelegramPhotoBatchCount(t, b, 1)

	if err := b.Disconnect(context.Background()); err != nil {
		t.Fatalf("Disconnect() error = %v", err)
	}
	waitForTelegramPhotoBatchCount(t, b, 0)

	select {
	case got := <-inbox:
		t.Fatalf("got pending photo event after disconnect: %#v", got)
	case <-time.After(40 * time.Millisecond):
	}
}

func runTelegramPhotoBatch(t *testing.T, mediaGroupID string, delay time.Duration) gateway.InboundEvent {
	t.Helper()
	mc := newMockClient()
	cacheDir := t.TempDir()
	mc.telegramFiles["photo-1"] = tgbotapi.File{FileID: "photo-1", FilePath: "photos/one.jpg"}
	mc.telegramFiles["photo-2"] = tgbotapi.File{FileID: "photo-2", FilePath: "photos/two.webp"}
	mc.downloads["photos/one.jpg"] = []byte("photo one")
	mc.downloads["photos/two.webp"] = []byte("photo two")

	b := New(Config{AttachmentCacheDir: cacheDir, MediaBatchDelay: delay}, mc, nil)
	inbox := make(chan gateway.InboundEvent, 4)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		_ = b.Run(ctx, inbox)
		close(done)
	}()
	defer func() {
		cancel()
		<-done
	}()

	mc.pushPhotoUpdate(42, 11, "first caption", mediaGroupID, []tgbotapi.PhotoSize{
		{FileID: "photo-1-small", Width: 64, Height: 64, FileSize: 3},
		{FileID: "photo-1", Width: 640, Height: 480, FileSize: 9},
	})
	mc.pushPhotoUpdate(42, 12, "", mediaGroupID, []tgbotapi.PhotoSize{
		{FileID: "photo-2", Width: 800, Height: 600, FileSize: 9},
	})

	var ev gateway.InboundEvent
	select {
	case ev = <-inbox:
	case <-time.After(300 * time.Millisecond):
		t.Fatal("timed out waiting for photo batch")
	}
	select {
	case extra := <-inbox:
		t.Fatalf("got extra unbatched photo event: %#v", extra)
	case <-time.After(2 * delay):
	}
	return ev
}

func assertTelegramPhotoAttachments(t *testing.T, ev gateway.InboundEvent, want int) {
	t.Helper()
	if len(ev.Attachments) != want {
		t.Fatalf("attachments = %#v, want %d", ev.Attachments, want)
	}
	for i, att := range ev.Attachments {
		if att.Kind != "photo" || att.URL == "" || att.MediaType == "" || att.SourceID == "" || att.SizeBytes == 0 {
			t.Fatalf("attachment[%d] = %#v", i, att)
		}
		if strings.Contains(ev.Text, att.URL) {
			t.Fatalf("Text leaks cached photo path %q: %q", att.URL, ev.Text)
		}
		if _, err := os.Stat(att.URL); err != nil {
			t.Fatalf("cached photo[%d] path %q: %v", i, att.URL, err)
		}
	}
}

func waitForTelegramPhotoBatchCount(t *testing.T, b *Bot, want int) {
	t.Helper()
	deadline := time.Now().Add(200 * time.Millisecond)
	for time.Now().Before(deadline) {
		if got := b.pendingPhotoBatchCount(); got == want {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("pending photo batch count = %d, want %d", b.pendingPhotoBatchCount(), want)
}
