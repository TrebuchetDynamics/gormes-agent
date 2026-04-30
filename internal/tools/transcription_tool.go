package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

const (
	defaultTranscriptionMaxBytes = 25 * 1024 * 1024
	defaultLocalSTTModel         = "base"
	defaultGroqSTTModel          = "whisper-large-v3-turbo"
	defaultOpenAISTTModel        = "whisper-1"
	defaultMistralSTTModel       = "voxtral-mini-latest"
	defaultXAISTTModel           = "grok-stt"
)

// TranscriptionEvidence is stable operator-facing evidence for STT outcomes.
type TranscriptionEvidence string

const (
	TranscriptionEvidenceOK                     TranscriptionEvidence = "stt_transcribed"
	TranscriptionEvidenceDisabled               TranscriptionEvidence = "stt_disabled"
	TranscriptionEvidenceAudioNotFound          TranscriptionEvidence = "audio_not_found"
	TranscriptionEvidenceAudioNotFile           TranscriptionEvidence = "audio_not_file"
	TranscriptionEvidenceUnsupportedAudioFormat TranscriptionEvidence = "unsupported_audio_format"
	TranscriptionEvidenceAudioTooLarge          TranscriptionEvidence = "audio_too_large"
	TranscriptionEvidenceProviderUnavailable    TranscriptionEvidence = "stt_provider_unavailable"
	TranscriptionEvidenceAPIError               TranscriptionEvidence = "stt_api_error"
	TranscriptionEvidenceInvalidArguments       TranscriptionEvidence = "stt_invalid_arguments"
)

// TranscriptionConfig controls the native STT helper. Provider is optional;
// empty means auto-select from available injected providers using Hermes order.
type TranscriptionConfig struct {
	Disabled     bool
	Provider     string
	MaxBytes     int64
	LocalModel   string
	GroqModel    string
	OpenAIModel  string
	MistralModel string
	XAIModel     string
	Language     string
}

// TranscriptionRequest is the public helper/tool input.
type TranscriptionRequest struct {
	AudioPath string
	Provider  string
	Model     string
	Language  string
	Format    string
}

// TranscriptionResult is the redacted helper/tool result envelope.
type TranscriptionResult struct {
	Success    bool                  `json:"success"`
	Transcript string                `json:"transcript"`
	Provider   string                `json:"provider,omitempty"`
	Model      string                `json:"model,omitempty"`
	Language   string                `json:"language,omitempty"`
	Evidence   TranscriptionEvidence `json:"evidence"`
	Error      string                `json:"error,omitempty"`
}

// TranscriptionProviderRequest is the normalized provider call input.
type TranscriptionProviderRequest struct {
	AudioPath string
	Provider  string
	Model     string
	Language  string
	Format    string
}

// TranscriptionProviderResult is the provider-specific response before the
// runner normalizes it into TranscriptionResult.
type TranscriptionProviderResult struct {
	Transcript string
	Provider   string
	Model      string
	Language   string
}

// TranscriptionProvider is implemented by real or fake STT backends. Tests use
// fakes; production wiring can add HTTP/local-command providers without
// changing gateway/channel contracts.
type TranscriptionProvider interface {
	Available(context.Context) bool
	Transcribe(context.Context, TranscriptionProviderRequest) (TranscriptionProviderResult, error)
}

// TranscriptionRunner validates audio and dispatches to injected providers.
type TranscriptionRunner struct {
	cfg       TranscriptionConfig
	providers map[string]TranscriptionProvider
}

func NewTranscriptionRunner(cfg TranscriptionConfig, providers map[string]TranscriptionProvider) *TranscriptionRunner {
	cloned := make(map[string]TranscriptionProvider, len(providers))
	for name, provider := range providers {
		key := normalizeTranscriptionProviderName(name)
		if key != "" && provider != nil {
			cloned[key] = provider
		}
	}
	return &TranscriptionRunner{cfg: cfg, providers: cloned}
}

func (r *TranscriptionRunner) Transcribe(ctx context.Context, req TranscriptionRequest) TranscriptionResult {
	if r == nil {
		return transcriptionFailure("", "", "", TranscriptionEvidenceProviderUnavailable, "no STT runner configured")
	}
	cfg := r.cfg
	if cfg.Disabled {
		return transcriptionFailure("", "", firstNonEmptyTranscription(req.Language, cfg.Language), TranscriptionEvidenceDisabled, "STT is disabled")
	}
	audioPath := strings.TrimSpace(req.AudioPath)
	if validation := validateTranscriptionAudio(audioPath, cfg.maxBytes()); validation.Evidence != "" {
		return validation
	}
	providerName, provider, evidence := r.selectProvider(ctx, req.Provider)
	if evidence != "" {
		requested := firstNonEmptyTranscription(req.Provider, cfg.Provider, "auto")
		return transcriptionFailure(requested, "", firstNonEmptyTranscription(req.Language, cfg.Language), evidence, "no STT provider available")
	}
	model := normalizeTranscriptionModel(providerName, firstNonEmptyTranscription(req.Model, cfg.modelFor(providerName)))
	language := firstNonEmptyTranscription(req.Language, cfg.Language)
	providerResult, err := provider.Transcribe(ctx, TranscriptionProviderRequest{
		AudioPath: audioPath,
		Provider:  providerName,
		Model:     model,
		Language:  language,
		Format:    strings.TrimSpace(req.Format),
	})
	if err != nil {
		return transcriptionFailure(providerName, model, language, TranscriptionEvidenceAPIError, redactTranscriptionText(err.Error()))
	}
	transcript := strings.TrimSpace(providerResult.Transcript)
	if transcript == "" {
		return transcriptionFailure(providerName, model, language, TranscriptionEvidenceAPIError, "provider returned an empty transcript")
	}
	return TranscriptionResult{
		Success:    true,
		Transcript: transcript,
		Provider:   firstNonEmptyTranscription(providerResult.Provider, providerName),
		Model:      firstNonEmptyTranscription(providerResult.Model, model),
		Language:   firstNonEmptyTranscription(providerResult.Language, language),
		Evidence:   TranscriptionEvidenceOK,
	}
}

func (c TranscriptionConfig) maxBytes() int64 {
	if c.MaxBytes > 0 {
		return c.MaxBytes
	}
	return defaultTranscriptionMaxBytes
}

func (c TranscriptionConfig) modelFor(provider string) string {
	switch normalizeTranscriptionProviderName(provider) {
	case "local", "local_command":
		return c.LocalModel
	case "groq":
		return c.GroqModel
	case "openai":
		return c.OpenAIModel
	case "mistral":
		return c.MistralModel
	case "xai":
		return c.XAIModel
	default:
		return ""
	}
}

func (r *TranscriptionRunner) selectProvider(ctx context.Context, requested string) (string, TranscriptionProvider, TranscriptionEvidence) {
	explicit := normalizeTranscriptionProviderName(firstNonEmptyTranscription(requested, r.cfg.Provider))
	if explicit != "" && explicit != "auto" {
		provider := r.providers[explicit]
		if provider == nil || !provider.Available(ctx) {
			return "", nil, TranscriptionEvidenceProviderUnavailable
		}
		return explicit, provider, ""
	}
	for _, name := range []string{"local", "local_command", "groq", "openai", "mistral", "xai"} {
		provider := r.providers[name]
		if provider != nil && provider.Available(ctx) {
			return name, provider, ""
		}
	}
	return "", nil, TranscriptionEvidenceProviderUnavailable
}

func validateTranscriptionAudio(path string, maxBytes int64) TranscriptionResult {
	if path == "" {
		return transcriptionFailure("", "", "", TranscriptionEvidenceAudioNotFound, "audio path is required")
	}
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return transcriptionFailure("", "", "", TranscriptionEvidenceAudioNotFound, "audio file not found")
		}
		return transcriptionFailure("", "", "", TranscriptionEvidenceAudioNotFound, redactTranscriptionText(err.Error()))
	}
	if !info.Mode().IsRegular() {
		return transcriptionFailure("", "", "", TranscriptionEvidenceAudioNotFile, "audio path is not a regular file")
	}
	if !supportedTranscriptionExt(filepath.Ext(path)) {
		return transcriptionFailure("", "", "", TranscriptionEvidenceUnsupportedAudioFormat, "unsupported audio format")
	}
	if info.Size() > maxBytes {
		return transcriptionFailure("", "", "", TranscriptionEvidenceAudioTooLarge, fmt.Sprintf("audio file exceeds max bytes (%d)", maxBytes))
	}
	return TranscriptionResult{}
}

func supportedTranscriptionExt(ext string) bool {
	switch strings.ToLower(strings.TrimSpace(ext)) {
	case ".mp3", ".mp4", ".mpeg", ".mpga", ".m4a", ".wav", ".webm", ".ogg", ".aac", ".flac":
		return true
	default:
		return false
	}
}

func normalizeTranscriptionProviderName(provider string) string {
	return strings.ToLower(strings.TrimSpace(provider))
}

func normalizeTranscriptionModel(provider, model string) string {
	name := strings.TrimSpace(model)
	switch normalizeTranscriptionProviderName(provider) {
	case "local", "local_command":
		if name == "" || isCloudSTTModelName(name) {
			return defaultLocalSTTModel
		}
		return name
	case "groq":
		if name == "" || isOpenAISTTModelName(name) {
			return defaultGroqSTTModel
		}
		return name
	case "openai":
		if name == "" || isGroqSTTModelName(name) {
			return defaultOpenAISTTModel
		}
		return name
	case "mistral":
		if name == "" {
			return defaultMistralSTTModel
		}
		return name
	case "xai":
		if name == "" {
			return defaultXAISTTModel
		}
		return name
	default:
		return name
	}
}

func isCloudSTTModelName(model string) bool {
	return isOpenAISTTModelName(model) || isGroqSTTModelName(model)
}

func isOpenAISTTModelName(model string) bool {
	switch model {
	case "whisper-1", "gpt-4o-mini-transcribe", "gpt-4o-transcribe":
		return true
	default:
		return false
	}
}

func isGroqSTTModelName(model string) bool {
	switch model {
	case "whisper-large-v3", "whisper-large-v3-turbo", "distil-whisper-large-v3-en":
		return true
	default:
		return false
	}
}

func transcriptionFailure(provider, model, language string, evidence TranscriptionEvidence, message string) TranscriptionResult {
	return TranscriptionResult{
		Success:  false,
		Provider: strings.TrimSpace(provider),
		Model:    strings.TrimSpace(model),
		Language: strings.TrimSpace(language),
		Evidence: evidence,
		Error:    redactTranscriptionText(message),
	}
}

var transcriptionSecretPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)\bBearer\s+[A-Za-z0-9._~+/=-]+`),
	regexp.MustCompile(`(?i)\b(sk|key|token|secret)[-_]?[A-Za-z0-9]*[=:]\s*["']?[^"'\s]+`),
	regexp.MustCompile(`\b[A-Za-z0-9_\-]{20,}\.[A-Za-z0-9_\-]{20,}\.[A-Za-z0-9_\-]{20,}\b`),
}

func redactTranscriptionText(text string) string {
	redacted := strings.TrimSpace(text)
	for _, pattern := range transcriptionSecretPatterns {
		redacted = pattern.ReplaceAllString(redacted, "[redacted]")
	}
	if len(redacted) > 240 {
		redacted = redacted[:240] + "..."
	}
	if redacted == "" {
		return "redacted STT error"
	}
	return redacted
}

func firstNonEmptyTranscription(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

// TranscriptionTool exposes the helper through the standard Go-native tool
// contract. Provider implementations stay injected through TranscriptionRunner.
type TranscriptionTool struct {
	runner *TranscriptionRunner
}

func NewTranscriptionTool(runner *TranscriptionRunner) *TranscriptionTool {
	return &TranscriptionTool{runner: runner}
}

func (*TranscriptionTool) Name() string { return "transcribe_audio" }

func (*TranscriptionTool) Description() string {
	return "Transcribe a local audio file using the configured STT provider."
}

func (*TranscriptionTool) Schema() json.RawMessage {
	return json.RawMessage(`{"type":"object","properties":{"audio_path":{"type":"string","description":"Local path to the audio file to transcribe."},"provider":{"type":"string","enum":["auto","local","local_command","groq","openai","mistral","xai"],"description":"Optional STT provider. auto uses local, local_command, groq, openai, mistral, then xai from configured availability."},"model":{"type":"string","description":"Optional provider-specific STT model override."},"language":{"type":"string","description":"Optional BCP-47/ISO language hint such as en or es."},"format":{"type":"string","description":"Optional output/input format hint for provider adapters."}},"required":["audio_path"]}`)
}

func (*TranscriptionTool) Timeout() time.Duration { return 60 * time.Second }

func (t *TranscriptionTool) Execute(ctx context.Context, args json.RawMessage) (json.RawMessage, error) {
	var in struct {
		AudioPath string `json:"audio_path"`
		Provider  string `json:"provider"`
		Model     string `json:"model"`
		Language  string `json:"language"`
		Format    string `json:"format"`
	}
	if err := json.Unmarshal(args, &in); err != nil {
		result := transcriptionFailure("", "", "", TranscriptionEvidenceInvalidArguments, "invalid transcription args: "+err.Error())
		return json.Marshal(result)
	}
	result := t.runner.Transcribe(ctx, TranscriptionRequest{
		AudioPath: in.AudioPath,
		Provider:  in.Provider,
		Model:     in.Model,
		Language:  in.Language,
		Format:    in.Format,
	})
	return json.Marshal(result)
}
