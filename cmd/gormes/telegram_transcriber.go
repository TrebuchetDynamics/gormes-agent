//go:build !slim

package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/channels/telegram"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tools"
	wasiwhisper "github.com/TrebuchetDynamics/gormes-agent/internal/tools/whisper"
	whisperaudio "github.com/TrebuchetDynamics/gormes-agent/internal/tools/whisper/audio"
)

const (
	wasiWhisperModelEnv             = "GORMES_WASI_WHISPER_MODEL"
	wasiWhisperTranscriptionTimeout = 2 * time.Minute
	wasiWhisperMaxChunkDuration     = 30 * time.Second
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

type wasiWhisperTranscriber interface {
	TranscribeWAV(ctx context.Context, path string) (string, error)
	Close(ctx context.Context) error
}

type wasiWhisperAudioTranscriber struct {
	modelPath string

	mu             sync.Mutex
	transcriber    wasiWhisperTranscriber
	newTranscriber func(context.Context, string) (wasiWhisperTranscriber, error)
	convertToWAV   whisperaudio.Converter
}

func newWASIWhisperTranscriberFromEnv() telegram.AudioTranscriber {
	modelPath := strings.TrimSpace(os.Getenv(wasiWhisperModelEnv))
	if modelPath == "" {
		return nil
	}
	info, err := os.Stat(modelPath)
	if err != nil || !info.Mode().IsRegular() {
		return nil
	}
	return &wasiWhisperAudioTranscriber{
		modelPath:      modelPath,
		newTranscriber: newWASIWhisperTranscriber,
	}
}

func newWASIWhisperTranscriber(ctx context.Context, modelPath string) (wasiWhisperTranscriber, error) {
	return wasiwhisper.NewTranscriber(ctx, modelPath)
}

func (a *wasiWhisperAudioTranscriber) Transcribe(ctx context.Context, audio telegram.AudioInput) (string, error) {
	if a == nil {
		return "", errors.New("wasi whisper transcriber unavailable")
	}
	if len(audio.Data) == 0 {
		return "", errors.New("wasi whisper received empty audio")
	}
	ctx, cancel := context.WithTimeout(ctx, wasiWhisperTranscriptionTimeout)
	defer cancel()

	pcm, err := whisperaudio.Preprocess(ctx, audio.Data, audio.MediaType, whisperaudio.PreprocessOptions{
		FileName:  audio.FileName,
		Converter: a.convertToWAV,
	})
	if err != nil {
		return "", fmt.Errorf("wasi whisper audio preprocess: %w", err)
	}
	chunks := whisperaudio.ChunkPCM(pcm.Samples, pcm.SampleRate, wasiWhisperMaxChunkDuration)
	if len(chunks) == 0 {
		return "", errors.New("wasi whisper audio preprocess produced no chunks")
	}

	transcriber, err := a.getTranscriber(ctx)
	if err != nil {
		return "", err
	}

	dir, err := os.MkdirTemp("", "gormes-telegram-audio-wasi-*")
	if err != nil {
		return "", fmt.Errorf("wasi whisper tempdir: %w", err)
	}
	defer os.RemoveAll(dir)

	transcripts := make([]string, 0, len(chunks))
	for i, samples := range chunks {
		raw, err := whisperaudio.EncodePCM16MonoWAV(whisperaudio.PCM{
			SampleRate: pcm.SampleRate,
			Samples:    samples,
		})
		if err != nil {
			return "", fmt.Errorf("wasi whisper audio encode chunk: %w", err)
		}
		chunkPath := filepath.Join(dir, fmt.Sprintf("chunk-%04d.wav", i+1))
		if err := os.WriteFile(chunkPath, raw, 0o600); err != nil {
			return "", fmt.Errorf("wasi whisper temp write: %w", err)
		}
		transcript, err := transcriber.TranscribeWAV(ctx, chunkPath)
		if err != nil {
			return "", err
		}
		if trimmed := strings.TrimSpace(transcript); trimmed != "" {
			transcripts = append(transcripts, trimmed)
		}
	}
	return strings.Join(transcripts, "\n"), nil
}

func (a *wasiWhisperAudioTranscriber) getTranscriber(ctx context.Context) (wasiWhisperTranscriber, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.transcriber != nil {
		return a.transcriber, nil
	}
	factory := a.newTranscriber
	if factory == nil {
		factory = newWASIWhisperTranscriber
	}
	transcriber, err := factory(ctx, a.modelPath)
	if err != nil {
		return nil, err
	}
	a.transcriber = transcriber
	return transcriber, nil
}

var (
	newLocalTelegramAudioTranscriber       = telegram.NewWhisperTranscriberFromEnv
	newWASIWhisperTelegramAudioTranscriber = newWASIWhisperTranscriberFromEnv
	newHTTPSTTTelegramAudioTranscriber     = newHTTPAudioTranscriberFromEnv
)

// resolveTelegramAudioTranscriber picks an AudioTranscriber for the Telegram
// channel: local whisper-CLI shim first (free, offline), in-binary WASI
// Whisper second (when a model file is configured), HTTP STT provider third,
// nil last (channel falls back to attachment markers, no transcription).
// Local-first matches Hermes' default stance.
func resolveTelegramAudioTranscriber() telegram.AudioTranscriber {
	return telegram.ResolveAudioTranscriber(
		newLocalTelegramAudioTranscriber(),
		newWASIWhisperTelegramAudioTranscriber(),
		newHTTPSTTTelegramAudioTranscriber(),
	)
}
