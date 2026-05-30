package media

import (
	"strings"
	"testing"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func TestDocumentMediaHelpers(t *testing.T) {
	if got := CleanMediaType(" Text/Plain; charset=utf-8 "); got != "text/plain" {
		t.Fatalf("CleanMediaType() = %q", got)
	}
	if got := InferExtension("", "application/pdf"); got != ".pdf" {
		t.Fatalf("InferExtension() = %q, want .pdf", got)
	}
	if got := DisplayFileName("../unsafe.txt", ".txt", "document"); got != "unsafe.txt" {
		t.Fatalf("DisplayFileName() = %q", got)
	}
	if !ShouldInlineTextDocument(".md", InlineTextDocumentSize) {
		t.Fatal("ShouldInlineTextDocument(.md) = false, want true")
	}
	if ShouldInlineTextDocument(".md", InlineTextDocumentSize+1) {
		t.Fatal("ShouldInlineTextDocument(over cap) = true, want false")
	}
}

func TestDocumentMarkersRemainOperatorReadable(t *testing.T) {
	unsupported := UnsupportedDocumentMarker("setup.exe", "application/x-msdownload", "unsupported extension \".exe\"")
	if !strings.Contains(unsupported, "Unsupported Telegram document type") || !strings.Contains(unsupported, "supported types:") {
		t.Fatalf("UnsupportedDocumentMarker() = %q", unsupported)
	}
	oversized := DocumentSizeMarker("document", "huge.pdf", "application/pdf", MaxDocumentBytes+1)
	if !strings.Contains(oversized, "too large") || !strings.Contains(oversized, "maximum=20 MB") {
		t.Fatalf("DocumentSizeMarker() = %q", oversized)
	}
	cached := CachedAttachmentMarker("video", "clip.mp4", "video/mp4", 12)
	if !strings.Contains(cached, "Telegram video message attached") || !strings.Contains(cached, "size=12 bytes") {
		t.Fatalf("CachedAttachmentMarker() = %q", cached)
	}
}

func TestPhotoHelpers(t *testing.T) {
	photo, ok := LargestPhoto([]tgbotapi.PhotoSize{
		{FileID: "small", Width: 10, Height: 10},
		{FileID: "large", Width: 20, Height: 20},
	})
	if !ok || photo.FileID != "large" {
		t.Fatalf("LargestPhoto() = %#v, %v; want large", photo, ok)
	}
	if got := PhotoExtension("photos/image.webp"); got != ".webp" {
		t.Fatalf("PhotoExtension() = %q", got)
	}
	if got := PhotoMediaType(".png"); got != "image/png" {
		t.Fatalf("PhotoMediaType() = %q", got)
	}
}
