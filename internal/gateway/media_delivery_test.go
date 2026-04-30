package gateway

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/hermes"
	"github.com/TrebuchetDynamics/gormes-agent/internal/kernel"
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

func TestManagerFinalResponseDeliversMediaWithoutLeakingTag(t *testing.T) {
	audioPath := filepath.Join(t.TempDir(), "voice.ogg")
	if err := os.WriteFile(audioPath, []byte("audio"), 0o600); err != nil {
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
		Platform: "telegram", ChatID: "42", MsgID: "99",
		Kind: EventSubmit, Text: "speak this",
	})
	waitFor(t, 200*time.Millisecond, func() bool {
		return len(fk.submitsSnapshot()) == 1
	})

	frames <- kernel.RenderFrame{
		Phase: kernel.PhaseIdle,
		History: []hermes.Message{
			{Role: "user", Content: "speak this"},
			{Role: "assistant", Content: "Here is the audio.\n[[audio_as_voice]]\nMEDIA:" + audioPath},
		},
	}

	waitFor(t, 500*time.Millisecond, func() bool {
		return len(tg.sentSnapshot()) >= 1 && len(tg.mediaSnapshot()) == 1
	})
	sent := tg.sentSnapshot()
	if strings.Contains(sent[len(sent)-1].Text, "MEDIA:") || strings.Contains(sent[len(sent)-1].Text, "audio_as_voice") {
		t.Fatalf("final text leaked media tag: %#v", sent)
	}
	media := tg.mediaSnapshot()[0]
	if media.ChatID != "42" || media.ReplyToMsgID != "99" || media.Media.Path != audioPath || !media.Media.AsVoice {
		t.Fatalf("media send = %+v, want reply-threaded voice attachment", media)
	}
}
