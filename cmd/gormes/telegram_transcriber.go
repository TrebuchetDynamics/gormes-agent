//go:build !slim

package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/channels/telegram"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tools"
)

// httpSTTAudioTranscriber adapts a tools.TranscriptionProvider into the
// telegram.AudioTranscriber seam. The underlying provider is file-path
// based, so the adapter materializes the in-memory audio bytes into a
// short-lived tempfile and removes it after Transcribe returns. No file
// IDs, paths, or token material are logged.
type httpSTTAudioTranscriber struct {
	provider tools.TranscriptionProvider
}

func (a httpSTTAudioTranscriber) Transcribe(ctx context.Context, audio telegram.AudioInput) (string, error) {
	if a.provider == nil {
		return "", errors.New("http STT provider unavailable")
	}
	if !a.provider.Available(ctx) {
		return "", errors.New("http STT provider not configured")
	}
	if len(audio.Data) == 0 {
		return "", errors.New("http STT received empty audio")
	}

	dir, err := os.MkdirTemp("", "gormes-telegram-audio-http-*")
	if err != nil {
		return "", fmt.Errorf("http STT tempdir: %w", err)
	}
	defer os.RemoveAll(dir)

	ext := strings.TrimSpace(filepath.Ext(audio.FileName))
	if ext == "" || len(ext) > 10 {
		ext = ".ogg"
	}
	path := filepath.Join(dir, "input"+ext)
	if err := os.WriteFile(path, audio.Data, 0o600); err != nil {
		return "", fmt.Errorf("http STT temp write: %w", err)
	}

	res, err := a.provider.Transcribe(ctx, tools.TranscriptionProviderRequest{AudioPath: path})
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(res.Transcript), nil
}

// newHTTPAudioTranscriberFromEnv returns an HTTP-backed AudioTranscriber when
// any HTTP STT provider key is present in the environment, else nil.
// Provider preference matches Hermes' free-first stance: Groq's free tier
// wins over paid OpenAI Whisper when both are configured. Future provider
// expansion (Mistral, xAI) plugs in here without changing callers.
func newHTTPAudioTranscriberFromEnv() telegram.AudioTranscriber {
	if strings.TrimSpace(os.Getenv("GROQ_API_KEY")) != "" {
		return httpSTTAudioTranscriber{
			provider: tools.NewTranscriptionGroqProvider(tools.TranscriptionProviderConfig{}),
		}
	}
	openAIKey := firstNonEmpty(
		strings.TrimSpace(os.Getenv("GORMES_STT_OPENAI_KEY")),
		strings.TrimSpace(os.Getenv("OPENAI_API_KEY")),
		strings.TrimSpace(os.Getenv("VOICE_TOOLS_OPENAI_KEY")),
	)
	if openAIKey != "" {
		return httpSTTAudioTranscriber{
			provider: tools.NewTranscriptionOpenAIProvider(tools.TranscriptionProviderConfig{}),
		}
	}
	return nil
}

// resolveTelegramAudioTranscriber picks an AudioTranscriber for the Telegram
// channel: local whisper-CLI shim first (free, offline), HTTP STT provider
// second (when configured), nil third (channel falls back to attachment
// markers, no transcription). Local-first matches Hermes' default stance.
func resolveTelegramAudioTranscriber() telegram.AudioTranscriber {
	return telegram.ResolveAudioTranscriber(
		telegram.NewWhisperTranscriberFromEnv(),
		newHTTPAudioTranscriberFromEnv(),
	)
}
