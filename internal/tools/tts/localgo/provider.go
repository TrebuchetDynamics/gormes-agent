//go:build !slim

package localgo

import (
	"context"
	"errors"
	"strings"

	speechtts "github.com/TrebuchetDynamics/gormes-agent/internal/speech/tts"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tools/tts/contracts"
)

const (
	DefaultProviderName  = "local_go"
	DefaultMaxTextLength = 2000
)

type Config struct {
	Runtime       speechtts.Runtime
	Disabled      bool
	MaxTextLength int
	SampleRate    int
}

// Provider adapts Gormes' Go-owned local TTS runtime to the shared TTS
// provider contract. The default runtime prefers a real local Piper neural
// synthesizer when GORMES_TTS_PIPER_MODEL is configured, then falls back to the
// deterministic fixture runtime for offline tests.
type Provider struct {
	runtime       speechtts.Runtime
	disabled      bool
	maxTextLength int
	sampleRate    int
}

func NewProvider(cfg Config) *Provider {
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
		maxLen = DefaultMaxTextLength
	}
	return &Provider{runtime: runtime, disabled: cfg.Disabled, maxTextLength: maxLen, sampleRate: cfg.SampleRate}
}

func (p *Provider) Available(context.Context) bool {
	return p != nil && !p.disabled && p.runtime != nil
}

func (p *Provider) MaxTextLength() int {
	if p == nil || p.maxTextLength <= 0 {
		return DefaultMaxTextLength
	}
	return p.maxTextLength
}

func (p *Provider) PreferredOutputFormat() string { return "wav" }

func (p *Provider) RuntimeForTest() speechtts.Runtime {
	if p == nil {
		return nil
	}
	return p.runtime
}

func (p *Provider) Synthesize(ctx context.Context, req contracts.ProviderRequest) (contracts.ProviderResult, error) {
	if !p.Available(ctx) {
		return contracts.ProviderResult{}, errors.New("tts_provider_unavailable: Go-owned local TTS runtime is disabled")
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
		return contracts.ProviderResult{}, err
	}
	provider := strings.TrimSpace(req.Provider)
	if provider == "" {
		provider = DefaultProviderName
	}
	return contracts.ProviderResult{FilePath: result.FilePath, Provider: provider}, nil
}
