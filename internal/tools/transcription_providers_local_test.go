//go:build !gormes_lite && !slim

package tools

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestLocalSTTProvider_Available(t *testing.T) {
	p := NewLocalSTTProvider(t.TempDir())
	if !p.Available(context.Background()) {
		t.Fatal("LocalSTTProvider should be available with default artifact config")
	}
}

func TestLocalSTTProvider_Available_BadCacheDir(t *testing.T) {
	// Unreadable cache dir should not affect Available() — it validates
	// the artifact definition, not the directory.
	p := NewLocalSTTProvider("/dev/null/nope")
	if !p.Available(context.Background()) {
		t.Fatal("Available should return true even with bad cache dir")
	}
}

func TestLocalSTTProvider_Transcribe_MissingFile(t *testing.T) {
	p := NewLocalSTTProvider(t.TempDir())
	_, err := p.Transcribe(context.Background(), TranscriptionProviderRequest{
		AudioPath: "/nonexistent/audio.wav",
	})
	if err == nil {
		t.Fatal("expected error for missing audio file")
	}
}

func TestLocalSTTProvider_Transcribe_EmptyPath(t *testing.T) {
	p := NewLocalSTTProvider(t.TempDir())
	_, err := p.Transcribe(context.Background(), TranscriptionProviderRequest{
		AudioPath: "",
	})
	if err == nil {
		t.Fatal("expected error for empty audio path")
	}
}

func TestLocalSTTProvider_Transcribe_NotWAV(t *testing.T) {
	// Create a non-WAV file to test that the transcriber rejects it gracefully.
	dir := t.TempDir()
	nonWAV := filepath.Join(dir, "test.txt")
	if err := os.WriteFile(nonWAV, []byte("not audio"), 0o644); err != nil {
		t.Fatal(err)
	}

	p := NewLocalSTTProvider(dir)
	result, err := p.Transcribe(context.Background(), TranscriptionProviderRequest{
		AudioPath: nonWAV,
	})
	if err != nil {
		// Accept error — the WASM model may download or fail gracefully.
		t.Logf("Transcribe returned error (expected for non-WAV): %v", err)
		return
	}
	// If it succeeded somehow, result should have meaningful data.
	if result.Transcript == "" && result.Provider == "" {
		t.Fatal("expected either transcript or error for non-audio file")
	}
}
