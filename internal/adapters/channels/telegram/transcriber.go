package telegram

import (
	"context"
	"errors"
	"fmt"
	"strings"

	telegramaudio "github.com/TrebuchetDynamics/gormes-agent/internal/adapters/channels/telegram/audio"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

const telegramAudioTranscriptionTimeout = telegramaudio.TranscriptionTimeout

// AudioInput is the sanitized media payload handed to an audio transcriber.
// It intentionally carries Telegram's file_id only as adapter-private metadata;
// transcribers should never print it or derive token-bearing URLs from it.
type AudioInput = telegramaudio.Input

// AudioTranscriber is the small STT seam used by the Telegram adapter. Tests
// provide a fake implementation; production wires CommandAudioTranscriber.
type AudioTranscriber = telegramaudio.Transcriber

// ResolveAudioTranscriber returns the first non-nil candidate, or nil when
// every candidate is nil. Callers list candidates in priority order: the
// local whisper-CLI shim first, then HTTP STT fallbacks. This keeps the
// channel package free of HTTP-provider dependencies — cmd wiring assembles
// the candidate list.
func ResolveAudioTranscriber(candidates ...AudioTranscriber) AudioTranscriber {
	return telegramaudio.Resolve(candidates...)
}

func (b *Bot) transcribeTelegramAudio(ctx context.Context, input AudioInput) (string, error) {
	if b == nil || b.cfg.AudioTranscriber == nil {
		return "", nil
	}
	if len(input.Data) == 0 {
		var err error
		input, _, _, err = b.materializeTelegramAudio(ctx, input)
		if err != nil {
			return "", err
		}
	}
	return b.cfg.AudioTranscriber.Transcribe(ctx, input)
}

func (b *Bot) materializeTelegramAudio(ctx context.Context, input AudioInput) (AudioInput, string, int64, error) {
	if b.client == nil {
		return input, "", 0, errors.New("telegram client unavailable")
	}
	filePath := ""
	if len(input.Data) == 0 {
		fileID := strings.TrimSpace(input.FileID)
		if fileID == "" {
			return input, "", 0, errors.New("telegram file id missing")
		}
		file, err := b.client.GetFile(tgbotapi.FileConfig{FileID: fileID})
		if err != nil {
			return input, "", 0, fmt.Errorf("telegram getFile failed: %w", err)
		}
		filePath = strings.TrimSpace(file.FilePath)
		if filePath == "" {
			return input, "", 0, errors.New("telegram file path missing")
		}
		data, err := b.client.DownloadFile(ctx, filePath)
		if err != nil {
			return input, "", 0, fmt.Errorf("telegram download failed: %w", err)
		}
		if len(data) == 0 {
			return input, "", 0, errors.New("telegram download returned empty audio")
		}
		input.Data = data
	}
	if input.MediaType = cleanTelegramMediaType(input.MediaType); input.MediaType == "" {
		input.MediaType = "audio/ogg"
	}
	if input.FileName = telegramAudioCacheFileName(input, filePath); input.FileName == "" {
		input.FileName = "audio.ogg"
	}
	path, err := b.cacheTelegramBytes("audio", input.FileName, input.Data)
	if err != nil {
		return input, "", int64(len(input.Data)), fmt.Errorf("telegram audio cache failed: %w", err)
	}
	return input, path, int64(len(input.Data)), nil
}

func telegramAudioCacheFileName(input AudioInput, filePath string) string {
	return telegramaudio.CacheFileName(input, filePath)
}

func telegramAudioCacheExtension(mediaType, filePath string) string {
	return telegramaudio.CacheExtension(mediaType, filePath)
}

func sanitizeTelegramAudioError(err error) string {
	return telegramaudio.SanitizeError(err)
}

// telegramAudioErrorDiagnostic returns a short, redacted token suitable for
// internal logging that pinpoints which failure mode tripped without leaking
// Telegram file_ids, direct URLs, or provider response bodies.
func telegramAudioErrorDiagnostic(err error) string {
	return telegramaudio.ErrorDiagnostic(err)
}

// telegramAudioErrorRedactedDetail returns a length-bounded, redacted
// substring of err.Error() suitable for the WARN log.
func telegramAudioErrorRedactedDetail(err error) string {
	return telegramaudio.ErrorRedactedDetail(err)
}

// CommandAudioTranscriber runs a local whisper-compatible CLI in a temporary
// directory. It never logs Telegram file IDs, file paths, direct URLs, or token
// material; callers receive only transcript text or a sanitized error.
type CommandAudioTranscriber = telegramaudio.CommandTranscriber

func NewWhisperTranscriberFromEnv() AudioTranscriber {
	return telegramaudio.NewWhisperTranscriberFromEnv()
}

func audioFileExtension(mediaType, fileName string) string {
	return telegramaudio.FileExtension(mediaType, fileName)
}
