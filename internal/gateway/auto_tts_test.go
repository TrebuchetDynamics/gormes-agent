package gateway

import (
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/hermes"
	"github.com/TrebuchetDynamics/gormes-agent/internal/kernel"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tools"
)

type fakeGatewayTTSTool struct {
	text string
}

func (f *fakeGatewayTTSTool) Name() string        { return "text_to_speech" }
func (f *fakeGatewayTTSTool) Description() string { return "fake tts" }
func (f *fakeGatewayTTSTool) Schema() json.RawMessage {
	return json.RawMessage(`{"type":"object","properties":{"text":{"type":"string"}}}`)
}
func (f *fakeGatewayTTSTool) Timeout() time.Duration { return time.Second }
func (f *fakeGatewayTTSTool) Execute(_ context.Context, args json.RawMessage) (json.RawMessage, error) {
	var in struct {
		Text string `json:"text"`
	}
	if err := json.Unmarshal(args, &in); err != nil {
		return nil, err
	}
	f.text = in.Text
	return json.Marshal(map[string]any{
		"success":   true,
		"media_tag": "MEDIA:/tmp/gormes-auto-tts.mp3",
		"evidence":  "tts_synthesized",
		"provider":  "edge",
	})
}

func TestInboundRequestsAudioReply(t *testing.T) {
	if !inboundRequestsAudioReply(InboundEvent{Attachments: []Attachment{{Kind: "voice"}}}) {
		t.Fatal("voice attachment did not request audio reply")
	}
	if !inboundRequestsAudioReply(InboundEvent{Text: "I cannot read right now, send audio too"}) {
		t.Fatal("cannot-read text did not request audio reply")
	}
	if !inboundRequestsAudioReply(InboundEvent{Text: "Mandamelo en audio, por favor"}) {
		t.Fatal("Spanish audio request did not request audio reply")
	}
	if inboundRequestsAudioReply(InboundEvent{Text: "plain written answer is fine"}) {
		t.Fatal("plain text unexpectedly requested audio reply")
	}
}

func TestManagerFinalDeliveryAddsTTSMediaForAudioRequestedTurn(t *testing.T) {
	reg := tools.NewRegistry()
	tts := &fakeGatewayTTSTool{}
	reg.MustRegister(tts)
	m := NewManagerWithSubmitter(ManagerConfig{ToolRegistry: reg}, &fakeKernel{}, slog.Default())
	frame := kernel.RenderFrame{
		Phase: kernel.PhaseIdle,
		History: []hermes.Message{{
			Role:    "assistant",
			Content: "Written answer for the operator.",
		}},
	}

	pages, media := m.formatFinalDeliveryPagesForTurn(context.Background(), "telegram", frame, "telegram:42", true)

	if len(pages) != 1 || pages[0] == "" {
		t.Fatalf("pages = %#v, want one written final page", pages)
	}
	if len(media) != 1 {
		t.Fatalf("media = %#v, want one synthesized audio attachment", media)
	}
	if media[0].Path != "/tmp/gormes-auto-tts.mp3" || media[0].Kind != OutboundMediaKindAudio {
		t.Fatalf("media[0] = %#v, want audio attachment from TTS MEDIA tag", media[0])
	}
	if tts.text != "Written answer for the operator." {
		t.Fatalf("tts text = %q, want raw final assistant text", tts.text)
	}
}

func TestManagerFinalDeliveryAddsTTSMediaWhenSessionTTSEnabled(t *testing.T) {
	reg := tools.NewRegistry()
	reg.MustRegister(&fakeGatewayTTSTool{})
	m := NewManagerWithSubmitter(ManagerConfig{ToolRegistry: reg}, &fakeKernel{}, slog.Default())
	m.setTTSConfig("telegram:42", TTSConfig{Enabled: true})
	frame := kernel.RenderFrame{
		Phase: kernel.PhaseIdle,
		History: []hermes.Message{{
			Role:    "assistant",
			Content: "Read this response aloud.",
		}},
	}

	_, media := m.formatFinalDeliveryPagesForTurn(context.Background(), "telegram", frame, "telegram:42", false)

	if len(media) != 1 {
		t.Fatalf("media = %#v, want /tts on to synthesize audio", media)
	}
}

func TestTTSOnSelectsUsableDefaultEngine(t *testing.T) {
	ch := newFakeChannel("telegram")
	m := NewManagerWithSubmitter(ManagerConfig{}, &fakeKernel{}, slog.Default())

	m.handleTTSCommand(context.Background(), ch, InboundEvent{
		Platform: "telegram",
		ChatID:   "42",
		Text:     "/tts on",
	})

	cfg := m.getTTSConfig("telegram:42")
	if !cfg.Enabled || cfg.Engine != TTSEngineEdge {
		t.Fatalf("cfg = %+v, want enabled Edge TTS", cfg)
	}
}

func TestManagerAudioRequestedTurnAddsDeliveryGuidance(t *testing.T) {
	tg := newFakeChannel("telegram")
	fk := &fakeKernel{}
	m := NewManagerWithSubmitter(ManagerConfig{
		AllowedChats: map[string]string{"telegram": "42"},
	}, fk, slog.Default())
	if err := m.Register(tg); err != nil {
		t.Fatalf("Register: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { _ = m.Run(ctx) }()

	tg.pushInbound(InboundEvent{
		Platform: "telegram",
		ChatID:   "42",
		UserID:   "42",
		MsgID:    "10",
		Kind:     EventSubmit,
		Text:     "I cannot read right now",
	})

	waitFor(t, 200*time.Millisecond, func() bool {
		return len(fk.submitsSnapshot()) == 1
	})
	got := fk.submitsSnapshot()[0].SessionContext
	for _, want := range []string{
		"Gateway audio delivery is enabled for this turn.",
		"Do not claim that you generated audio yourself",
		"do not claim the TTS provider is unavailable",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("SessionContext missing %q:\n%s", want, got)
		}
	}
}
