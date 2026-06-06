package engine

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/tools/whisper/testfixture"
)

func TestTranscriberRejectsMissingModel(t *testing.T) {
	ctx := context.Background()
	_, err := NewTranscriber(ctx, filepath.Join(t.TempDir(), "missing-model.bin"), nil)
	if !transcriberErrorCodeIs(err, TranscriberModelUnavailable) {
		t.Fatalf("NewTranscriber error = %v, want %s", err, TranscriberModelUnavailable)
	}
}

func TestDecodeWAVRejectsUnsupportedInput(t *testing.T) {
	path := filepath.Join(t.TempDir(), "not-a-wav.wav")
	if err := os.WriteFile(path, []byte("not a wave"), 0o600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	_, err := DecodePCM16Mono16kWAV(path)
	if !transcriberErrorCodeIs(err, TranscriberWAVUnsupported) {
		t.Fatalf("DecodePCM16Mono16kWAV error = %v, want %s", err, TranscriberWAVUnsupported)
	}
}

func TestTranscriberTranscribesFixtureWAV(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	modelPath := testTinyEnModelPath(t, ctx)
	transcriber, err := NewTranscriber(ctx, modelPath, readWhisperWASM(t))
	if err != nil {
		t.Fatalf("NewTranscriber: %v", err)
	}
	defer func() {
		if err := transcriber.Close(context.Background()); err != nil {
			t.Fatalf("Close: %v", err)
		}
	}()

	transcript, err := transcriber.TranscribeWAV(ctx, testfixture.JFKWAVPath())
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

func transcriberErrorCodeIs(err error, code string) bool {
	var transcribeErr *TranscriberError
	if !errors.As(err, &transcribeErr) {
		return false
	}
	return transcribeErr.Code == code
}
