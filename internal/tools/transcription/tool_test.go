//go:build !slim

package transcription

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestTranscriptionValidateAudioInput(t *testing.T) {
	ctx := context.Background()
	provider := &fakeTranscriptionProvider{available: true}
	runner := NewTranscriptionRunner(TranscriptionConfig{
		Provider: "local",
		MaxBytes: 8,
	}, map[string]TranscriptionProvider{
		"local": provider,
	})

	missing := filepath.Join(t.TempDir(), "missing.ogg")
	result := runner.Transcribe(ctx, TranscriptionRequest{AudioPath: missing})
	if result.Success || result.Evidence != TranscriptionEvidenceAudioNotFound {
		t.Fatalf("missing result = %+v, want audio_not_found failure", result)
	}

	dirResult := runner.Transcribe(ctx, TranscriptionRequest{AudioPath: t.TempDir()})
	if dirResult.Success || dirResult.Evidence != TranscriptionEvidenceAudioNotFile {
		t.Fatalf("directory result = %+v, want audio_not_file failure", dirResult)
	}

	unsupported := filepath.Join(t.TempDir(), "clip.txt")
	if err := os.WriteFile(unsupported, []byte("not audio"), 0o600); err != nil {
		t.Fatal(err)
	}
	unsupportedResult := runner.Transcribe(ctx, TranscriptionRequest{AudioPath: unsupported})
	if unsupportedResult.Success || unsupportedResult.Evidence != TranscriptionEvidenceUnsupportedAudioFormat {
		t.Fatalf("unsupported result = %+v, want unsupported_audio_format failure", unsupportedResult)
	}

	oversized := filepath.Join(t.TempDir(), "clip.ogg")
	if err := os.WriteFile(oversized, []byte("0123456789"), 0o600); err != nil {
		t.Fatal(err)
	}
	oversizedResult := runner.Transcribe(ctx, TranscriptionRequest{AudioPath: oversized})
	if oversizedResult.Success || oversizedResult.Evidence != TranscriptionEvidenceAudioTooLarge {
		t.Fatalf("oversized result = %+v, want audio_too_large failure", oversizedResult)
	}

	if provider.calls != 0 {
		t.Fatalf("provider calls = %d, want 0 for invalid input", provider.calls)
	}
}

func TestTranscriptionProviderSelection(t *testing.T) {
	ctx := context.Background()
	audio := writeTestAudioFile(t, "clip.ogg", []byte("fake audio"))
	providers := map[string]TranscriptionProvider{
		"local":         &fakeTranscriptionProvider{available: false},
		"local_command": &fakeTranscriptionProvider{available: false},
		"groq":          &fakeTranscriptionProvider{available: true, result: TranscriptionProviderResult{Transcript: "groq transcript"}},
		"openai":        &fakeTranscriptionProvider{available: true, result: TranscriptionProviderResult{Transcript: "openai transcript"}},
	}

	auto := NewTranscriptionRunner(TranscriptionConfig{}, providers).Transcribe(ctx, TranscriptionRequest{AudioPath: audio})
	if !auto.Success || auto.Provider != "groq" || auto.Transcript != "groq transcript" {
		t.Fatalf("auto result = %+v, want first available groq provider", auto)
	}

	explicit := NewTranscriptionRunner(TranscriptionConfig{Provider: "openai"}, providers).Transcribe(ctx, TranscriptionRequest{AudioPath: audio})
	if !explicit.Success || explicit.Provider != "openai" || explicit.Transcript != "openai transcript" {
		t.Fatalf("explicit result = %+v, want openai without auto fallback", explicit)
	}

	noFallback := NewTranscriptionRunner(TranscriptionConfig{Provider: "local_command"}, providers).Transcribe(ctx, TranscriptionRequest{AudioPath: audio})
	if noFallback.Success || noFallback.Evidence != TranscriptionEvidenceProviderUnavailable || noFallback.Provider != "local_command" {
		t.Fatalf("noFallback result = %+v, want explicit local_command unavailable", noFallback)
	}
}

func TestTranscriptionModelNormalization(t *testing.T) {
	ctx := context.Background()
	audio := writeTestAudioFile(t, "clip.ogg", []byte("fake audio"))

	cases := []struct {
		name     string
		provider string
		model    string
		want     string
	}{
		{name: "local cloud model falls back", provider: "local", model: "whisper-1", want: "base"},
		{name: "local custom model passes through", provider: "local", model: "large-v3", want: "large-v3"},
		{name: "groq openai model normalizes", provider: "groq", model: "gpt-4o-transcribe", want: "whisper-large-v3-turbo"},
		{name: "openai groq model normalizes", provider: "openai", model: "whisper-large-v3", want: "whisper-1"},
		{name: "mistral default", provider: "mistral", model: "", want: "voxtral-mini-latest"},
		{name: "xai default", provider: "xai", model: "", want: "grok-stt"},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			provider := &fakeTranscriptionProvider{available: true, result: TranscriptionProviderResult{Transcript: "ok"}}
			result := NewTranscriptionRunner(TranscriptionConfig{Provider: tt.provider}, map[string]TranscriptionProvider{
				tt.provider: provider,
			}).Transcribe(ctx, TranscriptionRequest{AudioPath: audio, Model: tt.model})
			if !result.Success || result.Model != tt.want {
				t.Fatalf("result = %+v, want model %q", result, tt.want)
			}
		})
	}
}

func TestTranscriptionResultEnvelope(t *testing.T) {
	ctx := context.Background()
	audio := writeTestAudioFile(t, "clip.ogg", []byte("fake audio"))
	success := NewTranscriptionRunner(TranscriptionConfig{Provider: "openai"}, map[string]TranscriptionProvider{
		"openai": &fakeTranscriptionProvider{available: true, result: TranscriptionProviderResult{
			Transcript: "Juan asked for repo status",
			Provider:   "openai",
			Model:      "whisper-1",
			Language:   "en",
		}},
	}).Transcribe(ctx, TranscriptionRequest{AudioPath: audio})
	if !success.Success || success.Evidence != TranscriptionEvidenceOK || success.Transcript == "" || success.Provider != "openai" || success.Model != "whisper-1" || success.Language != "en" {
		t.Fatalf("success = %+v, want populated result envelope", success)
	}

	failed := NewTranscriptionRunner(TranscriptionConfig{Provider: "openai"}, map[string]TranscriptionProvider{
		"openai": &fakeTranscriptionProvider{available: true, err: errors.New("Bearer sk-secret-token failed")},
	}).Transcribe(ctx, TranscriptionRequest{AudioPath: audio})
	if failed.Success || failed.Evidence != TranscriptionEvidenceAPIError {
		t.Fatalf("failed = %+v, want stt_api_error", failed)
	}
	if strings.Contains(failed.Error, "sk-secret-token") || !strings.Contains(failed.Error, "[redacted]") {
		t.Fatalf("failure error was not redacted: %+v", failed)
	}
}

func TestTranscriptionToolDescriptor(t *testing.T) {
	tool := NewTranscriptionTool(NewTranscriptionRunner(TranscriptionConfig{}, nil))
	if tool.Name() != "transcribe_audio" {
		t.Fatalf("tool name = %q", tool.Name())
	}
	var schema struct {
		Properties map[string]json.RawMessage `json:"properties"`
		Required   []string                   `json:"required"`
	}
	if err := json.Unmarshal(tool.Schema(), &schema); err != nil {
		t.Fatalf("schema invalid JSON: %v", err)
	}
	for _, field := range []string{"audio_path", "provider", "model", "language", "format"} {
		if _, ok := schema.Properties[field]; !ok {
			t.Fatalf("schema missing %q in %s", field, tool.Schema())
		}
	}
	if len(schema.Required) != 1 || schema.Required[0] != "audio_path" {
		t.Fatalf("required = %#v, want audio_path only", schema.Required)
	}
}

type fakeTranscriptionProvider struct {
	available bool
	calls     int
	result    TranscriptionProviderResult
	err       error
}

func (f *fakeTranscriptionProvider) Available(context.Context) bool {
	return f.available
}

func (f *fakeTranscriptionProvider) Transcribe(_ context.Context, req TranscriptionProviderRequest) (TranscriptionProviderResult, error) {
	f.calls++
	if f.result.Provider == "" {
		f.result.Provider = req.Provider
	}
	if f.result.Model == "" {
		f.result.Model = req.Model
	}
	return f.result, f.err
}
