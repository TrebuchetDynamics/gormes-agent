package audio

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

type fakeTranscriber struct {
	transcript string
	err        error
}

func (f fakeTranscriber) Transcribe(context.Context, Input) (string, error) {
	return f.transcript, f.err
}

func TestResolve(t *testing.T) {
	local := fakeTranscriber{transcript: "from-local"}
	fallback := fakeTranscriber{transcript: "from-http"}

	resolved := Resolve(nil, local, fallback)
	if resolved == nil {
		t.Fatal("Resolve returned nil")
	}
	got, err := resolved.Transcribe(context.Background(), Input{})
	if err != nil {
		t.Fatalf("Transcribe returned error: %v", err)
	}
	if got != "from-local" {
		t.Fatalf("transcript = %q, want from-local", got)
	}

	if Resolve(nil, nil) != nil {
		t.Fatal("Resolve returned non-nil for all nil candidates")
	}
}

func TestResolveReturnsCandidateAsIs(t *testing.T) {
	want := errors.New("provider down")
	resolved := Resolve(fakeTranscriber{err: want})
	if _, err := resolved.Transcribe(context.Background(), Input{}); !errors.Is(err, want) {
		t.Fatalf("Transcribe error = %v, want %v", err, want)
	}
}

func TestCacheFileName(t *testing.T) {
	got := CacheFileName(Input{Kind: "voice", FileName: "unsafe/name.mp3", MediaType: "audio/ogg"}, "")
	if got != "name.mp3" {
		t.Fatalf("CacheFileName = %q, want name.mp3", got)
	}

	got = CacheFileName(Input{Kind: "voice", MediaType: "audio/mpeg"}, "/tmp/audio.bin")
	if got != "audio.mp3" {
		t.Fatalf("CacheFileName with media type = %q, want audio.mp3", got)
	}

	got = CacheFileName(Input{Kind: "voice"}, "")
	if got != "voice.ogg" {
		t.Fatalf("CacheFileName fallback = %q, want voice.ogg", got)
	}
}

func TestSanitizeAndDiagnostics(t *testing.T) {
	cases := []struct {
		err        error
		sanitized  string
		diagnostic string
	}{
		{errors.New("telegram getFile failed: bot123456:SECRET"), "telegram getFile failed", "telegram_getfile_failed"},
		{errors.New("telegram download returned empty audio"), "telegram download failed", "telegram_empty_download"},
		{errors.New("context deadline exceeded"), "audio transcription timed out", "transcription_timeout"},
		{errors.New("groq stt http 429: nope"), "audio transcription rejected by provider", "stt_http_429"},
	}
	for _, tc := range cases {
		if got := SanitizeError(tc.err); got != tc.sanitized {
			t.Fatalf("SanitizeError(%v) = %q, want %q", tc.err, got, tc.sanitized)
		}
		if got := ErrorDiagnostic(tc.err); got != tc.diagnostic {
			t.Fatalf("ErrorDiagnostic(%v) = %q, want %q", tc.err, got, tc.diagnostic)
		}
	}
}

func TestErrorRedactedDetail(t *testing.T) {
	got := ErrorRedactedDetail(errors.New("GET https://api.telegram.org/file/bot123456:SECRET/path failed"))
	if got != "GET <redacted-telegram-file-url>" {
		t.Fatalf("ErrorRedactedDetail = %q", got)
	}
}

func TestNewWhisperTranscriberFromEnvFindsUserLocalBin(t *testing.T) {
	home := t.TempDir()
	binDir := filepath.Join(home, ".local", "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatal(err)
	}
	whisperPath := filepath.Join(binDir, "whisper")
	if err := os.WriteFile(whisperPath, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	t.Setenv("HOME", home)
	t.Setenv("PATH", filepath.Join(t.TempDir(), "missing-bin"))
	t.Setenv("GORMES_WHISPER_COMMAND", "")

	got := NewWhisperTranscriberFromEnv()
	transcriber, ok := got.(CommandTranscriber)
	if !ok {
		t.Fatalf("NewWhisperTranscriberFromEnv() = %T, want CommandTranscriber", got)
	}
	if transcriber.Command != whisperPath {
		t.Fatalf("Command = %q, want %q", transcriber.Command, whisperPath)
	}
}
