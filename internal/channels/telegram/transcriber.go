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
	case strings.Contains(msg, "groq stt http "):
		return "audio transcription rejected by provider"
	case strings.Contains(msg, "groq stt") || strings.Contains(msg, "openai stt") || strings.Contains(msg, "http stt"):
		return "audio transcription provider failed"
	case strings.Contains(msg, "telegram file id missing") || strings.Contains(msg, "telegram file path missing"):
		return "telegram file metadata missing"
	case strings.Contains(msg, "empty audio"):
		return "telegram download returned empty audio"
	default:
		return "audio unavailable"
	}
}

// telegramAudioErrorDiagnostic returns a short, redacted token suitable for
// internal logging that pinpoints which failure mode tripped without leaking
// Telegram file_ids, direct URLs, or provider response bodies. It mirrors
// sanitizeTelegramAudioError's classification but stays in snake_case so log
// scrapers can group failures cleanly. Provider HTTP status codes are
// preserved when present (Groq STT HTTP 401 → "stt_http_401").
func telegramAudioErrorDiagnostic(err error) string {
	if err == nil {
		return ""
	}
	msg := strings.ToLower(err.Error())
	switch {
	case strings.Contains(msg, "getfile"):
		return "telegram_getfile_failed"
	case strings.Contains(msg, "download") && strings.Contains(msg, "empty"):
		return "telegram_empty_download"
	case strings.Contains(msg, "download"):
		return "telegram_download_failed"
	case strings.Contains(msg, "timeout"), strings.Contains(msg, "deadline"):
		return "transcription_timeout"
	case strings.Contains(msg, "groq stt http "):
		// Format: "Groq STT HTTP 401: ..." → extract the 3-digit code.
		if idx := strings.Index(msg, "groq stt http "); idx >= 0 {
			tail := msg[idx+len("groq stt http "):]
			tail = strings.TrimSpace(tail)
			if len(tail) >= 3 {
				code := tail[:3]
				if _, convErr := strconvAtoi(code); convErr == nil {
					return "stt_http_" + code
				}
			}
		}
		return "stt_http_unknown"
	case strings.Contains(msg, "groq stt"):
		return "stt_groq_local_failure"
	case strings.Contains(msg, "openai stt"):
		return "stt_openai_local_failure"
	case strings.Contains(msg, "http stt"):
		return "stt_adapter_failure"
	case strings.Contains(msg, "whisper"), strings.Contains(msg, "transcrib"):
		return "stt_local_command_failure"
	case strings.Contains(msg, "telegram file id missing"):
		return "telegram_file_id_missing"
	case strings.Contains(msg, "telegram file path missing"):
		return "telegram_file_path_missing"
	case strings.Contains(msg, "empty audio"):
		return "transcriber_empty_audio"
	default:
		return "unclassified"
	}
}

// strconvAtoi is a thin local indirection so telegramAudioErrorDiagnostic
// stays in this file without forcing a strconv import on the rest of the
// package surface (which never needed strconv before).
func strconvAtoi(s string) (int, error) {
	n := 0
	for _, r := range s {
		if r < '0' || r > '9' {
			return 0, fmt.Errorf("not a digit: %q", s)
		}
		n = n*10 + int(r-'0')
	}
	return n, nil
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
