//go:build !slim

package contracts

import "context"

// Evidence is stable operator-facing evidence for TTS outcomes.
type Evidence string

const (
	EvidenceOK                     Evidence = "tts_synthesized"
	EvidenceDisabled               Evidence = "tts_disabled"
	EvidenceInvalidArguments       Evidence = "tts_invalid_arguments"
	EvidenceUnsupportedAudioFormat Evidence = "unsupported_audio_format"
	EvidenceProviderUnavailable    Evidence = "tts_provider_unavailable"
	EvidenceAPIError               Evidence = "tts_api_error"
	EvidenceOutputMissing          Evidence = "tts_output_missing"
)

// Request is the public helper input.
type Request struct {
	Text       string
	OutputPath string
	Provider   string
	Platform   string
	Voice      string
	Speed      float64
}

// Result is the redacted helper/tool result envelope.
type Result struct {
	Success         bool     `json:"success"`
	FilePath        string   `json:"file_path,omitempty"`
	MediaTag        string   `json:"media_tag,omitempty"`
	Provider        string   `json:"provider,omitempty"`
	VoiceCompatible bool     `json:"voice_compatible,omitempty"`
	Truncated       bool     `json:"truncated,omitempty"`
	Evidence        Evidence `json:"evidence"`
	Error           string   `json:"error,omitempty"`
}

// ProviderRequest is the normalized provider call input.
type ProviderRequest struct {
	Text       string
	OutputPath string
	Provider   string
	Platform   string
	Voice      string
	Speed      float64
}

// ProviderResult is the provider-specific response before the runner normalizes it.
type ProviderResult struct {
	FilePath        string
	Provider        string
	VoiceCompatible bool
}

// Provider is implemented by real or fake synthesis backends.
type Provider interface {
	Available(context.Context) bool
	Synthesize(context.Context, ProviderRequest) (ProviderResult, error)
}
