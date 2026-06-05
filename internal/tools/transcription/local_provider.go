//go:build !gormes_lite && !slim

package transcription

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/tools/whisper"
	whisperaudio "github.com/TrebuchetDynamics/gormes-agent/internal/tools/whisper/audio"
)

const (
	localSTTMaxChunkDuration = 30 * time.Second
	localSTTChunkOverlap     = 1 * time.Second
)

type localSTTWhisperTranscriber interface {
	TranscribeWAV(context.Context, string) (string, error)
	Close(context.Context) error
}

type LocalSTTProvider struct {
	cacheDir       string
	client         *http.Client
	ensureModel    func(context.Context, whisper.ModelArtifact) (string, error)
	newTranscriber func(context.Context, string) (localSTTWhisperTranscriber, error)
	convertToWAV   whisperaudio.Converter

	mu                sync.Mutex
	cachedModelPath   string
	cachedTranscriber localSTTWhisperTranscriber
}

func NewLocalSTTProvider(cacheDir string) *LocalSTTProvider {
	if cacheDir == "" {
		cacheDir = filepath.Join(os.Getenv("HOME"), ".gormes", "cache", "whisper")
	}
	p := &LocalSTTProvider{
		cacheDir: cacheDir,
		client:   http.DefaultClient,
	}
	p.ensureModel = func(ctx context.Context, artifact whisper.ModelArtifact) (string, error) {
		return whisper.EnsureModel(ctx, artifact, p.cacheDir, p.client)
	}
	p.newTranscriber = func(ctx context.Context, modelPath string) (localSTTWhisperTranscriber, error) {
		return whisper.NewTranscriber(ctx, modelPath)
	}
	return p
}

func (p *LocalSTTProvider) Available(ctx context.Context) bool {
	_, a := whisper.ResolveModelArtifact("", "")
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
	pcm = whisperaudio.NormalizeSpeechPCM(pcm)
	modelName, modelArtifact := whisper.ResolveModelArtifact(req.Model, req.Language)
	modelPath, err := p.localModel(ctx, modelArtifact)
	if err != nil {
		return TranscriptionProviderResult{}, err
	}
	t, err := p.localTranscriber(ctx, modelPath)
	if err != nil {
		return TranscriptionProviderResult{}, err
	}

	chunks := whisperaudio.ChunkPCMWithOverlap(pcm.Samples, pcm.SampleRate, localSTTMaxChunkDuration, localSTTChunkOverlap)
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
	transcript := stitchChunkTranscripts(transcripts)
	if strings.TrimSpace(transcript) == "" {
		return TranscriptionProviderResult{}, errors.New("empty whisper transcript")
	}

	return TranscriptionProviderResult{
		Transcript: transcript,
		Provider:   "local",
		Model:      modelName,
	}, nil
}

func stitchChunkTranscripts(transcripts []string) string {
	stitched := ""
	for _, transcript := range transcripts {
		transcript = strings.TrimSpace(transcript)
		if transcript == "" {
			continue
		}
		if stitched == "" {
			stitched = transcript
			continue
		}
		stitched = appendTranscriptWithOverlap(stitched, transcript)
	}
	return stitched
}

func appendTranscriptWithOverlap(previous, next string) string {
	prevFields := strings.Fields(previous)
	nextFields := strings.Fields(next)
	if len(prevFields) == 0 {
		return strings.TrimSpace(next)
	}
	if len(nextFields) == 0 {
		return strings.TrimSpace(previous)
	}
	maxOverlap := len(prevFields)
	if len(nextFields) < maxOverlap {
		maxOverlap = len(nextFields)
	}
	for n := maxOverlap; n > 0; n-- {
		if equalFoldWords(prevFields[len(prevFields)-n:], nextFields[:n]) {
			if n == len(nextFields) {
				return strings.TrimSpace(previous)
			}
			return strings.TrimSpace(previous + " " + strings.Join(nextFields[n:], " "))
		}
	}
	return strings.TrimSpace(previous + "\n" + next)
}

func equalFoldWords(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if strings.Trim(strings.ToLower(a[i]), `.,!?;:"'()[]{}-`) != strings.Trim(strings.ToLower(b[i]), `.,!?;:"'()[]{}-`) {
			return false
		}
	}
	return true
}

func (p *LocalSTTProvider) localModel(ctx context.Context, artifact whisper.ModelArtifact) (string, error) {
	if p.ensureModel != nil {
		return p.ensureModel(ctx, artifact)
	}
	return whisper.EnsureModel(ctx, artifact, p.cacheDir, p.client)
}

func (p *LocalSTTProvider) localTranscriber(ctx context.Context, modelPath string) (localSTTWhisperTranscriber, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.cachedTranscriber != nil && p.cachedModelPath == modelPath {
		return p.cachedTranscriber, nil
	}
	if p.cachedTranscriber != nil {
		_ = p.cachedTranscriber.Close(ctx)
		p.cachedTranscriber = nil
		p.cachedModelPath = ""
	}
	var transcriber localSTTWhisperTranscriber
	var err error
	if p.newTranscriber != nil {
		transcriber, err = p.newTranscriber(ctx, modelPath)
	} else {
		transcriber, err = whisper.NewTranscriber(ctx, modelPath)
	}
	if err != nil {
		return nil, err
	}
	p.cachedModelPath = modelPath
	p.cachedTranscriber = transcriber
	return transcriber, nil
}

func (p *LocalSTTProvider) Close(ctx context.Context) error {
	if p == nil {
		return nil
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.cachedTranscriber == nil {
		return nil
	}
	transcriber := p.cachedTranscriber
	p.cachedTranscriber = nil
	p.cachedModelPath = ""
	return transcriber.Close(ctx)
}
