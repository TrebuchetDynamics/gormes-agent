package discord

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/bwmarrin/discordgo"

	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway"
)

var (
	testDiscordPNG = append([]byte("\x89PNG\r\n\x1a\n"), bytes.Repeat([]byte{0}, 64)...)
	testDiscordOGG = append([]byte("OggS"), bytes.Repeat([]byte{0}, 64)...)
	testDiscordPDF = []byte("%PDF-1.4\nfake pdf\n%%EOF")
)

func TestDiscordAttachmentPrefersAuthenticatedBytesAndPreservesMetadata(t *testing.T) {
	ms := newMockSession()
	cacheDir := t.TempDir()
	ms.attachmentBytes["img-1"] = testDiscordPNG
	ms.attachmentBytes["audio-1"] = testDiscordOGG
	ms.attachmentBytes["doc-1"] = testDiscordPDF
	var fallbackCalls atomic.Int32

	b := New(Config{
		AllowedChannelID:   "parent-42",
		AttachmentCacheDir: cacheDir,
		AttachmentHTTPClient: &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
			fallbackCalls.Add(1)
			return nil, errUnexpectedDiscordFallback
		})},
	}, ms, nil)
	b.rememberThread(&discordgo.Channel{
		ID:       "thread-99",
		Type:     discordgo.ChannelTypeGuildPublicThread,
		ParentID: "parent-42",
		Name:     "incident",
		GuildID:  "guild-7",
	})
	inbox, cancel, done := runDiscordBot(t, b)
	defer stopDiscordBot(cancel, done)

	ms.deliver(&discordgo.MessageCreate{Message: &discordgo.Message{
		ID:        "msg-100",
		ChannelID: "thread-99",
		GuildID:   "guild-7",
		Content:   "please inspect",
		Author:    &discordgo.User{ID: "user-5", Bot: false},
		Attachments: []*discordgo.MessageAttachment{
			{ID: "img-1", URL: "https://cdn.discordapp.com/attachments/x/photo.png", Filename: "photo.png", ContentType: "image/png", Size: len(testDiscordPNG)},
			{ID: "audio-1", URL: "https://cdn.discordapp.com/attachments/x/voice.ogg", Filename: "voice.ogg", ContentType: "audio/ogg", Size: len(testDiscordOGG)},
			{ID: "doc-1", URL: "https://cdn.discordapp.com/attachments/x/report.pdf", Filename: "report.pdf", ContentType: "application/pdf", Size: len(testDiscordPDF)},
		},
	}})

	ev := receiveDiscordEvent(t, inbox)
	if ev.Platform != "discord" || ev.ChatID != "parent-42" || ev.ThreadID != "thread-99" || ev.ParentChatID != "parent-42" || ev.GuildID != "guild-7" || ev.MessageID != "msg-100" {
		t.Fatalf("metadata = %+v, want forum parent/thread/guild/message preserved", ev)
	}
	if ev.Text != "please inspect" {
		t.Fatalf("Text = %q, want original content", ev.Text)
	}
	if strings.Contains(ev.SubmitText(), "cdn.discordapp.com") {
		t.Fatalf("SubmitText leaks raw CDN URL: %q", ev.SubmitText())
	}
	if fallbackCalls.Load() != 0 {
		t.Fatalf("fallback HTTP calls = %d, want 0 when authenticated bytes are available", fallbackCalls.Load())
	}
	assertDiscordAttachment(t, ev.Attachments, 0, "image", "photo.png", "image/png", "img-1", len(testDiscordPNG), cacheDir)
	assertDiscordAttachment(t, ev.Attachments, 1, "audio", "voice.ogg", "audio/ogg", "audio-1", len(testDiscordOGG), cacheDir)
	assertDiscordAttachment(t, ev.Attachments, 2, "document", "report.pdf", "application/pdf", "doc-1", len(testDiscordPDF), cacheDir)
}

func TestDiscordAttachmentSafeFallbackCachesDocument(t *testing.T) {
	ms := newMockSession()
	ms.attachmentErr = errMockDiscordAttachmentReadUnavailable
	cacheDir := t.TempDir()
	var fallbackCalls atomic.Int32
	b := New(Config{
		AllowedChannelID:   "42",
		AttachmentCacheDir: cacheDir,
		AttachmentHTTPClient: &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			fallbackCalls.Add(1)
			if req.URL.Host != "cdn.discordapp.com" {
				t.Fatalf("fallback host = %q, want Discord CDN", req.URL.Host)
			}
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     http.Header{"Content-Type": []string{"application/pdf"}},
				Body:       io.NopCloser(bytes.NewReader(testDiscordPDF)),
			}, nil
		})},
	}, ms, nil)
	inbox, cancel, done := runDiscordBot(t, b)
	defer stopDiscordBot(cancel, done)

	ms.deliver(discordMessageWithAttachment("fallback-doc", "summarize", discordAttachment("doc-safe", "report.pdf", "application/pdf", len(testDiscordPDF), "https://cdn.discordapp.com/attachments/x/report.pdf")))

	ev := receiveDiscordEvent(t, inbox)
	if fallbackCalls.Load() != 1 {
		t.Fatalf("fallback HTTP calls = %d, want 1", fallbackCalls.Load())
	}
	assertDiscordAttachment(t, ev.Attachments, 0, "document", "report.pdf", "application/pdf", "doc-safe", len(testDiscordPDF), cacheDir)
	if !strings.Contains(ev.Text, "summarize") {
		t.Fatalf("Text = %q, want content preserved", ev.Text)
	}
}

func TestDiscordAttachmentUnsafeFallbackIsBlockedWithoutHTTP(t *testing.T) {
	ms := newMockSession()
	ms.attachmentErr = errMockDiscordAttachmentReadUnavailable
	var fallbackCalls atomic.Int32
	b := New(Config{
		AllowedChannelID:   "42",
		AttachmentCacheDir: t.TempDir(),
		AttachmentHTTPClient: &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
			fallbackCalls.Add(1)
			return nil, errUnexpectedDiscordFallback
		})},
	}, ms, nil)
	inbox, cancel, done := runDiscordBot(t, b)
	defer stopDiscordBot(cancel, done)

	ms.deliver(discordMessageWithAttachment("unsafe-doc", "", discordAttachment("doc-unsafe", "report.pdf", "application/pdf", 100, "http://169.254.169.254/latest/meta-data")))

	ev := receiveDiscordEvent(t, inbox)
	if fallbackCalls.Load() != 0 {
		t.Fatalf("fallback HTTP calls = %d, want unsafe URL blocked before HTTP", fallbackCalls.Load())
	}
	if len(ev.Attachments) != 0 {
		t.Fatalf("attachments = %#v, want no unsafe cached path", ev.Attachments)
	}
	if !strings.Contains(ev.Text, "discord_attachment_blocked_ssrf") {
		t.Fatalf("Text = %q, want SSRF evidence", ev.Text)
	}
	if strings.Contains(ev.Text, "169.254.169.254") {
		t.Fatalf("Text leaks unsafe URL: %q", ev.Text)
	}
}

func TestDiscordAttachmentRejectsHTMLFallbackPayload(t *testing.T) {
	ms := newMockSession()
	ms.attachmentErr = errMockDiscordAttachmentReadUnavailable
	b := New(Config{
		AllowedChannelID:   "42",
		AttachmentCacheDir: t.TempDir(),
		AttachmentHTTPClient: &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     http.Header{"Content-Type": []string{"text/html"}},
				Body:       io.NopCloser(strings.NewReader("<html>forbidden</html>")),
			}, nil
		})},
	}, ms, nil)
	inbox, cancel, done := runDiscordBot(t, b)
	defer stopDiscordBot(cancel, done)

	ms.deliver(discordMessageWithAttachment("html-image", "", discordAttachment("img-html", "photo.png", "image/png", 64, "https://cdn.discordapp.com/attachments/x/photo.png")))

	ev := receiveDiscordEvent(t, inbox)
	if len(ev.Attachments) != 0 {
		t.Fatalf("attachments = %#v, want HTML/error payload rejected", ev.Attachments)
	}
	if !strings.Contains(ev.Text, "discord_attachment_unavailable") {
		t.Fatalf("Text = %q, want unavailable evidence", ev.Text)
	}
}

func TestDiscordAttachmentTextDocumentInjectionIsBounded(t *testing.T) {
	ms := newMockSession()
	cacheDir := t.TempDir()
	ms.attachmentBytes["txt-small"] = []byte("alpha\nbeta\n")
	ms.attachmentBytes["txt-large"] = bytes.Repeat([]byte("x"), 100*1024+1)
	b := New(Config{AllowedChannelID: "42", AttachmentCacheDir: cacheDir}, ms, nil)
	inbox, cancel, done := runDiscordBot(t, b)
	defer stopDiscordBot(cancel, done)

	ms.deliver(&discordgo.MessageCreate{Message: &discordgo.Message{
		ID:        "msg-text",
		ChannelID: "42",
		Content:   "summarize",
		Author:    &discordgo.User{ID: "user-5", Bot: false},
		Attachments: []*discordgo.MessageAttachment{
			discordAttachment("txt-small", "notes.txt", "text/plain", len(ms.attachmentBytes["txt-small"]), "https://cdn.discordapp.com/attachments/x/notes.txt"),
			discordAttachment("txt-large", "big.txt", "text/plain", len(ms.attachmentBytes["txt-large"]), "https://cdn.discordapp.com/attachments/x/big.txt"),
		},
	}})

	ev := receiveDiscordEvent(t, inbox)
	smallIdx := strings.Index(ev.Text, "[Content of notes.txt]:\nalpha\nbeta")
	captionIdx := strings.Index(ev.Text, "summarize")
	if smallIdx < 0 || captionIdx < 0 || smallIdx > captionIdx {
		t.Fatalf("Text = %q, want small text content before message text", ev.Text)
	}
	if strings.Contains(ev.Text, "[Content of big.txt]") {
		t.Fatalf("Text = %q, want over-cap text cached without raw injection", ev.Text)
	}
	assertDiscordAttachment(t, ev.Attachments, 0, "document", "notes.txt", "text/plain", "txt-small", len(ms.attachmentBytes["txt-small"]), cacheDir)
	assertDiscordAttachment(t, ev.Attachments, 1, "document", "big.txt", "text/plain", "txt-large", len(ms.attachmentBytes["txt-large"]), cacheDir)
}

func TestDiscordAttachmentOversizedSkipsDownloadWithEvidence(t *testing.T) {
	ms := newMockSession()
	b := New(Config{AllowedChannelID: "42", AttachmentCacheDir: t.TempDir()}, ms, nil)
	inbox, cancel, done := runDiscordBot(t, b)
	defer stopDiscordBot(cancel, done)

	ms.deliver(discordMessageWithAttachment("huge-doc", "", discordAttachment("doc-huge", "huge.pdf", "application/pdf", 33*1024*1024, "https://cdn.discordapp.com/attachments/x/huge.pdf")))

	ev := receiveDiscordEvent(t, inbox)
	if got := ms.attachmentReadCount(); got != 0 {
		t.Fatalf("authenticated reads = %d, want no download for oversized attachment", got)
	}
	if len(ev.Attachments) != 0 {
		t.Fatalf("attachments = %#v, want none", ev.Attachments)
	}
	if !strings.Contains(ev.Text, "discord_attachment_too_large") {
		t.Fatalf("Text = %q, want too-large evidence", ev.Text)
	}
}

func runDiscordBot(t *testing.T, b *Bot) (<-chan gateway.InboundEvent, context.CancelFunc, <-chan struct{}) {
	t.Helper()
	inbox := make(chan gateway.InboundEvent, 8)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		_ = b.Run(ctx, inbox)
		close(done)
	}()
	time.Sleep(10 * time.Millisecond)
	return inbox, cancel, done
}

func stopDiscordBot(cancel context.CancelFunc, done <-chan struct{}) {
	cancel()
	<-done
}

func receiveDiscordEvent(t *testing.T, inbox <-chan gateway.InboundEvent) gateway.InboundEvent {
	t.Helper()
	select {
	case ev := <-inbox:
		return ev
	case <-time.After(300 * time.Millisecond):
		t.Fatal("no inbound event")
		return gateway.InboundEvent{}
	}
}

func discordMessageWithAttachment(id, content string, attachment *discordgo.MessageAttachment) *discordgo.MessageCreate {
	return &discordgo.MessageCreate{Message: &discordgo.Message{
		ID:          id,
		ChannelID:   "42",
		Content:     content,
		Author:      &discordgo.User{ID: "user-5", Bot: false},
		Attachments: []*discordgo.MessageAttachment{attachment},
	}}
}

func discordAttachment(id, fileName, mediaType string, size int, url string) *discordgo.MessageAttachment {
	return &discordgo.MessageAttachment{
		ID:          id,
		URL:         url,
		Filename:    fileName,
		ContentType: mediaType,
		Size:        size,
	}
}

func assertDiscordAttachment(t *testing.T, attachments []gateway.Attachment, index int, kind, fileName, mediaType, sourceID string, size int, cacheDir string) {
	t.Helper()
	if len(attachments) <= index {
		t.Fatalf("attachments = %#v, want index %d", attachments, index)
	}
	att := attachments[index]
	if att.Kind != kind || att.FileName != fileName || att.MediaType != mediaType || att.SourceID != sourceID || att.SizeBytes != int64(size) {
		t.Fatalf("attachment[%d] = %#v, want kind=%s file=%s media=%s source=%s size=%d", index, att, kind, fileName, mediaType, sourceID, size)
	}
	if !strings.HasPrefix(att.URL, cacheDir+string(os.PathSeparator)) {
		t.Fatalf("attachment[%d].URL = %q, want under %q", index, att.URL, cacheDir)
	}
	if filepath.Clean(att.URL) != att.URL {
		t.Fatalf("attachment[%d].URL = %q, want clean path", index, att.URL)
	}
	if _, err := os.Stat(att.URL); err != nil {
		t.Fatalf("cached attachment path %q: %v", att.URL, err)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

var errUnexpectedDiscordFallback = &discordAttachmentTestError{"unexpected fallback HTTP call"}

type discordAttachmentTestError struct{ msg string }

func (e *discordAttachmentTestError) Error() string { return e.msg }
