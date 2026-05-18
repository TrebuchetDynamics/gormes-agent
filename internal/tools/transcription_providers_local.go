//go:build !gormes_lite && !slim

package tools

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/wasi/whisper"
	whisperaudio "github.com/TrebuchetDynamics/gormes-agent/internal/wasi/whisper/audio"
)

const localSTTMaxChunkDuration = 30 * time.Second

type localSTTWhisperTranscriber interface {
	TranscribeWAV(context.Context, string) (string, error)
	Close(context.Context) error
}

type LocalSTTProvider struct {
	cacheDir       string
	client         *http.Client
	ensureModel    func(context.Context) (string, error)
	newTranscriber func(context.Context, string) (localSTTWhisperTranscriber, error)
	convertToWAV   whisperaudio.Converter
}

func NewLocalSTTProvider(cacheDir string) *LocalSTTProvider {
	if cacheDir == "" {
		cacheDir = filepath.Join(os.Getenv("HOME"), ".gormes", "cache", "whisper")
	}
	p := &LocalSTTProvider{
		cacheDir: cacheDir,
		client:   http.DefaultClient,
	}
	p.ensureModel = func(ctx context.Context) (string, error) {
		return whisper.EnsureModel(ctx, whisper.TinyEnModelArtifact, p.cacheDir, p.client)
	}
	p.newTranscriber = func(ctx context.Context, modelPath string) (localSTTWhisperTranscriber, error) {
		return whisper.NewTranscriber(ctx, modelPath)
	}
	return p
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

	raw, err := os.ReadFile(audioPath)
	if err != nil {
		return TranscriptionProviderResult{}, err
	}
	pcm, err := whisperaudio.Preprocess(ctx, raw, "", whisperaudio.PreprocessOptions{
		FileName:  filepath.Base(audioPath),
		Converter: p.convertToWAV,
	})
	if err != nil {
		return TranscriptionProviderResult{}, err
	}
	modelPath, err := p.localModel(ctx)
	if err != nil {
		return TranscriptionProviderResult{}, err
	}
	t, err := p.localTranscriber(ctx, modelPath)
	if err != nil {
		return TranscriptionProviderResult{}, err
	}
	defer t.Close(ctx)

	chunks := whisperaudio.ChunkPCM(pcm.Samples, pcm.SampleRate, localSTTMaxChunkDuration)
	if len(chunks) == 0 {
		return TranscriptionProviderResult{}, errors.New("audio preprocess produced no chunks")
	}
	dir, err := os.MkdirTemp("", "gormes-local-stt-*")
	if err != nil {
		return TranscriptionProviderResult{}, err
	}
	defer os.RemoveAll(dir)

	transcripts := make([]string, 0, len(chunks))
	for i, samples := range chunks {
		wav, err := whisperaudio.EncodePCM16MonoWAV(whisperaudio.PCM{Samples: samples, SampleRate: pcm.SampleRate})
		if err != nil {
			return TranscriptionProviderResult{}, err
		}
		base := strings.TrimSuffix(filepath.Base(audioPath), filepath.Ext(audioPath))
		path := filepath.Join(dir, fmt.Sprintf("%s-%04d.wav", base, i+1))
		if err := os.WriteFile(path, wav, 0o600); err != nil {
			return TranscriptionProviderResult{}, err
		}
		transcript, err := t.TranscribeWAV(ctx, path)
		if err != nil {
			return TranscriptionProviderResult{}, err
		}
		if transcript = strings.TrimSpace(transcript); transcript != "" {
			transcripts = append(transcripts, transcript)
		}
	}
	transcript := strings.Join(transcripts, "\n")
	if strings.TrimSpace(transcript) == "" {
		return TranscriptionProviderResult{}, errors.New("empty whisper transcript")
	}

	return TranscriptionProviderResult{
		Transcript: transcript,
		Provider:   "local",
		Model:      "tiny.en",
	}, nil
}

func (p *LocalSTTProvider) localModel(ctx context.Context) (string, error) {
	if p.ensureModel != nil {
		return p.ensureModel(ctx)
	}
	return whisper.EnsureModel(ctx, whisper.TinyEnModelArtifact, p.cacheDir, p.client)
}

func (p *LocalSTTProvider) localTranscriber(ctx context.Context, modelPath string) (localSTTWhisperTranscriber, error) {
	if p.newTranscriber != nil {
		return p.newTranscriber(ctx, modelPath)
	}
	return whisper.NewTranscriber(ctx, modelPath)
}
