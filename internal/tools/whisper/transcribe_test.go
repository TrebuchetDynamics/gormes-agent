package whisper

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestTranscriberRejectsMissingModel(t *testing.T) {
	ctx := context.Background()
	_, err := NewTranscriber(ctx, filepath.Join(t.TempDir(), "missing-model.bin"))
	if !transcriberErrorCodeIs(err, TranscriberModelUnavailable) {
		t.Fatalf("NewTranscriber error = %v, want %s", err, TranscriberModelUnavailable)
	}
}

func TestDecodeWAVRejectsUnsupportedInput(t *testing.T) {
	path := filepath.Join(t.TempDir(), "not-a-wav.wav")
	if err := os.WriteFile(path, []byte("not a wave"), 0o600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	_, err := decodePCM16Mono16kWAV(path)
	if !transcriberErrorCodeIs(err, TranscriberWAVUnsupported) {
		t.Fatalf("decodePCM16Mono16kWAV error = %v, want %s", err, TranscriberWAVUnsupported)
	}
}

func TestTranscriberTranscribesFixtureWAV(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	modelPath := testTinyEnModelPath(t, ctx)
	transcriber, err := NewTranscriber(ctx, modelPath)
	if err != nil {
		t.Fatalf("NewTranscriber: %v", err)
	}
	defer func() {
		if err := transcriber.Close(context.Background()); err != nil {
			t.Fatalf("Close: %v", err)
		}
	}()

	transcript, err := transcriber.TranscribeWAV(ctx, filepath.Join("testdata", "jfk.wav"))
	if err != nil {
		t.Fatalf("TranscribeWAV: %v", err)
	}
	normalized := strings.ToLower(transcript)
	for _, want := range []string{"ask not", "your country", "what you can do"} {
		if !strings.Contains(normalized, want) {
			t.Fatalf("transcript missing %q:\n%s", want, transcript)
		}
	}
}

func testTinyEnModelPath(t *testing.T, ctx context.Context) string {
	t.Helper()
	if path := os.Getenv("GORMES_WASI_WHISPER_MODEL"); path != "" {
		if err := verifyModelFile(path, TinyEnModelArtifact); err != nil {
			t.Fatalf("verify %s: %v", path, err)
		}
		return path
	}

	cacheDir := os.Getenv("GORMES_WASI_WHISPER_MODEL_CACHE")
	if cacheDir == "" {
		userCache, err := os.UserCacheDir()
		if err != nil {
			t.Fatalf("resolve user cache dir: %v", err)
		}
		cacheDir = filepath.Join(userCache, "gormes", "wasi-whisper")
	}
	path, err := EnsureModel(ctx, TinyEnModelArtifact, cacheDir, nil)
	if err != nil {
		t.Fatalf("EnsureModel(%s): %v", cacheDir, err)
	}
	return path
}

func transcriberErrorCodeIs(err error, code string) bool {
	var transcribeErr *TranscriberError
	if !errors.As(err, &transcribeErr) {
		return false
	}
	return transcribeErr.Code == code
}
