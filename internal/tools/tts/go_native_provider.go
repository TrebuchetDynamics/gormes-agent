//go:build !slim

package tts

import (
	"context"
	"errors"
	"strings"

	speechtts "github.com/TrebuchetDynamics/gormes-agent/internal/speech/tts"
)

type GoNativeTTSProviderConfig struct {
	Runtime       speechtts.Runtime
	Disabled      bool
	MaxTextLength int
	SampleRate    int
}

// GoNativeTTSProvider adapts Gormes' Go-owned local TTS runtime to the existing
// Hermes-compatible TTSProvider contract. The default runtime prefers a real
// local Piper neural synthesizer when GORMES_TTS_PIPER_MODEL is configured, then
// falls back to the deterministic fixture runtime for offline tests.
type GoNativeTTSProvider struct {
	runtime       speechtts.Runtime
	disabled      bool
	maxTextLength int
	sampleRate    int
}

func NewGoNativeTTSProvider(cfg GoNativeTTSProviderConfig) *GoNativeTTSProvider {
	runtime := cfg.Runtime
	if runtime == nil {
		if piper := speechtts.NewPiperSynthesizerFromEnv(); piper != nil {
			runtime = piper
		}
	}
	if runtime == nil {
		runtime = speechtts.NewFixtureSynthesizer()
	}
	maxLen := cfg.MaxTextLength
	if maxLen <= 0 {
		maxLen = MaxTextLengthLocalGo
	}
	return &GoNativeTTSProvider{runtime: runtime, disabled: cfg.Disabled, maxTextLength: maxLen, sampleRate: cfg.SampleRate}
}

func (p *GoNativeTTSProvider) Available(context.Context) bool {
	return p != nil && !p.disabled && p.runtime != nil
}

func (p *GoNativeTTSProvider) MaxTextLength() int {
	if p == nil || p.maxTextLength <= 0 {
		return MaxTextLengthLocalGo
	}
	return p.maxTextLength
}

func (p *GoNativeTTSProvider) PreferredOutputFormat() string { return "wav" }

func (p *GoNativeTTSProvider) Synthesize(ctx context.Context, req TTSProviderRequest) (TTSProviderResult, error) {
	if !p.Available(ctx) {
		return TTSProviderResult{}, errors.New("tts_provider_unavailable: Go-owned local TTS runtime is disabled")
	}
	result, err := p.runtime.Synthesize(ctx, speechtts.Request{
		Text:          req.Text,
		OutputPath:    req.OutputPath,
		Voice:         strings.TrimSpace(req.Voice),
		Speed:         req.Speed,
		SampleRate:    p.sampleRate,
		MaxTextLength: p.MaxTextLength(),
	})
	if err != nil {
		return TTSProviderResult{}, err
	}
	return TTSProviderResult{FilePath: result.FilePath, Provider: firstNonEmptyTTS(req.Provider, ProviderNameLocalGo)}, nil
}
