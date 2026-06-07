package ttsconfig

import (
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway/gatewaytest"
)

func TestResolveSpeedAliases(t *testing.T) {
	for _, raw := range []string{"very-fast", "veryfast", "very_fast", "very fast", " VERY FAST "} {
		got, ok := ResolveSpeed(raw)
		if !ok || got != SpeedVeryFast {
			t.Fatalf("ResolveSpeed(%q) = %q, %v; want very-fast, true", raw, got, ok)
		}
	}
	if _, ok := ResolveSpeed("warp"); ok {
		t.Fatal("ResolveSpeed(warp) unexpectedly succeeded")
	}
}

func TestDefaultVoiceForEngine(t *testing.T) {
	if got := DefaultVoiceForEngine(EngineEdge); got != "en-US-AriaNeural" {
		t.Fatalf("edge default voice = %q", got)
	}
	if got := DefaultVoiceForEngine(EngineLocal); got != "default" {
		t.Fatalf("local default voice = %q", got)
	}
}

func TestSortedVoiceListingDoesNotChangeDefaultVoice(t *testing.T) {
	before := DefaultVoiceForEngine(EngineElevenLabs)
	voices := VoicesForEngineSorted(EngineElevenLabs)
	if len(voices) == 0 {
		t.Fatal("expected listed elevenlabs voices")
	}
	voices[0] = "mutated-test-voice"
	if got := DefaultVoiceForEngine(EngineElevenLabs); got != before {
		t.Fatalf("default elevenlabs voice changed after sorted listing: got %q want %q", got, before)
	}
}

func TestConfigString(t *testing.T) {
	got := DefaultConfig.String()
	gatewaytest.AssertContainsAll(t, got, "TTS: enabled", "engine: edge", "voice: en-US-AriaNeural", "speed: normal", "language: auto")
}
