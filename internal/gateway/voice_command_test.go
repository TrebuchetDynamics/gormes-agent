package gateway

import (
	"context"
	"log/slog"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway/voicemode"
	"github.com/TrebuchetDynamics/gormes-agent/internal/kernel"
	"github.com/TrebuchetDynamics/gormes-agent/internal/llm"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tools"
)

func newVoiceCommandTestManager(t *testing.T) (*Manager, *fakeChannel, *voicemode.MemoryStore) {
	t.Helper()
	store := voicemode.NewMemoryStore()
	m := NewManagerWithSubmitter(ManagerConfig{VoiceModeStore: store}, &fakeKernel{}, slog.Default())
	return m, newFakeChannel("telegram"), store
}

func voiceTestEvent(text string) InboundEvent {
	return InboundEvent{
		Kind:     EventVoice,
		Platform: "telegram",
		ChatID:   "42",
		Text:     text,
	}
}

func lastSentText(t *testing.T, ch *fakeChannel) string {
	t.Helper()
	sent := ch.sentSnapshot()
	if len(sent) == 0 {
		t.Fatal("no outbound reply recorded")
	}
	return sent[len(sent)-1].Text
}

func TestVoiceCommandOnPersistsVoiceOnly(t *testing.T) {
	m, ch, store := newVoiceCommandTestManager(t)
	m.handleVoiceCommand(context.Background(), ch, voiceTestEvent("/voice on"))

	if got := lastSentText(t, ch); !strings.Contains(got, "Voice mode enabled (voice only)") {
		t.Fatalf("reply = %q, want voice-only confirmation", got)
	}
	mode, found, err := store.Get(context.Background(), "telegram:42")
	if err != nil || !found || mode != voicemode.ModeVoiceOnly {
		t.Fatalf("stored mode = (%q, %v, %v), want voice_only", mode, found, err)
	}
}

func TestVoiceCommandOffPersistsOff(t *testing.T) {
	m, ch, store := newVoiceCommandTestManager(t)
	if err := store.Set(context.Background(), "telegram:42", voicemode.ModeAll); err != nil {
		t.Fatalf("seed store: %v", err)
	}
	m.handleVoiceCommand(context.Background(), ch, voiceTestEvent("/voice off"))

	if got := lastSentText(t, ch); !strings.Contains(got, "Voice mode disabled") {
		t.Fatalf("reply = %q, want disabled confirmation", got)
	}
	mode, found, err := store.Get(context.Background(), "telegram:42")
	if err != nil || !found || mode != voicemode.ModeOff {
		t.Fatalf("stored mode = (%q, %v, %v), want off", mode, found, err)
	}
}

func TestVoiceCommandTTSEnablesAllReplies(t *testing.T) {
	m, ch, store := newVoiceCommandTestManager(t)
	m.handleVoiceCommand(context.Background(), ch, voiceTestEvent("/voice tts"))

	if got := lastSentText(t, ch); !strings.Contains(got, "Every reply will be spoken") {
		t.Fatalf("reply = %q, want all-replies confirmation", got)
	}
	mode, found, err := store.Get(context.Background(), "telegram:42")
	if err != nil || !found || mode != voicemode.ModeAll {
		t.Fatalf("stored mode = (%q, %v, %v), want all", mode, found, err)
	}
}

func TestVoiceCommandStatusReportsStoredMode(t *testing.T) {
	m, ch, store := newVoiceCommandTestManager(t)
	if err := store.Set(context.Background(), "telegram:42", voicemode.ModeVoiceOnly); err != nil {
		t.Fatalf("seed store: %v", err)
	}
	m.handleVoiceCommand(context.Background(), ch, voiceTestEvent("/voice status"))

	if got := lastSentText(t, ch); !strings.Contains(got, "Voice mode: voice only") {
		t.Fatalf("reply = %q, want status line", got)
	}
}

func TestVoiceCommandBareTogglesFromOff(t *testing.T) {
	m, ch, store := newVoiceCommandTestManager(t)
	m.handleVoiceCommand(context.Background(), ch, voiceTestEvent("/voice"))

	if got := lastSentText(t, ch); !strings.Contains(got, "Voice mode enabled (voice only)") {
		t.Fatalf("reply = %q, want toggle-on confirmation", got)
	}
	mode, _, err := store.Get(context.Background(), "telegram:42")
	if err != nil || mode != voicemode.ModeVoiceOnly {
		t.Fatalf("stored mode = (%q, %v), want voice_only", mode, err)
	}

	m.handleVoiceCommand(context.Background(), ch, voiceTestEvent("/voice"))
	mode, _, err = store.Get(context.Background(), "telegram:42")
	if err != nil || mode != voicemode.ModeOff {
		t.Fatalf("stored mode after second toggle = (%q, %v), want off", mode, err)
	}
}

func TestVoiceCommandInvalidArgumentShowsUsage(t *testing.T) {
	m, ch, _ := newVoiceCommandTestManager(t)
	m.handleVoiceCommand(context.Background(), ch, voiceTestEvent("/voice maybe"))

	if got := lastSentText(t, ch); !strings.Contains(got, "Usage: /voice [on|off|tts|status]") {
		t.Fatalf("reply = %q, want usage guidance", got)
	}
}

func TestVoiceCommandIsRecognizedGatewayCommand(t *testing.T) {
	def, ok := ResolveCommand("voice")
	if !ok {
		t.Fatal("voice is not registered in the gateway command registry")
	}
	if def.Kind != EventVoice {
		t.Fatalf("voice kind = %v, want EventVoice", def.Kind)
	}
	if def.ActiveTurnPolicy == CommandActiveTurnPolicyUnavailable {
		t.Fatal("voice is still marked unavailable")
	}
}

func TestVoiceModeOffSuppressesAutoTTS(t *testing.T) {
	reg := tools.NewRegistry()
	tts := &fakeGatewayTTSTool{}
	reg.MustRegister(tts)
	store := voicemode.NewMemoryStore()
	if err := store.Set(context.Background(), "telegram:42", voicemode.ModeOff); err != nil {
		t.Fatalf("seed store: %v", err)
	}
	m := NewManagerWithSubmitter(ManagerConfig{ToolRegistry: reg, VoiceModeStore: store}, &fakeKernel{}, slog.Default())
	frame := kernel.RenderFrame{
		Phase:   kernel.PhaseIdle,
		History: []llm.Message{{Role: "assistant", Content: "Spoken candidate."}},
	}

	_, media := m.formatFinalDeliveryPagesForTurn(context.Background(), "telegram", frame, "telegram:42", true)
	if len(media) != 0 {
		t.Fatalf("media = %#v, want none while voice mode is off", media)
	}
	if tts.text != "" {
		t.Fatalf("tts invoked for %q while voice mode is off", tts.text)
	}
}

func TestVoiceModeAllSpeaksNonAudioTurns(t *testing.T) {
	reg := tools.NewRegistry()
	tts := &fakeGatewayTTSTool{}
	reg.MustRegister(tts)
	store := voicemode.NewMemoryStore()
	if err := store.Set(context.Background(), "telegram:42", voicemode.ModeAll); err != nil {
		t.Fatalf("seed store: %v", err)
	}
	m := NewManagerWithSubmitter(ManagerConfig{ToolRegistry: reg, VoiceModeStore: store}, &fakeKernel{}, slog.Default())
	frame := kernel.RenderFrame{
		Phase:   kernel.PhaseIdle,
		History: []llm.Message{{Role: "assistant", Content: "Spoken candidate."}},
	}

	_, media := m.formatFinalDeliveryPagesForTurn(context.Background(), "telegram", frame, "telegram:42", false)
	if len(media) != 1 {
		t.Fatalf("media = %#v, want one spoken attachment in all mode", media)
	}
	if tts.text != "Spoken candidate." {
		t.Fatalf("tts text = %q", tts.text)
	}
}

func TestVoiceModeVoiceOnlySpeaksAudioTurnsOnly(t *testing.T) {
	reg := tools.NewRegistry()
	tts := &fakeGatewayTTSTool{}
	reg.MustRegister(tts)
	store := voicemode.NewMemoryStore()
	if err := store.Set(context.Background(), "telegram:42", voicemode.ModeVoiceOnly); err != nil {
		t.Fatalf("seed store: %v", err)
	}
	m := NewManagerWithSubmitter(ManagerConfig{ToolRegistry: reg, VoiceModeStore: store}, &fakeKernel{}, slog.Default())
	frame := kernel.RenderFrame{
		Phase:   kernel.PhaseIdle,
		History: []llm.Message{{Role: "assistant", Content: "Spoken candidate."}},
	}

	if _, media := m.formatFinalDeliveryPagesForTurn(context.Background(), "telegram", frame, "telegram:42", false); len(media) != 0 {
		t.Fatalf("media = %#v, want none for a typed turn in voice_only mode", media)
	}
	if _, media := m.formatFinalDeliveryPagesForTurn(context.Background(), "telegram", frame, "telegram:42", true); len(media) != 1 {
		t.Fatalf("media = %#v, want one attachment for an audio turn in voice_only mode", media)
	}
}
