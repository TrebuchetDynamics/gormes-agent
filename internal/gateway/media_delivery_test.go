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
