package audio

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

	telegrammedia "github.com/TrebuchetDynamics/gormes-agent/internal/adapters/channels/telegram/media"
)

const TranscriptionTimeout = 2 * time.Minute

// Input is the sanitized media payload handed to an audio transcriber.
// It intentionally carries Telegram's file_id only as adapter-private metadata;
// transcribers should never print it or derive token-bearing URLs from it.
type Input struct {
	Kind      string
	FileID    string
	MediaType string
	FileName  string
	Duration  time.Duration
	Data      []byte
}

// Transcriber is the small STT seam used by the Telegram adapter.
type Transcriber interface {
	Transcribe(ctx context.Context, audio Input) (string, error)
}

// Resolve returns the first non-nil candidate, or nil when every candidate is nil.
func Resolve(candidates ...Transcriber) Transcriber {
	for _, c := range candidates {
		if c != nil {
			return c
		}
	}
	return nil
}

func CacheFileName(input Input, filePath string) string {
	if fileName := telegrammedia.SafeFileName(input.FileName); fileName != "" {
		if ext := CacheExtension(input.MediaType, fileName); ext != "" {
			return strings.TrimSuffix(fileName, filepath.Ext(fileName)) + ext
		}
		return fileName
	}
	ext := CacheExtension(input.MediaType, filePath)
	if base := telegrammedia.SafeFileName(filepath.Base(strings.TrimSpace(filePath))); base != "" && base != "." {
		return strings.TrimSuffix(base, filepath.Ext(base)) + ext
	}
	kind := strings.ToLower(strings.TrimSpace(input.Kind))
	if kind == "" {
		kind = "audio"
	}
	return kind + ext
}

func CacheExtension(mediaType, filePath string) string {
	switch ext := strings.ToLower(filepath.Ext(strings.TrimSpace(filePath))); ext {
	case ".oga", ".opus":
		return ".ogg"
	case ".aac", ".flac", ".m4a", ".mp3", ".mp4", ".mpeg", ".mpga", ".ogg", ".wav", ".webm":
		return ext
	}
	switch telegrammedia.CleanMediaType(mediaType) {
	case "audio/ogg", "audio/opus":
		return ".ogg"
	case "audio/mpeg", "audio/mp3":
		return ".mp3"
	case "audio/mp4", "audio/aac":
		return ".m4a"
	case "audio/flac":
		return ".flac"
	case "audio/wav", "audio/wave", "audio/x-wav":
		return ".wav"
	case "audio/webm":
		return ".webm"
	default:
		return ".ogg"
	}
}

func SanitizeError(err error) string {
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

// ErrorDiagnostic returns a short, redacted token suitable for internal logging.
func ErrorDiagnostic(err error) string {
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
		if idx := strings.Index(msg, "groq stt http "); idx >= 0 {
			tail := strings.TrimSpace(msg[idx+len("groq stt http "):])
			if len(tail) >= 3 {
				code := tail[:3]
				if _, convErr := atoi(code); convErr == nil {
					return "stt_http_" + code
				}
			}
		}
		return "stt_http_unknown"
	case strings.Contains(msg, "groq stt http:"):
		return "stt_groq_network_failure"
	case strings.Contains(msg, "groq stt open audio"):
		return "stt_groq_file_open_failure"
	case strings.Contains(msg, "groq stt request"):
		return "stt_groq_request_build_failure"
	case strings.Contains(msg, "groq stt parse response"):
		return "stt_groq_parse_failure"
	case strings.Contains(msg, "groq stt copy audio"):
		return "stt_groq_copy_failure"
	case strings.Contains(msg, "groq stt close writer"):
		return "stt_groq_writer_close_failure"
	case strings.Contains(msg, "groq stt"):
		return "stt_groq_form_failure"
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

// ErrorRedactedDetail returns a length-bounded, redacted substring of err.Error().
func ErrorRedactedDetail(err error) string {
	if err == nil {
		return ""
	}
	raw := err.Error()
	if i := strings.Index(raw, "https://api.telegram.org/file/bot"); i >= 0 {
		raw = raw[:i] + "<redacted-telegram-file-url>"
	}
	if i := strings.Index(strings.ToLower(raw), "bot"); i >= 0 {
		if i+3 < len(raw) && raw[i+3] >= '0' && raw[i+3] <= '9' {
			end := i + 3
			for end < len(raw) && raw[end] != ' ' && raw[end] != '"' && raw[end] != '\'' && raw[end] != '/' {
				end++
			}
			raw = raw[:i] + "<redacted-bot-token>" + raw[end:]
		}
	}
	if len(raw) > 256 {
		raw = raw[:256] + "...(truncated)"
	}
	return raw
}

func atoi(s string) (int, error) {
	n := 0
	for _, r := range s {
		if r < '0' || r > '9' {
			return 0, fmt.Errorf("not a digit: %q", s)
		}
		n = n*10 + int(r-'0')
	}
	return n, nil
}

// CommandTranscriber runs a local whisper-compatible CLI in a temporary directory.
type CommandTranscriber struct {
	Command string
	Model   string
	Timeout time.Duration
}

func NewWhisperTranscriberFromEnv() Transcriber {
	cmd := strings.TrimSpace(os.Getenv("GORMES_WHISPER_COMMAND"))
	if cmd == "" {
		var err error
		cmd, err = exec.LookPath("whisper")
		if err != nil {
			cmd = defaultUserWhisperCommand()
			if cmd == "" {
				return nil
			}
		}
	}
	model := strings.TrimSpace(os.Getenv("GORMES_WHISPER_MODEL"))
	if model == "" {
		model = "tiny"
	}
	return CommandTranscriber{Command: cmd, Model: model, Timeout: TranscriptionTimeout}
}

func defaultUserWhisperCommand() string {
	home := strings.TrimSpace(os.Getenv("HOME"))
	if home == "" {
		return ""
	}
	candidate := filepath.Join(home, ".local", "bin", "whisper")
	info, err := os.Stat(candidate)
	if err != nil || info.IsDir() || info.Mode()&0o111 == 0 {
		return ""
	}
	return candidate
}

func (t CommandTranscriber) Transcribe(ctx context.Context, input Input) (string, error) {
	cmdPath := strings.TrimSpace(t.Command)
	if cmdPath == "" {
		return "", errors.New("whisper command unavailable")
	}
	timeout := t.Timeout
	if timeout <= 0 {
		timeout = TranscriptionTimeout
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	dir, err := os.MkdirTemp("", "gormes-telegram-audio-*")
	if err != nil {
		return "", fmt.Errorf("audio tempdir: %w", err)
	}
	defer os.RemoveAll(dir)

	ext := FileExtension(input.MediaType, input.FileName)
	name := "input" + ext
	inPath := filepath.Join(dir, name)
	if err := os.WriteFile(inPath, input.Data, 0o600); err != nil {
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

func FileExtension(mediaType, fileName string) string {
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
