//go:build !slim

package tools

import (
	"context"

	"github.com/TrebuchetDynamics/gormes-agent/internal/tools/tts"
)

const (
	TTSEvidenceOK                     TTSEvidence = tts.TTSEvidenceOK
	TTSEvidenceDisabled               TTSEvidence = tts.TTSEvidenceDisabled
	TTSEvidenceInvalidArguments       TTSEvidence = tts.TTSEvidenceInvalidArguments
	TTSEvidenceUnsupportedAudioFormat TTSEvidence = tts.TTSEvidenceUnsupportedAudioFormat
	TTSEvidenceProviderUnavailable    TTSEvidence = tts.TTSEvidenceProviderUnavailable
	TTSEvidenceAPIError               TTSEvidence = tts.TTSEvidenceAPIError
	TTSEvidenceOutputMissing          TTSEvidence = tts.TTSEvidenceOutputMissing

	DefaultEdgeTTSVoice       = tts.DefaultEdgeTTSVoice
	DefaultEdgeTTSRegion      = tts.DefaultEdgeTTSRegion
	DefaultOpenAIVoice        = tts.DefaultOpenAIVoice
	DefaultOpenAIModel        = tts.DefaultOpenAIModel
	DefaultOpenAIBaseURL      = tts.DefaultOpenAIBaseURL
	DefaultEdgeTTSBaseURL     = tts.DefaultEdgeTTSBaseURL
	DefaultEdgeTTSContentType = tts.DefaultEdgeTTSContentType
	DefaultOpenAIContentType  = tts.DefaultOpenAIContentType
	DefaultMiniMaxTTSModel    = tts.DefaultMiniMaxTTSModel
	DefaultMiniMaxTTSVoiceID  = tts.DefaultMiniMaxTTSVoiceID
	DefaultMiniMaxTTSBaseURL  = tts.DefaultMiniMaxTTSBaseURL

	ProviderNameEdge       = tts.ProviderNameEdge
	ProviderNameOpenAI     = tts.ProviderNameOpenAI
	ProviderNameElevenLabs = tts.ProviderNameElevenLabs
	ProviderNameMiniMax    = tts.ProviderNameMiniMax
	ProviderNameGemini     = tts.ProviderNameGemini
	ProviderNameNeuTTS     = tts.ProviderNameNeuTTS
	ProviderNameKittenTTS  = tts.ProviderNameKittenTTS
	ProviderNamePiper      = tts.ProviderNamePiper

	MaxTextLengthEdge       = tts.MaxTextLengthEdge
	MaxTextLengthOpenAI     = tts.MaxTextLengthOpenAI
	MaxTextLengthXAI        = tts.MaxTextLengthXAI
	MaxTextLengthMiniMax    = tts.MaxTextLengthMiniMax
	MaxTextLengthMistral    = tts.MaxTextLengthMistral
	MaxTextLengthGemini     = tts.MaxTextLengthGemini
	MaxTextLengthElevenLabs = tts.MaxTextLengthElevenLabs
	MaxTextLengthNeuTTS     = tts.MaxTextLengthNeuTTS
	MaxTextLengthKittenTTS  = tts.MaxTextLengthKittenTTS
	MaxTextLengthPiper      = tts.MaxTextLengthPiper
)

type TTSEvidence = tts.TTSEvidence
type TTSConfig = tts.TTSConfig
type TTSRequest = tts.TTSRequest
type TTSResult = tts.TTSResult
type TTSProviderRequest = tts.TTSProviderRequest
type TTSProviderResult = tts.TTSProviderResult
type TTSProvider = tts.TTSProvider
type TTSRunner = tts.TTSRunner
type TextToSpeechTool = tts.TextToSpeechTool
type TTSProviderConfig = tts.TTSProviderConfig
type TTSProviderCredential = tts.TTSProviderCredential
type TTSEdgeProvider = tts.TTSEdgeProvider
type TTSOpenAIProvider = tts.TTSOpenAIProvider
type TTSMiniMaxProvider = tts.TTSMiniMaxProvider
type TTSProviderStatus = tts.TTSProviderStatus
type LazyLocalTTSProvider = tts.LazyLocalTTSProvider
type TTSCommandProviderConfig = tts.TTSCommandProviderConfig
type TTSCommandExecution = tts.TTSCommandExecution
type TTSCommandRunner = tts.TTSCommandRunner
type TTSCommandProvider = tts.TTSCommandProvider
type EdgeTTSCommandProvider = tts.EdgeTTSCommandProvider

func NewTTSRunner(cfg TTSConfig, providers map[string]TTSProvider) *TTSRunner {
	return tts.NewTTSRunner(cfg, providers)
}

func NewTextToSpeechTool(runner *TTSRunner) *TextToSpeechTool {
	return tts.NewTextToSpeechTool(runner)
}

func BuiltinTTSProviderNames() []string { return tts.BuiltinTTSProviderNames() }

func ResolveTTSProviderCredential(provider string, cfg TTSProviderConfig) TTSProviderCredential {
	return tts.ResolveTTSProviderCredential(provider, cfg)
}

func NewTTSEdgeProvider(config TTSProviderConfig) *TTSEdgeProvider {
	return tts.NewTTSEdgeProvider(config)
}

func NewTTSOpenAIProvider(config TTSProviderConfig) *TTSOpenAIProvider {
	return tts.NewTTSOpenAIProvider(config)
}

func NewTTSMiniMaxProvider(config TTSProviderConfig) *TTSMiniMaxProvider {
	return tts.NewTTSMiniMaxProvider(config)
}

func RegisterTTSProviders(into map[string]TTSProvider, cfg TTSProviderConfig) {
	tts.RegisterTTSProviders(into, cfg)
}

func ValidateTTSProviderConfig(provider string, cfg TTSProviderConfig) error {
	return tts.ValidateTTSProviderConfig(provider, cfg)
}

func TTSProviderMaxTextLength(provider string) int { return tts.TTSProviderMaxTextLength(provider) }

func TTSProviderMaxTextLengthForConfig(provider string, ttsConfig map[string]any) int {
	return tts.TTSProviderMaxTextLengthForConfig(provider, ttsConfig)
}

func TTSSpeedForProvider(provider string, ttsConfig map[string]any) float64 {
	return tts.TTSSpeedForProvider(provider, ttsConfig)
}

func EdgeTTSRateString(speed float64) string { return tts.EdgeTTSRateString(speed) }

func OpenAITTSSpeed(speed float64) float64 { return tts.OpenAITTSSpeed(speed) }

func NewLazyLocalTTSProvider(provider string, probe func(context.Context) error) *LazyLocalTTSProvider {
	return tts.NewLazyLocalTTSProvider(provider, probe)
}

func NewTTSCommandProvider(name string, cfg TTSCommandProviderConfig, runner TTSCommandRunner) TTSProvider {
	return tts.NewTTSCommandProvider(name, cfg, runner)
}

func ResolveTTSCommandProviderConfig(provider string, ttsConfig map[string]any) (TTSCommandProviderConfig, bool) {
	return tts.ResolveTTSCommandProviderConfig(provider, ttsConfig)
}

func RegisterTTSCommandProviders(into map[string]TTSProvider, ttsConfig map[string]any, runner TTSCommandRunner) {
	tts.RegisterTTSCommandProviders(into, ttsConfig, runner)
}

func NewEdgeTTSCommandProviderFromEnv() TTSProvider { return tts.NewEdgeTTSCommandProviderFromEnv() }
