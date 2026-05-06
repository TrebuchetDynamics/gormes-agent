package telegram

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway"
)

func TestTelegramDocumentSupportedPDFCachesDescriptor(t *testing.T) {
	mc := newMockClient()
	cacheDir := t.TempDir()
	payload := []byte("%PDF-1.7\nbody")
	mc.telegramFiles["doc-pdf"] = tgbotapi.File{FileID: "doc-pdf", FilePath: "documents/report.pdf"}
	mc.downloads["documents/report.pdf"] = payload

	b := New(Config{AttachmentCacheDir: cacheDir}, mc, nil)
	ev := telegramDocumentEvent(t, b, tgbotapi.Document{
		FileID:   "doc-pdf",
		FileName: "report.pdf",
		MimeType: "application/pdf",
		FileSize: len(payload),
	}, "please inspect")

	if !strings.Contains(ev.Text, "please inspect") {
		t.Fatalf("Text = %q, want caption preserved", ev.Text)
	}
	if !strings.Contains(ev.Text, "Telegram document message attached") {
		t.Fatalf("Text = %q, want document evidence marker", ev.Text)
	}
	if strings.Contains(ev.Text, cacheDir) {
		t.Fatalf("Text leaks cache path %q: %q", cacheDir, ev.Text)
	}
	if len(ev.Attachments) != 1 {
		t.Fatalf("attachments = %d, want 1: %#v", len(ev.Attachments), ev.Attachments)
	}
	att := ev.Attachments[0]
	if att.Kind != "document" || att.FileName != "report.pdf" || att.MediaType != "application/pdf" || att.SourceID != "doc-pdf" {
		t.Fatalf("attachment descriptor = %#v", att)
	}
	if att.SizeBytes != int64(len(payload)) {
		t.Fatalf("SizeBytes = %d, want %d", att.SizeBytes, len(payload))
	}
	if !strings.HasPrefix(att.URL, cacheDir+string(os.PathSeparator)) {
		t.Fatalf("cached URL = %q, want under %q", att.URL, cacheDir)
	}
	got, err := os.ReadFile(att.URL)
	if err != nil {
		t.Fatalf("read cached document: %v", err)
	}
	if !bytes.Equal(got, payload) {
		t.Fatalf("cached bytes = %q, want %q", got, payload)
	}
	submit := ev.SubmitText()
	if !strings.Contains(submit, "sizeBytes=") || !strings.Contains(submit, "report.pdf") {
		t.Fatalf("SubmitText() = %q, want filename and size evidence", submit)
	}
}

func TestTelegramDocumentSupportedFileMatrixCachesDescriptors(t *testing.T) {
	tests := []struct {
		name          string
		fileName      string
		mediaType     string
		payload       []byte
		wantFileName  string
		wantInline    bool
		wantMediaType string
	}{
		{
			name:          "csv",
			fileName:      "data.csv",
			mediaType:     "text/csv",
			payload:       []byte("a,b\n1,2\n"),
			wantFileName:  "data.csv",
			wantMediaType: "text/csv",
		},
		{
			name:          "markdown",
			fileName:      "notes.md",
			mediaType:     "text/markdown",
			payload:       []byte("# Notes\nsmall markdown\n"),
			wantFileName:  "notes.md",
			wantInline:    true,
			wantMediaType: "text/markdown",
		},
		{
			name:          "zip",
			fileName:      "bundle.zip",
			mediaType:     "application/zip",
			payload:       []byte("zip bytes"),
			wantFileName:  "bundle.zip",
			wantMediaType: "application/zip",
		},
		{
			name:          "missing filename uses MIME",
			mediaType:     "application/pdf",
			payload:       []byte("%PDF inferred"),
			wantFileName:  "document.pdf",
			wantMediaType: "application/pdf",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := newMockClient()
			cacheDir := t.TempDir()
			fileID := "doc-" + strings.ReplaceAll(tt.name, " ", "-")
			mc.telegramFiles[fileID] = tgbotapi.File{FileID: fileID, FilePath: "documents/" + tt.wantFileName}
			mc.downloads["documents/"+tt.wantFileName] = tt.payload
			b := New(Config{AttachmentCacheDir: cacheDir}, mc, nil)

			ev := telegramDocumentEvent(t, b, tgbotapi.Document{
				FileID:   fileID,
				FileName: tt.fileName,
				MimeType: tt.mediaType,
				FileSize: len(tt.payload),
			}, "caption")

			if len(ev.Attachments) != 1 {
				t.Fatalf("attachments = %#v, want one cached descriptor", ev.Attachments)
			}
			att := ev.Attachments[0]
			if att.Kind != "document" || att.FileName != tt.wantFileName || att.MediaType != tt.wantMediaType || att.SizeBytes != int64(len(tt.payload)) {
				t.Fatalf("attachment = %#v", att)
			}
			if _, err := os.Stat(att.URL); err != nil {
				t.Fatalf("cached path %q: %v", att.URL, err)
			}
			if tt.wantInline && !strings.Contains(ev.Text, "[Content of "+tt.wantFileName+"]") {
				t.Fatalf("Text = %q, want inline content", ev.Text)
			}
			if !tt.wantInline && strings.Contains(ev.Text, "[Content of "+tt.wantFileName+"]") {
				t.Fatalf("Text = %q, want descriptor without inline content", ev.Text)
			}
		})
	}
}

func TestTelegramDocumentSmallTextInjectsContentBeforeCaption(t *testing.T) {
	mc := newMockClient()
	cacheDir := t.TempDir()
	payload := []byte("alpha\nbeta\n")
	mc.telegramFiles["doc-txt"] = tgbotapi.File{FileID: "doc-txt", FilePath: "documents/notes.txt"}
	mc.downloads["documents/notes.txt"] = payload

	b := New(Config{AttachmentCacheDir: cacheDir}, mc, nil)
	ev := telegramDocumentEvent(t, b, tgbotapi.Document{
		FileID:   "doc-txt",
		FileName: "notes.txt",
		MimeType: "text/plain",
		FileSize: len(payload),
	}, "summarize this")

	contentIdx := strings.Index(ev.Text, "[Content of notes.txt]:\nalpha\nbeta")
	captionIdx := strings.Index(ev.Text, "summarize this")
	if contentIdx < 0 || captionIdx < 0 || contentIdx > captionIdx {
		t.Fatalf("Text = %q, want injected content before caption", ev.Text)
	}
	if len(ev.Attachments) != 1 || ev.Attachments[0].URL == "" {
		t.Fatalf("attachments = %#v, want cached text document", ev.Attachments)
	}
}

func TestTelegramDocumentLargeOrBinaryTextCachesWithoutRawInjection(t *testing.T) {
	tests := []struct {
		name    string
		payload []byte
	}{
		{name: "over cap", payload: bytes.Repeat([]byte("a"), 100*1024+1)},
		{name: "binary", payload: []byte{0xff, 0xfe, 0xfd}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := newMockClient()
			cacheDir := t.TempDir()
			mc.telegramFiles["doc-txt"] = tgbotapi.File{FileID: "doc-txt", FilePath: "documents/notes.txt"}
			mc.downloads["documents/notes.txt"] = tt.payload

			b := New(Config{AttachmentCacheDir: cacheDir}, mc, nil)
			ev := telegramDocumentEvent(t, b, tgbotapi.Document{
				FileID:   "doc-txt",
				FileName: "notes.txt",
				MimeType: "text/plain",
				FileSize: len(tt.payload),
			}, "caption")

			if strings.Contains(ev.Text, "[Content of notes.txt]") {
				t.Fatalf("Text = %q, want no raw content injection", ev.Text)
			}
			if !strings.Contains(ev.Text, "caption") || len(ev.Attachments) != 1 {
				t.Fatalf("event = %#v, want caption and cached attachment", ev)
			}
		})
	}
}

func TestTelegramDocumentUnsafeInputsEmitExplicitEvidenceWithoutDownload(t *testing.T) {
	tests := []struct {
		name string
		doc  tgbotapi.Document
		want string
	}{
		{
			name: "unsupported extension",
			doc:  tgbotapi.Document{FileID: "bad-ext", FileName: "setup.exe", MimeType: "application/x-msdownload", FileSize: 12},
			want: "Unsupported Telegram document type",
		},
		{
			name: "missing filename and mime",
			doc:  tgbotapi.Document{FileID: "no-evidence", FileSize: 12},
			want: "missing filename or MIME type",
		},
		{
			name: "unknown size",
			doc:  tgbotapi.Document{FileID: "no-size", FileName: "report.pdf", MimeType: "application/pdf"},
			want: "size could not be verified",
		},
		{
			name: "oversized",
			doc:  tgbotapi.Document{FileID: "huge", FileName: "huge.pdf", MimeType: "application/pdf", FileSize: 21 * 1024 * 1024},
			want: "too large",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := newMockClient()
			b := New(Config{AttachmentCacheDir: t.TempDir()}, mc, nil)
			ev := telegramDocumentEvent(t, b, tt.doc, "")

			if strings.TrimSpace(ev.SubmitText()) == "" {
				t.Fatal("SubmitText() is blank; want operator-visible evidence")
			}
			if !strings.Contains(ev.Text, tt.want) {
				t.Fatalf("Text = %q, want %q", ev.Text, tt.want)
			}
			if len(ev.Attachments) != 0 {
				t.Fatalf("attachments = %#v, want no unsafe download descriptor", ev.Attachments)
			}
			if got := mc.downloadCallCount(); got != 0 {
				t.Fatalf("download calls = %d, want 0", got)
			}
		})
	}
}

func TestTelegramDocumentAndNativeVideoCacheAsVideo(t *testing.T) {
	tests := []struct {
		name  string
		event func(*mockClient, string) gateway.InboundEvent
	}{
		{
			name: "mp4 document",
			event: func(mc *mockClient, cacheDir string) gateway.InboundEvent {
				payload := []byte("mp4 document bytes")
				mc.telegramFiles["doc-mp4"] = tgbotapi.File{FileID: "doc-mp4", FilePath: "videos/clip.mp4"}
				mc.downloads["videos/clip.mp4"] = payload
				b := New(Config{AttachmentCacheDir: cacheDir}, mc, nil)
				return telegramDocumentEvent(t, b, tgbotapi.Document{
					FileID:   "doc-mp4",
					FileName: "clip.mp4",
					MimeType: "video/mp4",
					FileSize: len(payload),
				}, "watch this")
			},
		},
		{
			name: "native video",
			event: func(mc *mockClient, cacheDir string) gateway.InboundEvent {
				payload := []byte("native video bytes")
				mc.telegramFiles["native-video"] = tgbotapi.File{FileID: "native-video", FilePath: "videos/native.mp4"}
				mc.downloads["videos/native.mp4"] = payload
				b := New(Config{AttachmentCacheDir: cacheDir}, mc, nil)
				u := tgbotapi.Update{Message: &tgbotapi.Message{
					MessageID: 9,
					Caption:   "watch this",
					Video:     &tgbotapi.Video{FileID: "native-video", FileName: "native.mp4", MimeType: "video/mp4", FileSize: len(payload)},
					Chat:      &tgbotapi.Chat{ID: 42, Type: "private"},
					From:      &tgbotapi.User{ID: 42, FirstName: "tester"},
				}}
				ev, ok := b.toInboundEvent(context.Background(), u)
				if !ok {
					t.Fatal("toInboundEvent skipped native video")
				}
				return ev
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := newMockClient()
			ev := tt.event(mc, t.TempDir())
			if len(ev.Attachments) != 1 {
				t.Fatalf("attachments = %#v, want one video", ev.Attachments)
			}
			att := ev.Attachments[0]
			if att.Kind != "video" || att.MediaType != "video/mp4" || att.URL == "" || att.SizeBytes == 0 {
				t.Fatalf("video attachment = %#v", att)
			}
			if !strings.Contains(ev.Text, "watch this") || !strings.Contains(ev.Text, "Telegram video message attached") {
				t.Fatalf("Text = %q, want caption and video evidence", ev.Text)
			}
			if _, err := os.Stat(att.URL); err != nil {
				t.Fatalf("cached video path %q: %v", att.URL, err)
			}
		})
	}
}

func telegramDocumentEvent(t *testing.T, b *Bot, document tgbotapi.Document, caption string) gateway.InboundEvent {
	t.Helper()
	u := tgbotapi.Update{Message: &tgbotapi.Message{
		MessageID: 7,
		Caption:   caption,
		Document:  &document,
		Chat:      &tgbotapi.Chat{ID: 42, Type: "private"},
		From:      &tgbotapi.User{ID: 42, FirstName: "tester"},
	}}
	ev, ok := b.toInboundEvent(context.Background(), u)
	if !ok {
		t.Fatal("toInboundEvent skipped document")
	}
	if ev.Kind != gateway.EventSubmit {
		t.Fatalf("Kind = %v, want submit", ev.Kind)
	}
	for _, att := range ev.Attachments {
		if att.URL != "" && filepath.Clean(att.URL) != att.URL {
			t.Fatalf("attachment URL = %q, want clean path", att.URL)
		}
	}
	return ev
}
