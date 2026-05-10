package gateway

import (
	"bytes"
	"context"
	"encoding/base64"
	"image"
	"image/color"
	"image/jpeg"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/session"
)

// writeFixtureJPEGForManager mirrors writeFixtureJPEG in turn_adapter_test.go
// but lives here so this test file is self-contained. Generates a synthetic
// 4x4 solid-color JPEG into the supplied dir.
func writeFixtureJPEGForManager(t *testing.T, dir, name string, fillR, fillG, fillB uint8) string {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 4, 4))
	fill := color.RGBA{R: fillR, G: fillG, B: fillB, A: 255}
	for y := 0; y < 4; y++ {
		for x := 0; x < 4; x++ {
			img.Set(x, y, fill)
		}
	}
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 90}); err != nil {
		t.Fatalf("encode fixture jpeg: %v", err)
	}
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, buf.Bytes(), 0o600); err != nil {
		t.Fatalf("write fixture jpeg: %v", err)
	}
	return path
}

// TestManager_SubmitPinned_PhotoAttachmentBecomesImageURLContentPart drives the
// real channel-inbound submission path (Manager.submitPinned, called from
// Manager.Run when the fake channel pushes an InboundEvent). The test fails if
// the photo bytes do not reach the kernel as an image_url ContentPart — which
// is the actual regression observed in production: the operator sent a
// screenshot, the bot replied with "vision/OCR backend not configured" because
// the image bytes never made it onto the kernel turn message.
func TestManager_SubmitPinned_PhotoAttachmentBecomesImageURLContentPart(t *testing.T) {
	dir := t.TempDir()
	jpgPath := writeFixtureJPEGForManager(t, dir, "screenshot.jpg", 100, 200, 150)
	wantBytes, err := os.ReadFile(jpgPath)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	wantDataURI := "data:image/jpeg;base64," + base64.StdEncoding.EncodeToString(wantBytes)

	platform := "telegram"
	tg := newFakeChannel(platform)
	fk := &fakeKernel{}
	smap := session.NewMemMap()
	if err := smap.Put(context.Background(), platform+":42", "sess-stored"); err != nil {
		t.Fatalf("Put: %v", err)
	}
	cfg := ManagerConfig{
		AllowedChats: map[string]string{platform: "42"},
		SessionMap:   smap,
	}
	m := NewManagerWithSubmitter(cfg, fk, slog.Default())
	if err := m.Register(tg); err != nil {
		t.Fatalf("Register: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { _ = m.Run(ctx) }()

	tg.pushInbound(InboundEvent{
		Platform: platform,
		ChatID:   "42",
		UserID:   "7",
		MsgID:    "m-photo-1",
		Kind:     EventSubmit,
		Text:     "what is in this picture?",
		Attachments: []Attachment{
			{Kind: "photo", URL: jpgPath, MediaType: "image/jpeg", FileName: "screenshot.jpg", SizeBytes: int64(len(wantBytes))},
		},
	})

	waitFor(t, 1*time.Second, func() bool {
		return len(fk.submitsSnapshot()) == 1
	})
	submits := fk.submitsSnapshot()
	if len(submits) != 1 {
		t.Fatalf("kernel got %d submits, want 1", len(submits))
	}
	ev := submits[0]
	if len(ev.ContentParts) == 0 {
		t.Fatal("kernel.PlatformEvent.ContentParts is empty; expected at least one image_url part for the photo attachment. The Telegram-inbound channel path (Manager.submitPinned) is not populating ContentParts from ev.Attachments.")
	}
	var sawImage bool
	for _, part := range ev.ContentParts {
		if part.Type == "image_url" {
			sawImage = true
			if part.ImageURL != wantDataURI {
				t.Fatalf("image_url payload mismatch:\n got %q\nwant %q", part.ImageURL, wantDataURI)
			}
		}
	}
	if !sawImage {
		t.Fatal("no image_url ContentPart found in kernel submit; expected one for the photo attachment")
	}
}

func TestManager_SubmitPinned_ImageOnlyPhotoUsesDefaultPromptContentPart(t *testing.T) {
	dir := t.TempDir()
	jpgPath := writeFixtureJPEGForManager(t, dir, "screenshot.jpg", 100, 200, 150)

	platform := "telegram"
	tg := newFakeChannel(platform)
	fk := &fakeKernel{}
	smap := session.NewMemMap()
	if err := smap.Put(context.Background(), platform+":42", "sess-stored"); err != nil {
		t.Fatalf("Put: %v", err)
	}
	cfg := ManagerConfig{
		AllowedChats: map[string]string{platform: "42"},
		SessionMap:   smap,
	}
	m := NewManagerWithSubmitter(cfg, fk, slog.Default())
	if err := m.Register(tg); err != nil {
		t.Fatalf("Register: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { _ = m.Run(ctx) }()

	tg.pushInbound(InboundEvent{
		Platform: platform,
		ChatID:   "42",
		UserID:   "7",
		MsgID:    "m-photo-only",
		Kind:     EventSubmit,
		Attachments: []Attachment{
			{Kind: "photo", URL: jpgPath, MediaType: "image/jpeg", FileName: "screenshot.jpg"},
		},
	})

	waitFor(t, 1*time.Second, func() bool {
		return len(fk.submitsSnapshot()) == 1
	})
	submits := fk.submitsSnapshot()
	if len(submits) != 1 {
		t.Fatalf("kernel got %d submits, want 1", len(submits))
	}
	ev := submits[0]
	if !strings.Contains(ev.Text, "Attachments:") {
		t.Fatalf("plain text projection = %q, want attachment marker preserved", ev.Text)
	}
	wantTextPart := "What do you see in this image?\n\n[Image attached at: " + jpgPath + "]"
	if len(ev.ContentParts) != 2 {
		t.Fatalf("ContentParts len = %d, want text plus image: %+v", len(ev.ContentParts), ev.ContentParts)
	}
	if ev.ContentParts[0].Type != "text" || ev.ContentParts[0].Text != wantTextPart {
		t.Fatalf("ContentParts[0] = %+v, want default image prompt %q", ev.ContentParts[0], wantTextPart)
	}
	if strings.Contains(ev.ContentParts[0].Text, "Attachments:") {
		t.Fatalf("ContentParts[0].Text = %q, must not use generated attachment marker as caption", ev.ContentParts[0].Text)
	}
}
