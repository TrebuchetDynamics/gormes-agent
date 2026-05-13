//go:build !gormes_lite && !slim

package tools

import (
	"context"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/wasi/whisper"
)

func TestLocalSTTProvider_Available(t *testing.T) {
	p := NewLocalSTTProvider(t.TempDir())
	if !p.Available(context.Background()) {
		t.Fatal("LocalSTTProvider should be available with default artifact config")
	}
}

func TestLocalSTTProvider_Transcribe_RejectsEmptyPath(t *testing.T) {
	p := NewLocalSTTProvider(t.TempDir())
	_, err := p.Transcribe(context.Background(), TranscriptionProviderRequest{AudioPath: ""})
	if err == nil {
		t.Fatal("expected error for empty audio path")
	}
}

func TestLocalSTTProvider_Transcribe_RejectsMissingFile(t *testing.T) {
	p := NewLocalSTTProvider(t.TempDir())
	_, err := p.Transcribe(context.Background(), TranscriptionProviderRequest{AudioPath: "/nonexistent/audio.wav"})
	if err == nil {
		t.Fatal("expected error for missing audio file")
	}
}

func TestLocalSTTProvider_Transcribe_RejectsNonWAV(t *testing.T) {
	dir := t.TempDir()
	nonWAV := filepath.Join(dir, "test.txt")
	if err := os.WriteFile(nonWAV, []byte("not audio"), 0o644); err != nil {
		t.Fatal(err)
	}
	p := NewLocalSTTProvider(dir)
	_, err := p.Transcribe(context.Background(), TranscriptionProviderRequest{AudioPath: nonWAV})
	if err == nil {
		t.Fatal("expected error for non-WAV file")
	}
}

func TestLocalSTTProvider_Transcribe_JFKFixture(t *testing.T) {
	jfkPath := filepath.Join("..", "wasi", "whisper", "testdata", "jfk.wav")
	if _, err := os.Stat(jfkPath); err != nil {
		t.Skip("jfk.wav test fixture not available:", err)
	}
	cacheDir := localSTTFixtureModelCacheDir(t)
	modelPath := filepath.Join(cacheDir, whisper.TinyEnModelArtifact.Filename)
	if _, err := os.Stat(modelPath); err != nil {
		if os.IsNotExist(err) {
			t.Skipf("WASI Whisper tiny.en model is not cached at %s; run internal/wasi/whisper integration tests or set GORMES_WASI_WHISPER_MODEL_CACHE", modelPath)
		}
		t.Fatalf("stat cached model: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	p := NewLocalSTTProvider(cacheDir)
	p.client = &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return nil, errors.New("network disabled in local STT provider fixture test")
	})}
	result, err := p.Transcribe(ctx, TranscriptionProviderRequest{AudioPath: jfkPath})
	if err != nil {
		if modelDownloadUnavailable(err) {
			t.Skipf("WASI Whisper tiny.en model unavailable from cache %s: %v", cacheDir, err)
		}
		t.Fatalf("Transcribe jfk.wav: %v", err)
	}
	if result.Transcript == "" {
		t.Fatal("empty transcript from jfk.wav")
	}
	if result.Provider != "local" {
		t.Fatalf("provider = %q, want local", result.Provider)
	}
	if result.Model != "tiny.en" {
		t.Fatalf("model = %q, want tiny.en", result.Model)
	}

	normalized := strings.ToLower(result.Transcript)
	for _, want := range []string{"ask not", "your country", "what you can do"} {
		if !strings.Contains(normalized, want) {
			t.Fatalf("transcript missing %q:\n%s", want, result.Transcript)
		}
	}
}

func localSTTFixtureModelCacheDir(t *testing.T) string {
	t.Helper()
	if cacheDir := strings.TrimSpace(os.Getenv("GORMES_WASI_WHISPER_MODEL_CACHE")); cacheDir != "" {
		return cacheDir
	}
	userCache, err := os.UserCacheDir()
	if err != nil {
		t.Skipf("resolve user cache dir: %v", err)
	}
	return filepath.Join(userCache, "gormes", "wasi-whisper")
}

func modelDownloadUnavailable(err error) bool {
	var cacheErr *whisper.ModelCacheError
	return errors.As(err, &cacheErr) && cacheErr.Code == whisper.ModelCacheDownloadFailed
}
