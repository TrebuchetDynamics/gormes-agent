//go:build !slim

package main

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/channels/telegram"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tools"
)

// fakeTranscriptionProvider implements tools.TranscriptionProvider for tests
// without touching live HTTP endpoints. It records the AudioPath the
// adapter wrote so we can assert the bytes survived the trip through the
// tempfile bridge.
type fakeTranscriptionProvider struct {
	available     bool
	availableErr  error
	transcript    string
	err           error
	receivedReq   tools.TranscriptionProviderRequest
	receivedBytes []byte
}

func (f *fakeTranscriptionProvider) Available(ctx context.Context) bool {
	_ = ctx
	return f.available
}

func (f *fakeTranscriptionProvider) Transcribe(ctx context.Context, req tools.TranscriptionProviderRequest) (tools.TranscriptionProviderResult, error) {
	_ = ctx
	f.receivedReq = req
	if req.AudioPath != "" {
		if data, readErr := os.ReadFile(req.AudioPath); readErr == nil {
			f.receivedBytes = data
		}
	}
	if f.err != nil {
		return tools.TranscriptionProviderResult{}, f.err
	}
	return tools.TranscriptionProviderResult{Transcript: f.transcript}, nil
}

func TestHTTPSTTAudioTranscriber_BridgesBytesToProviderAndReturnsTranscript(t *testing.T) {
	provider := &fakeTranscriptionProvider{available: true, transcript: "hello world"}
	adapter := httpSTTAudioTranscriber{provider: provider}

	wantBytes := []byte("OggS\x00fake-opus-payload")
	got, err := adapter.Transcribe(context.Background(), telegram.AudioInput{
		Kind:      "voice",
		MediaType: "audio/ogg",
		FileName:  "voice-message.oga",
		Data:      wantBytes,
	})
	if err != nil {
		t.Fatalf("Transcribe returned error: %v", err)
	}
	if got != "hello world" {
		t.Fatalf("expected transcript %q, got %q", "hello world", got)
	}
	if string(provider.receivedBytes) != string(wantBytes) {
		t.Fatalf("provider received wrong bytes: want %q got %q", wantBytes, provider.receivedBytes)
	}
	if !strings.HasSuffix(provider.receivedReq.AudioPath, ".oga") {
		t.Fatalf("expected tempfile to keep .oga extension from FileName, got path %q", provider.receivedReq.AudioPath)
	}
	// Tempfile must be cleaned up after Transcribe returns.
	if _, statErr := os.Stat(provider.receivedReq.AudioPath); !os.IsNotExist(statErr) {
		t.Fatalf("expected tempfile to be removed, stat err: %v", statErr)
	}
}

func TestHTTPSTTAudioTranscriber_ReturnsErrorWhenProviderUnavailable(t *testing.T) {
	provider := &fakeTranscriptionProvider{available: false}
	adapter := httpSTTAudioTranscriber{provider: provider}

	_, err := adapter.Transcribe(context.Background(), telegram.AudioInput{Data: []byte("anything")})
	if err == nil {
		t.Fatal("expected error when provider is unavailable, got nil")
	}
}

func TestHTTPSTTAudioTranscriber_ReturnsErrorWhenProviderTranscribeFails(t *testing.T) {
	wantErr := errors.New("provider boom")
	provider := &fakeTranscriptionProvider{available: true, err: wantErr}
	adapter := httpSTTAudioTranscriber{provider: provider}

	_, err := adapter.Transcribe(context.Background(), telegram.AudioInput{
		FileName: "x.ogg",
		Data:     []byte("payload"),
	})
	if err == nil {
		t.Fatal("expected propagated error, got nil")
	}
	if !errors.Is(err, wantErr) && !strings.Contains(err.Error(), "provider boom") {
		t.Fatalf("expected upstream error to surface, got %v", err)
	}
}

func TestHTTPSTTAudioTranscriber_FallsBackToOggExtensionWhenFileNameMissing(t *testing.T) {
	provider := &fakeTranscriptionProvider{available: true, transcript: "ok"}
	adapter := httpSTTAudioTranscriber{provider: provider}

	_, err := adapter.Transcribe(context.Background(), telegram.AudioInput{Data: []byte("x")})
	if err != nil {
		t.Fatalf("Transcribe returned error: %v", err)
	}
	if !strings.HasSuffix(provider.receivedReq.AudioPath, ".ogg") {
		t.Fatalf("expected default .ogg extension when FileName is empty, got %q", provider.receivedReq.AudioPath)
	}
}

func TestNewHTTPAudioTranscriberFromEnv_ReturnsNilWhenNoOpenAIKey(t *testing.T) {
	t.Setenv("GORMES_STT_OPENAI_KEY", "")
	t.Setenv("OPENAI_API_KEY", "")
	t.Setenv("VOICE_TOOLS_OPENAI_KEY", "")

	got := newHTTPAudioTranscriberFromEnv()
	if got != nil {
		t.Fatalf("expected nil when no OpenAI STT key is set, got %T", got)
	}
}

func TestNewHTTPAudioTranscriberFromEnv_ReturnsOpenAIAdapterWhenKeySet(t *testing.T) {
	t.Setenv("GORMES_STT_OPENAI_KEY", "")
	t.Setenv("VOICE_TOOLS_OPENAI_KEY", "")
	t.Setenv("OPENAI_API_KEY", "sk-test-not-real")

	got := newHTTPAudioTranscriberFromEnv()
	if got == nil {
		t.Fatal("expected non-nil HTTP transcriber when OPENAI_API_KEY is set, got nil")
	}
	if _, ok := got.(httpSTTAudioTranscriber); !ok {
		t.Fatalf("expected httpSTTAudioTranscriber adapter, got %T", got)
	}
}

func TestResolveTelegramAudioTranscriber_FallsBackToHTTPWhenNoLocalCLI(t *testing.T) {
	// Force NewWhisperTranscriberFromEnv to return nil by clearing the env
	// override and ensuring no whisper binary is on PATH for this test.
	t.Setenv("GORMES_WHISPER_COMMAND", "")
	t.Setenv("PATH", "/nonexistent")
	// Configure HTTP fallback.
	t.Setenv("GORMES_STT_OPENAI_KEY", "")
	t.Setenv("VOICE_TOOLS_OPENAI_KEY", "")
	t.Setenv("OPENAI_API_KEY", "sk-test-not-real")

	got := resolveTelegramAudioTranscriber()
	if got == nil {
		t.Fatal("expected resolver to return HTTP fallback when no local whisper is available, got nil")
	}
	if _, ok := got.(httpSTTAudioTranscriber); !ok {
		t.Fatalf("expected httpSTTAudioTranscriber, got %T", got)
	}
}

func TestResolveTelegramAudioTranscriber_ReturnsNilWhenNeitherLocalNorHTTPConfigured(t *testing.T) {
	t.Setenv("GORMES_WHISPER_COMMAND", "")
	t.Setenv("PATH", "/nonexistent")
	t.Setenv("GORMES_STT_OPENAI_KEY", "")
	t.Setenv("OPENAI_API_KEY", "")
	t.Setenv("VOICE_TOOLS_OPENAI_KEY", "")

	got := resolveTelegramAudioTranscriber()
	if got != nil {
		t.Fatalf("expected nil when neither local nor HTTP is configured, got %T", got)
	}
}
