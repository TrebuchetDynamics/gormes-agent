//go:build !slim

package tts

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestTextToSpeechToolDescriptor(t *testing.T) {
	tool := NewTextToSpeechTool(NewTTSRunner(TTSConfig{}, nil))
	if tool.Name() != "text_to_speech" {
		t.Fatalf("tool name = %q", tool.Name())
	}
	var schema struct {
		Properties map[string]json.RawMessage `json:"properties"`
		Required   []string                   `json:"required"`
	}
	if err := json.Unmarshal(tool.Schema(), &schema); err != nil {
		t.Fatalf("schema invalid JSON: %v", err)
	}
	for _, field := range []string{"text", "output_path", "language"} {
		if _, ok := schema.Properties[field]; !ok {
			t.Fatalf("schema missing %q in %s", field, tool.Schema())
		}
	}
	if len(schema.Required) != 1 || schema.Required[0] != "text" {
		t.Fatalf("required = %#v, want text only", schema.Required)
	}
}

func TestTextToSpeechResultEnvelope(t *testing.T) {
	ctx := context.Background()
	output := filepath.Join(t.TempDir(), "voice.ogg")
	provider := &fakeTTSProvider{
		available: true,
		result: TTSProviderResult{
			Provider:        "openai",
			VoiceCompatible: true,
		},
	}
	result := NewTTSRunner(TTSConfig{
		Provider: "openai",
		Now:      func() time.Time { return time.Date(2026, 4, 29, 12, 0, 0, 0, time.UTC) },
	}, map[string]TTSProvider{
		"openai": provider,
	}).Synthesize(ctx, TTSRequest{
		Text:       "Hello Juan",
		OutputPath: output,
		Platform:   "telegram",
	})

	if !result.Success || result.Evidence != TTSEvidenceOK {
		t.Fatalf("result = %+v, want success tts_synthesized", result)
	}
	if result.FilePath != output || result.Provider != "openai" || !result.VoiceCompatible {
		t.Fatalf("result envelope = %+v, want file/provider/voice-compatible populated", result)
	}
	if result.MediaTag != "[[audio_as_voice]]\nMEDIA:"+output {
		t.Fatalf("MediaTag = %q, want Hermes voice MEDIA tag", result.MediaTag)
	}
	if provider.calls != 1 || provider.last.Text != "Hello Juan" || provider.last.OutputPath != output {
		t.Fatalf("provider calls=%d last=%+v", provider.calls, provider.last)
	}
}

func TestTextToSpeechValidationAndRedaction(t *testing.T) {
	ctx := context.Background()
	runner := NewTTSRunner(TTSConfig{Provider: "edge"}, map[string]TTSProvider{
		"edge": &fakeTTSProvider{available: true, err: errors.New("Bearer sk-secret-token failed")},
	})

	empty := runner.Synthesize(ctx, TTSRequest{Text: "   "})
	if empty.Success || empty.Evidence != TTSEvidenceInvalidArguments {
		t.Fatalf("empty result = %+v, want invalid args", empty)
	}

	badOutput := runner.Synthesize(ctx, TTSRequest{Text: "hello", OutputPath: filepath.Join(t.TempDir(), "voice.txt")})
	if badOutput.Success || badOutput.Evidence != TTSEvidenceUnsupportedAudioFormat {
		t.Fatalf("bad output result = %+v, want unsupported audio format", badOutput)
	}

	output := filepath.Join(t.TempDir(), "voice.mp3")
	failed := runner.Synthesize(ctx, TTSRequest{Text: "hello", OutputPath: output})
	if failed.Success || failed.Evidence != TTSEvidenceAPIError {
		t.Fatalf("failed result = %+v, want tts_api_error", failed)
	}
	if strings.Contains(failed.Error, "sk-secret-token") || !strings.Contains(failed.Error, "[redacted]") {
		t.Fatalf("failure error was not redacted: %+v", failed)
	}
}

func TestTextToSpeechToolExecuteHonorsProviderOverride(t *testing.T) {
	output := filepath.Join(t.TempDir(), "voice.mp3")
	edge := &fakeTTSProvider{available: true}
	openai := &fakeTTSProvider{available: true}
	tool := NewTextToSpeechTool(NewTTSRunner(TTSConfig{Provider: "edge"}, map[string]TTSProvider{
		"edge":   edge,
		"openai": openai,
	}))
	raw, err := tool.Execute(context.Background(), json.RawMessage(`{"text":"Hello","output_path":`+strconvQuote(output)+`,"provider":"openai","platform":"telegram","voice":"nova","speed":"fast"}`))
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	var result TTSResult
	if err := json.Unmarshal(raw, &result); err != nil {
		t.Fatalf("result JSON: %v", err)
	}
	if !result.Success || result.Provider != "openai" {
		t.Fatalf("result = %+v, want openai provider success", result)
	}
	if edge.calls != 0 || openai.calls != 1 {
		t.Fatalf("provider calls edge/openai = %d/%d, want 0/1", edge.calls, openai.calls)
	}
	if openai.last.Platform != "telegram" || openai.last.Provider != "openai" || openai.last.Voice != "nova" || openai.last.Speed != 1.25 {
		t.Fatalf("openai request = %+v, want provider/platform/voice/speed override", openai.last)
	}
}

func TestTextToSpeechToolExecutePassesLanguageHint(t *testing.T) {
	output := filepath.Join(t.TempDir(), "voice.mp3")
	edge := &fakeTTSProvider{available: true}
	tool := NewTextToSpeechTool(NewTTSRunner(TTSConfig{Provider: "edge"}, map[string]TTSProvider{
		"edge": edge,
	}))
	raw, err := tool.Execute(context.Background(), json.RawMessage(`{"text":"Hola, puedo ayudarte.","output_path":`+strconvQuote(output)+`,"language":"auto"}`))
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	var result TTSResult
	if err := json.Unmarshal(raw, &result); err != nil {
		t.Fatalf("result JSON: %v", err)
	}
	if !result.Success || edge.calls != 1 {
		t.Fatalf("result=%+v calls=%d, want successful edge call", result, edge.calls)
	}
	if edge.last.Language != "auto" {
		t.Fatalf("provider language = %q, want auto", edge.last.Language)
	}
}

func TestTextToSpeechToolExecute(t *testing.T) {
	output := filepath.Join(t.TempDir(), "voice.mp3")
	tool := NewTextToSpeechTool(NewTTSRunner(TTSConfig{Provider: "edge"}, map[string]TTSProvider{
		"edge": &fakeTTSProvider{available: true},
	}))
	raw, err := tool.Execute(context.Background(), json.RawMessage(`{"text":"Hello","output_path":`+strconvQuote(output)+`}`))
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	var result TTSResult
	if err := json.Unmarshal(raw, &result); err != nil {
		t.Fatalf("result JSON: %v", err)
	}
	if !result.Success || result.FilePath != output || result.MediaTag != "MEDIA:"+output {
		t.Fatalf("result = %+v, want success envelope with MEDIA tag", result)
	}
}

func TestEdgeTTSVoiceForLanguageAutoDetectsSpanishOutput(t *testing.T) {
	got := edgeTTSVoiceForLanguage("auto", "Hola, claro. Puedo ayudarte con la tarea.")
	if got != "es-ES-ElviraNeural" {
		t.Fatalf("edgeTTSVoiceForLanguage(auto spanish) = %q, want es-ES-ElviraNeural", got)
	}
}

func TestEdgeTTSVoiceForLanguageAutoWeightsEnglishAndSpanish(t *testing.T) {
	english := edgeTTSVoiceForLanguage("auto", "Hello, I can help you with the task.")
	if english != "en-US-AriaNeural" {
		t.Fatalf("edgeTTSVoiceForLanguage(auto english) = %q, want en-US-AriaNeural", english)
	}
	spanishDominant := edgeTTSVoiceForLanguage("auto", "Hello Juan, gracias. Puedo ayudarte con la tarea.")
	if spanishDominant != "es-ES-ElviraNeural" {
		t.Fatalf("edgeTTSVoiceForLanguage(auto mixed Spanish-dominant) = %q, want es-ES-ElviraNeural", spanishDominant)
	}
}

func TestTelegramAccountPlatformTTSHelpers(t *testing.T) {
	if !shouldPreferOpusForTTS("openai", "telegram:ops") {
		t.Fatal("expected OpenAI TTS on account-scoped Telegram to prefer Opus output")
	}
	if shouldPreferOpusForTTS("openai", "discord") {
		t.Fatal("did not expect non-Telegram platforms to prefer Opus output")
	}
	if !shouldTreatTTSAsVoiceCompatible("openai", "/tmp/voice.ogg", "telegram:ops") {
		t.Fatal("expected account-scoped Telegram OGG output to be voice compatible")
	}
}

type fakeTTSProvider struct {
	available bool
	calls     int
	last      TTSProviderRequest
	result    TTSProviderResult
	err       error
}

func (f *fakeTTSProvider) Available(context.Context) bool {
	return f.available
}

func (f *fakeTTSProvider) Synthesize(_ context.Context, req TTSProviderRequest) (TTSProviderResult, error) {
	f.calls++
	f.last = req
	if f.err != nil {
		return TTSProviderResult{}, f.err
	}
	if err := os.MkdirAll(filepath.Dir(req.OutputPath), 0o700); err != nil {
		return TTSProviderResult{}, err
	}
	if err := os.WriteFile(req.OutputPath, []byte("audio bytes"), 0o600); err != nil {
		return TTSProviderResult{}, err
	}
	result := f.result
	if result.FilePath == "" {
		result.FilePath = req.OutputPath
	}
	if result.Provider == "" {
		result.Provider = req.Provider
	}
	return result, nil
}

func strconvQuote(s string) string {
	raw, _ := json.Marshal(s)
	return string(raw)
}
