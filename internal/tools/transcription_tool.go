//go:build !slim

package tools

import "github.com/TrebuchetDynamics/gormes-agent/internal/tools/transcription"

const (
	TranscriptionEvidenceOK                     TranscriptionEvidence = transcription.TranscriptionEvidenceOK
	TranscriptionEvidenceDisabled               TranscriptionEvidence = transcription.TranscriptionEvidenceDisabled
	TranscriptionEvidenceAudioNotFound          TranscriptionEvidence = transcription.TranscriptionEvidenceAudioNotFound
	TranscriptionEvidenceAudioNotFile           TranscriptionEvidence = transcription.TranscriptionEvidenceAudioNotFile
	TranscriptionEvidenceUnsupportedAudioFormat TranscriptionEvidence = transcription.TranscriptionEvidenceUnsupportedAudioFormat
	TranscriptionEvidenceAudioTooLarge          TranscriptionEvidence = transcription.TranscriptionEvidenceAudioTooLarge
	TranscriptionEvidenceProviderUnavailable    TranscriptionEvidence = transcription.TranscriptionEvidenceProviderUnavailable
	TranscriptionEvidenceAPIError               TranscriptionEvidence = transcription.TranscriptionEvidenceAPIError
	TranscriptionEvidenceInvalidArguments       TranscriptionEvidence = transcription.TranscriptionEvidenceInvalidArguments

	ProviderNameDevice  = transcription.ProviderNameDevice
	ProviderNameLocal   = transcription.ProviderNameLocal
	ProviderNameGroq    = transcription.ProviderNameGroq
	ProviderNameMistral = transcription.ProviderNameMistral
	ProviderNameXAI     = transcription.ProviderNameXAI

	DefaultGroqBaseURL     = transcription.DefaultGroqBaseURL
	DefaultMistralBaseURL  = transcription.DefaultMistralBaseURL
	DefaultXAIBaseURL      = transcription.DefaultXAIBaseURL
	DefaultGroqSTTModel    = transcription.DefaultGroqSTTModel
	DefaultMistralSTTModel = transcription.DefaultMistralSTTModel
	DefaultXAISTTModel     = transcription.DefaultXAISTTModel
	DefaultSTTTimeout      = transcription.DefaultSTTTimeout
)

type TranscriptionEvidence = transcription.TranscriptionEvidence
type TranscriptionConfig = transcription.TranscriptionConfig
type TranscriptionRequest = transcription.TranscriptionRequest
type TranscriptionResult = transcription.TranscriptionResult
type TranscriptionProviderRequest = transcription.TranscriptionProviderRequest
type TranscriptionProviderResult = transcription.TranscriptionProviderResult
type TranscriptionProvider = transcription.TranscriptionProvider
type TranscriptionRunner = transcription.TranscriptionRunner
type TranscriptionTool = transcription.TranscriptionTool
type TranscriptionProviderConfig = transcription.TranscriptionProviderConfig
type TranscriptionOpenAIProvider = transcription.TranscriptionOpenAIProvider
type TranscriptionGroqProvider = transcription.TranscriptionGroqProvider
type TranscriptionMistralProvider = transcription.TranscriptionMistralProvider
type TranscriptionXAIProvider = transcription.TranscriptionXAIProvider
type LocalSTTProvider = transcription.LocalSTTProvider

func NewTranscriptionRunner(cfg TranscriptionConfig, providers map[string]TranscriptionProvider) *TranscriptionRunner {
	return transcription.NewTranscriptionRunner(cfg, providers)
}

func NewTranscriptionTool(runner *TranscriptionRunner) *TranscriptionTool {
	return transcription.NewTranscriptionTool(runner)
}

func BuiltinTranscriptionProviderNames() []string {
	return transcription.BuiltinTranscriptionProviderNames()
}

func NewTranscriptionOpenAIProvider(config TranscriptionProviderConfig) *TranscriptionOpenAIProvider {
	return transcription.NewTranscriptionOpenAIProvider(config)
}

func NewTranscriptionGroqProvider(config TranscriptionProviderConfig) *TranscriptionGroqProvider {
	return transcription.NewTranscriptionGroqProvider(config)
}

func NewTranscriptionMistralProvider(config TranscriptionProviderConfig) *TranscriptionMistralProvider {
	return transcription.NewTranscriptionMistralProvider(config)
}

func NewTranscriptionXAIProvider(config TranscriptionProviderConfig) *TranscriptionXAIProvider {
	return transcription.NewTranscriptionXAIProvider(config)
}

func RegisterTranscriptionProviders(into map[string]TranscriptionProvider, cfg TranscriptionProviderConfig) {
	transcription.RegisterTranscriptionProviders(into, cfg)
}

func ValidateTranscriptionProviderConfig(provider string, cfg TranscriptionProviderConfig) error {
	return transcription.ValidateTranscriptionProviderConfig(provider, cfg)
}

func NewLocalSTTProvider(cacheDir string) *LocalSTTProvider {
	return transcription.NewLocalSTTProvider(cacheDir)
}
