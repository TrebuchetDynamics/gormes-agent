package whisper

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

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

func testTinyEnModelPath(t testing.TB, ctx context.Context) string {
	t.Helper()
	if path := strings.TrimSpace(os.Getenv("GORMES_WASI_WHISPER_MODEL")); path != "" {
		if err := verifyModelFile(path, TinyEnModelArtifact); err != nil {
			t.Fatalf("verify %s: %v", path, err)
		}
		return path
	}

	cacheDir := strings.TrimSpace(os.Getenv("GORMES_WASI_WHISPER_MODEL_CACHE"))
	if cacheDir == "" {
		userCache, err := os.UserCacheDir()
		if err != nil {
			t.Fatalf("resolve user cache dir: %v", err)
		}
		cacheDir = filepath.Join(userCache, "gormes", "wasi-whisper")
	}
	path, err := EnsureModel(ctx, TinyEnModelArtifact, cacheDir, nil)
	if err != nil {
		t.Skipf("WASI Whisper tiny.en model unavailable in %s: %v", cacheDir, err)
	}
	return path
}
