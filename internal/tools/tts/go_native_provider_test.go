//go:build !slim

package tts

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	speechtts "github.com/TrebuchetDynamics/gormes-agent/internal/speech/tts"
)

func TestGoNativeTTSProviderSynthesizesFixtureWAV(t *testing.T) {
	restoreEnv(t, "GORMES_TTS_PIPER_MODEL", "GORMES_TTS_PIPER_MODEL_CACHE", "GORMES_TTS_PIPER_CACHE_DIR", "GORMES_TTS_PIPER_VOICE", "GORMES_TTS_PIPER_BIN")
	output := filepath.Join(t.TempDir(), "local.wav")
	provider := NewGoNativeTTSProvider(GoNativeTTSProviderConfig{})

	result := NewTTSRunner(TTSConfig{Provider: ProviderNameLocalGo}, map[string]TTSProvider{
		ProviderNameLocalGo: provider,
		"edge":              &fakeTTSProvider{available: true},
	}).Synthesize(context.Background(), TTSRequest{Text: "hello local speech", OutputPath: output})

	if !result.Success || result.Evidence != TTSEvidenceOK || result.Provider != ProviderNameLocalGo {
		t.Fatalf("result = %+v, want local_go success", result)
	}
	data, err := os.ReadFile(output)
	if err != nil {
		t.Fatalf("read output: %v", err)
	}
	if string(data[:4]) != "RIFF" || string(data[8:12]) != "WAVE" {
		t.Fatalf("output is not a valid WAV header: %q %q", data[:4], data[8:12])
	}
	if result.MediaTag != "MEDIA:"+output {
		t.Fatalf("media tag = %q, want MEDIA path", result.MediaTag)
	}
}

func TestGoNativeTTSProviderUsesConfiguredPiperRuntime(t *testing.T) {
	restoreEnv(t, "GORMES_TTS_PIPER_MODEL", "GORMES_TTS_PIPER_MODEL_CACHE", "GORMES_TTS_PIPER_CACHE_DIR", "GORMES_TTS_PIPER_VOICE", "GORMES_TTS_PIPER_BIN")
	os.Setenv("GORMES_TTS_PIPER_MODEL", filepath.Join(t.TempDir(), "voice.onnx"))
	os.Setenv("GORMES_TTS_PIPER_BIN", "piper-test")

	provider := NewGoNativeTTSProvider(GoNativeTTSProviderConfig{})
	if _, ok := provider.RuntimeForTest().(*speechtts.PiperSynthesizer); !ok {
		t.Fatalf("runtime = %T, want PiperSynthesizer when model env is configured", provider.RuntimeForTest())
	}
}

func TestGoNativeTTSProviderUsesCachedPiperRuntime(t *testing.T) {
	restoreEnv(t, "GORMES_TTS_PIPER_MODEL", "GORMES_TTS_PIPER_MODEL_CACHE", "GORMES_TTS_PIPER_CACHE_DIR", "GORMES_TTS_PIPER_VOICE", "GORMES_TTS_PIPER_BIN")
	cache := t.TempDir()
	model := filepath.Join(cache, "en_US-lessac-medium.onnx")
	if err := os.WriteFile(model, []byte(strings.Repeat("m", 2048)), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(model+".json", []byte(`{"audio":{"sample_rate":22050}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	os.Setenv("GORMES_TTS_PIPER_MODEL_CACHE", cache)

	provider := NewGoNativeTTSProvider(GoNativeTTSProviderConfig{})
	if _, ok := provider.RuntimeForTest().(*speechtts.PiperSynthesizer); !ok {
		t.Fatalf("runtime = %T, want cached PiperSynthesizer", provider.RuntimeForTest())
	}
}

func TestGoNativeTTSProviderDoesNotFallThroughWhenExplicitlyUnavailable(t *testing.T) {
	edge := &fakeTTSProvider{available: true}
	result := NewTTSRunner(TTSConfig{Provider: ProviderNameLocalGo}, map[string]TTSProvider{
		ProviderNameLocalGo: NewGoNativeTTSProvider(GoNativeTTSProviderConfig{Disabled: true}),
		"edge":              edge,
	}).Synthesize(context.Background(), TTSRequest{Text: "hello", OutputPath: filepath.Join(t.TempDir(), "local.wav")})

	if result.Success || result.Evidence != TTSEvidenceProviderUnavailable || result.Provider != ProviderNameLocalGo {
		t.Fatalf("result = %+v, want explicit local_go provider_unavailable", result)
	}
	if edge.calls != 0 {
		t.Fatalf("explicit unavailable local_go fell through to edge (%d calls)", edge.calls)
	}
}

func TestGoNativeTTSProviderReturnsRedactedTypedSynthesisError(t *testing.T) {
	output := filepath.Join(t.TempDir(), "local.wav")
	result := NewTTSRunner(TTSConfig{Provider: ProviderNameLocalGo}, map[string]TTSProvider{
		ProviderNameLocalGo: NewGoNativeTTSProvider(GoNativeTTSProviderConfig{Runtime: failingSpeechRuntime{}}),
	}).Synthesize(context.Background(), TTSRequest{Text: "hello", OutputPath: output})

	if result.Success || result.Evidence != TTSEvidenceAPIError || !strings.Contains(result.Error, "tts_invalid_input") {
		t.Fatalf("result = %+v, want typed tts_api_error for unsupported text", result)
	}
}

type failingSpeechRuntime struct{}

func (failingSpeechRuntime) Synthesize(context.Context, speechtts.Request) (speechtts.Result, error) {
	return speechtts.Result{}, &speechtts.Error{Code: speechtts.ErrorCodeInvalidInput, Message: "unsupported text"}
}
