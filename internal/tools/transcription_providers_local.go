//go:build !gormes_lite && !slim

package tools

import (
	"context"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/wasi/whisper"
)

type LocalSTTProvider struct {
	cacheDir string
	client   *http.Client
}

func NewLocalSTTProvider(cacheDir string) *LocalSTTProvider {
	if cacheDir == "" {
		cacheDir = filepath.Join(os.Getenv("HOME"), ".gormes", "cache", "whisper")
	}
	return &LocalSTTProvider{
		cacheDir: cacheDir,
		client:   http.DefaultClient,
	}
}

func (p *LocalSTTProvider) Available(ctx context.Context) bool {
	a := whisper.TinyEnModelArtifact
	return strings.TrimSpace(a.Filename) != "" && strings.TrimSpace(a.URL) != "" && a.SizeBytes > 0
}

func (p *LocalSTTProvider) Transcribe(ctx context.Context, req TranscriptionProviderRequest) (TranscriptionProviderResult, error) {
	audioPath := strings.TrimSpace(req.AudioPath)
	if audioPath == "" {
		return TranscriptionProviderResult{}, errors.New("audio path is required")
	}
	if _, err := os.Stat(audioPath); err != nil {
		return TranscriptionProviderResult{}, err
	}

	modelPath, err := whisper.EnsureModel(ctx, whisper.TinyEnModelArtifact, p.cacheDir, p.client)
	if err != nil {
		return TranscriptionProviderResult{}, err
	}

	t, err := whisper.NewTranscriber(ctx, modelPath)
	if err != nil {
		return TranscriptionProviderResult{}, err
	}
	defer t.Close(ctx)

	transcript, err := t.TranscribeWAV(ctx, audioPath)
	if err != nil {
		return TranscriptionProviderResult{}, err
	}

	return TranscriptionProviderResult{
		Transcript: transcript,
		Provider:   "local",
		Model:      "tiny.en",
	}, nil
}
