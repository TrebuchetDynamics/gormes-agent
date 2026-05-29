package gateway

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/kernel"
	"github.com/TrebuchetDynamics/gormes-agent/internal/llm"
)

func TestPrepareMediaDeliveryContentExtractsHermesTags(t *testing.T) {
	audioPath := filepath.Join(t.TempDir(), "voice.ogg")
	if err := os.WriteFile(audioPath, []byte("audio"), 0o600); err != nil {
		t.Fatal(err)
	}

	content := PrepareMediaDeliveryContent("Here is the audio.\n[[audio_as_voice]]\nMEDIA:" + audioPath + "\nDone.")
	if strings.Contains(content.Text, "MEDIA:") || strings.Contains(content.Text, "audio_as_voice") {
		t.Fatalf("cleaned text leaked media tag: %q", content.Text)
	}
	if content.Text != "Here is the audio.\nDone." {
		t.Fatalf("Text = %q, want media line stripped", content.Text)
	}
	if len(content.Media) != 1 || content.Media[0].Path != audioPath || !content.Media[0].AsVoice {
		t.Fatalf("Media = %+v, want one voice attachment", content.Media)
	}
}

func TestPrepareMediaDeliveryContentExtractsImageDocumentVideoTags(t *testing.T) {
	dir := t.TempDir()
	cases := []string{
		"photo.png",
		"photo.jpg",
		"photo.jpeg",
		"preview.webp",
		"report.pdf",
		"data.csv",
		"notes.txt",
		"bundle.zip",
		"clip.mp4",
	}
	for _, name := range cases {
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(dir, name)
			if err := os.WriteFile(path, []byte("media"), 0o600); err != nil {
				t.Fatal(err)
			}

			content := PrepareMediaDeliveryContent("Here is a file.\nMEDIA:" + path + "\nDone.")
			if strings.Contains(content.Text, "MEDIA:") {
				t.Fatalf("cleaned text leaked media tag: %q", content.Text)
			}
			if content.Text != "Here is a file.\nDone." {
				t.Fatalf("Text = %q, want media line stripped", content.Text)
			}
			if len(content.Media) != 1 || content.Media[0].Path != path {
				t.Fatalf("Media = %+v, want one extracted media path %q", content.Media, path)
			}
		})
	}
}

func TestPrepareMediaDeliveryContentPreservesMixedMediaOrder(t *testing.T) {
	dir := t.TempDir()
	paths := []string{
		fileWithContent(t, dir, "voice.ogg", "audio"),
		fileWithContent(t, dir, "photo.png", "image"),
		fileWithContent(t, dir, "report.pdf", "pdf"),
		fileWithContent(t, dir, "clip.mp4", "video"),
	}

	content := PrepareMediaDeliveryContent("Files:\n[[audio_as_voice]]\nMEDIA:" + paths[0] + "\nMEDIA:" + paths[1] + "\nMEDIA:" + paths[2] + "\nMEDIA:" + paths[3] + "\nDone.")
	if content.Text != "Files:\nDone." {
		t.Fatalf("Text = %q, want all media lines stripped", content.Text)
	}
	if len(content.Media) != len(paths) {
		t.Fatalf("Media len = %d, want %d: %+v", len(content.Media), len(paths), content.Media)
	}
	for i, path := range paths {
		if content.Media[i].Path != path {
			t.Fatalf("Media[%d].Path = %q, want %q; all=%+v", i, content.Media[i].Path, path, content.Media)
		}
	}
	if !content.Media[0].AsVoice {
		t.Fatalf("Media[0].AsVoice = false, want voice marker preserved")
	}
}

func TestPrepareMediaDeliveryContentRejectsUnsafeMediaTags(t *testing.T) {
	content := PrepareMediaDeliveryContent("listen MEDIA:../plain.txt")
	if len(content.Media) != 0 {
		t.Fatalf("Media = %+v, want unsafe path ignored", content.Media)
	}
	if !strings.Contains(content.Text, "[MEDIA:redacted]") {
		t.Fatalf("Text = %q, want redacted marker for unsafe media", content.Text)
	}
	if len(content.Evidence) != 1 || content.Evidence[0].Code != MediaDeliveryEvidenceIgnored || content.Evidence[0].Target != "[redacted]" {
		t.Fatalf("Evidence = %+v, want redacted ignored evidence", content.Evidence)
	}
}

func TestManagerMediaFallbackDoesNotLeakLocalPath(t *testing.T) {
	ch := &sendOnlyChannel{name: "fallback"}
	m := NewManagerWithSubmitter(ManagerConfig{}, &fakeKernel{}, slog.Default())
	m.deliverMedia(context.Background(), ch, "42", "99", "", []OutboundMedia{{Path: "/tmp/private/voice.ogg", AsVoice: true}})
	if len(ch.sent) != 1 {
		t.Fatalf("sent count = %d, want one fallback message", len(ch.sent))
	}
	if strings.Contains(ch.sent[0], "/tmp/private") || strings.Contains(ch.sent[0], "voice.ogg") {
		t.Fatalf("fallback leaked local path: %q", ch.sent[0])
	}
	if ch.sent[0] != "Media attachment unavailable." {
		t.Fatalf("fallback text = %q", ch.sent[0])
	}
}

type sendOnlyChannel struct {
	name string
	sent []string
}

func (c *sendOnlyChannel) Name() string                                   { return c.name }
func (c *sendOnlyChannel) Run(context.Context, chan<- InboundEvent) error { return nil }
func (c *sendOnlyChannel) Send(_ context.Context, _ string, text string) (string, error) {
	c.sent = append(c.sent, text)
	return "1", nil
}

func TestManagerFinalResponseDeliversMediaWithoutLeakingTag(t *testing.T) {
	audioPath := filepath.Join(t.TempDir(), "voice.ogg")
	if err := os.WriteFile(audioPath, []byte("audio"), 0o600); err != nil {
		t.Fatal(err)
	}
	photoPath := filepath.Join(t.TempDir(), "photo.png")
	if err := os.WriteFile(photoPath, []byte("image"), 0o600); err != nil {
		t.Fatal(err)
	}
	reportPath := filepath.Join(t.TempDir(), "report.pdf")
	if err := os.WriteFile(reportPath, []byte("pdf"), 0o600); err != nil {
		t.Fatal(err)
	}

	tg := newFakeChannel("telegram")
	frames := make(chan kernel.RenderFrame, 8)
	fk := &fakeKernel{}
	m := NewManagerWithSubmitter(ManagerConfig{
		AllowedChats: map[string]string{"telegram": "42"},
		CoalesceMs:   10,
	}, fk, slog.Default())
	m.setRenderChan(frames)
	_ = m.Register(tg)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { _ = m.Run(ctx) }()

	tg.pushInbound(InboundEvent{
		Platform: "telegram", ChatID: "42", ThreadID: "topic-7", MsgID: "99",
		Kind: EventSubmit, Text: "speak this",
	})
	waitFor(t, 200*time.Millisecond, func() bool {
		return len(fk.submitsSnapshot()) == 1
	})

	frames <- kernel.RenderFrame{
		Phase: kernel.PhaseIdle,
		History: []llm.Message{
			{Role: "user", Content: "speak this"},
			{Role: "assistant", Content: "Here are the files.\n[[audio_as_voice]]\nMEDIA:" + audioPath + "\nMEDIA:" + photoPath + "\nMEDIA:" + reportPath},
		},
	}

	waitFor(t, 500*time.Millisecond, func() bool {
		return len(tg.sentSnapshot()) >= 1 && len(tg.mediaSnapshot()) == 3
	})
	sent := tg.sentSnapshot()
	if strings.Contains(sent[len(sent)-1].Text, "MEDIA:") || strings.Contains(sent[len(sent)-1].Text, "audio_as_voice") {
		t.Fatalf("final text leaked media tag: %#v", sent)
	}
	media := tg.mediaSnapshot()
	wantPaths := []string{audioPath, photoPath, reportPath}
	for i, wantPath := range wantPaths {
		if media[i].ChatID != "42" || media[i].ReplyToMsgID != "99" || media[i].Media.ThreadID != "topic-7" || media[i].Media.Path != wantPath {
			t.Fatalf("media[%d] = %+v, want reply-threaded attachment path %q", i, media[i], wantPath)
		}
	}
	if !media[0].Media.AsVoice {
		t.Fatalf("media[0].AsVoice = false, want voice attachment")
	}
}

func fileWithContent(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}
