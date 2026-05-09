package telegram

import (
	"context"
	"errors"
	"fmt"
	"mime"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

const telegramAudioTranscriptionTimeout = 2 * time.Minute

// AudioInput is the sanitized media payload handed to an audio transcriber.
// It intentionally carries Telegram's file_id only as adapter-private metadata;
// transcribers should never print it or derive token-bearing URLs from it.
type AudioInput struct {
	Kind      string
	FileID    string
	MediaType string
	FileName  string
	Duration  time.Duration
	Data      []byte
}

// AudioTranscriber is the small STT seam used by the Telegram adapter. Tests
// provide a fake implementation; production wires CommandAudioTranscriber.
type AudioTranscriber interface {
	Transcribe(ctx context.Context, audio AudioInput) (string, error)
}

// ResolveAudioTranscriber returns the first non-nil candidate, or nil when
// every candidate is nil. Callers list candidates in priority order: the
// local whisper-CLI shim first, then HTTP STT fallbacks. This keeps the
// channel package free of HTTP-provider dependencies — cmd wiring assembles
// the candidate list.
func ResolveAudioTranscriber(candidates ...AudioTranscriber) AudioTranscriber {
	for _, c := range candidates {
		if c != nil {
			return c
		}
	}
	return nil
}

func (b *Bot) transcribeTelegramAudio(ctx context.Context, input AudioInput) (string, error) {
	if b == nil || b.cfg.AudioTranscriber == nil {
		return "", nil
	}
	if b.client == nil {
		return "", errors.New("telegram client unavailable")
	}
	fileID := strings.TrimSpace(input.FileID)
	if fileID == "" {
		return "", errors.New("telegram file id missing")
	}
	file, err := b.client.GetFile(tgbotapi.FileConfig{FileID: fileID})
	if err != nil {
		return "", fmt.Errorf("telegram getFile failed: %w", err)
	}
	filePath := strings.TrimSpace(file.FilePath)
	if filePath == "" {
		return "", errors.New("telegram file path missing")
	}
	data, err := b.client.DownloadFile(ctx, filePath)
	if err != nil {
		return "", fmt.Errorf("telegram download failed: %w", err)
	}
	if len(data) == 0 {
		return "", errors.New("telegram download returned empty audio")
	}
	input.Data = data
	return b.cfg.AudioTranscriber.Transcribe(ctx, input)
}

func sanitizeTelegramAudioError(err error) string {
	if err == nil {
		return ""
	}
	msg := strings.ToLower(err.Error())
	switch {
	case strings.Contains(msg, "getfile"):
		return "telegram getFile failed"
	case strings.Contains(msg, "download"):
		return "telegram download failed"
	case strings.Contains(msg, "timeout"), strings.Contains(msg, "deadline"):
		return "audio transcription timed out"
	case strings.Contains(msg, "whisper"), strings.Contains(msg, "transcrib"):
		return "audio transcription failed"
	default:
		return "audio unavailable"
	}
}

// CommandAudioTranscriber runs a local whisper-compatible CLI in a temporary
// directory. It never logs Telegram file IDs, file paths, direct URLs, or token
// material; callers receive only transcript text or a sanitized error.
type CommandAudioTranscriber struct {
	Command string
	Model   string
	Timeout time.Duration
}

func NewWhisperTranscriberFromEnv() AudioTranscriber {
	cmd := strings.TrimSpace(os.Getenv("GORMES_WHISPER_COMMAND"))
	if cmd == "" {
		var err error
		cmd, err = exec.LookPath("whisper")
		if err != nil {
			return nil
		}
	}
	model := strings.TrimSpace(os.Getenv("GORMES_WHISPER_MODEL"))
	if model == "" {
		model = "tiny"
	}
	return CommandAudioTranscriber{Command: cmd, Model: model, Timeout: telegramAudioTranscriptionTimeout}
}

func (t CommandAudioTranscriber) Transcribe(ctx context.Context, audio AudioInput) (string, error) {
	cmdPath := strings.TrimSpace(t.Command)
	if cmdPath == "" {
		return "", errors.New("whisper command unavailable")
	}
	timeout := t.Timeout
	if timeout <= 0 {
		timeout = telegramAudioTranscriptionTimeout
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	dir, err := os.MkdirTemp("", "gormes-telegram-audio-*")
	if err != nil {
		return "", fmt.Errorf("audio tempdir: %w", err)
	}
	defer os.RemoveAll(dir)

	ext := audioFileExtension(audio.MediaType, audio.FileName)
	name := "input" + ext
	inPath := filepath.Join(dir, name)
	if err := os.WriteFile(inPath, audio.Data, 0o600); err != nil {
		return "", fmt.Errorf("audio temp write: %w", err)
	}

	args := []string{inPath, "--output_dir", dir, "--output_format", "txt", "--verbose", "False"}
	if model := strings.TrimSpace(t.Model); model != "" {
		args = append(args, "--model", model)
	}
	cmd := exec.CommandContext(ctx, cmdPath, args...)
	if out, err := cmd.CombinedOutput(); err != nil {
		_ = out
		if ctx.Err() != nil {
			return "", ctx.Err()
		}
		return "", errors.New("whisper transcription failed")
	}

	base := strings.TrimSuffix(name, filepath.Ext(name)) + ".txt"
	transcriptPath := filepath.Join(dir, base)
	data, err := os.ReadFile(transcriptPath)
	if err != nil {
		return "", fmt.Errorf("whisper transcript missing: %w", err)
	}
	return strings.TrimSpace(string(data)), nil
}

func audioFileExtension(mediaType, fileName string) string {
	if ext := strings.TrimSpace(filepath.Ext(fileName)); ext != "" && len(ext) <= 10 {
		return ext
	}
	if mt := strings.TrimSpace(mediaType); mt != "" {
		if exts, err := mime.ExtensionsByType(mt); err == nil && len(exts) > 0 {
			return exts[0]
		}
	}
	return ".ogg"
}
