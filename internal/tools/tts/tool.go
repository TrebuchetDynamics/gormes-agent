//go:build !slim

package tts

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/tools/toolkit"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tools/tts/contracts"
)

const (
	defaultTTSProvider      = "edge"
	defaultTTSMaxTextLength = 4000
)

// TTSEvidence is stable operator-facing evidence for TTS outcomes.
type TTSEvidence = contracts.Evidence

const (
	TTSEvidenceOK                     = contracts.EvidenceOK
	TTSEvidenceDisabled               = contracts.EvidenceDisabled
	TTSEvidenceInvalidArguments       = contracts.EvidenceInvalidArguments
	TTSEvidenceUnsupportedAudioFormat = contracts.EvidenceUnsupportedAudioFormat
	TTSEvidenceProviderUnavailable    = contracts.EvidenceProviderUnavailable
	TTSEvidenceAPIError               = contracts.EvidenceAPIError
	TTSEvidenceOutputMissing          = contracts.EvidenceOutputMissing
)

// TTSConfig controls the native text-to-speech helper. Provider is optional;
// empty uses Hermes' default edge provider when available.
type TTSConfig struct {
	Disabled       bool
	Provider       string
	OutputDir      string
	MaxTextLength  int
	ProviderConfig map[string]any
	Now            func() time.Time
}

// TTSRequest is the public helper input. Provider is intentionally not exposed
// in the model-facing schema yet; production config chooses the provider.
type TTSRequest = contracts.Request

// TTSResult is the redacted helper/tool result envelope.
type TTSResult = contracts.Result

// TTSProviderRequest is the normalized provider call input.
type TTSProviderRequest = contracts.ProviderRequest

// TTSProviderResult is the provider-specific response before the runner
// normalizes it into TTSResult.
type TTSProviderResult = contracts.ProviderResult

// TTSProvider is implemented by real or fake synthesis backends. Tests use
// fakes; production can inject local-command or HTTP providers without changing
// gateway/channel contracts.
type TTSProvider = contracts.Provider

// TTSRunner validates text/output paths and dispatches to injected providers.
type TTSRunner struct {
	cfg       TTSConfig
	providers map[string]TTSProvider
}

func NewTTSRunner(cfg TTSConfig, providers map[string]TTSProvider) *TTSRunner {
	cloned := make(map[string]TTSProvider, len(providers))
	for name, provider := range providers {
		key := normalizeTTSProviderName(name)
		if key != "" && provider != nil {
			cloned[key] = provider
		}
	}
	return &TTSRunner{cfg: cfg, providers: cloned}
}

func (r *TTSRunner) Synthesize(ctx context.Context, req TTSRequest) TTSResult {
	if r == nil {
		return ttsFailure("", TTSEvidenceProviderUnavailable, "no TTS runner configured")
	}
	cfg := r.cfg
	if cfg.Disabled {
		return ttsFailure("", TTSEvidenceDisabled, "TTS is disabled")
	}
	if strings.TrimSpace(req.Text) == "" {
		return ttsFailure("", TTSEvidenceInvalidArguments, "text is required")
	}
	providerName, provider, evidence := r.selectProvider(ctx, req.Provider)
	if evidence != "" {
		requested := firstNonEmptyTTS(req.Provider, cfg.Provider, defaultTTSProvider)
		return ttsFailure(requested, evidence, "no TTS provider available")
	}
	text, truncated := normalizeTTSText(req.Text, cfg.maxTextLength(providerName, provider))
	if text == "" {
		return ttsFailure(providerName, TTSEvidenceInvalidArguments, "text is required")
	}
	outputPath, validation := r.outputPath(req.OutputPath, providerName, provider, req.Platform)
	if validation.Evidence != "" {
		return validation
	}
	if err := os.MkdirAll(filepath.Dir(outputPath), 0o700); err != nil {
		return ttsFailure(providerName, TTSEvidenceInvalidArguments, redactTTSText(err.Error()))
	}

	providerResult, err := provider.Synthesize(ctx, TTSProviderRequest{
		Text:       text,
		OutputPath: outputPath,
		Provider:   providerName,
		Platform:   strings.ToLower(strings.TrimSpace(req.Platform)),
		Voice:      strings.TrimSpace(req.Voice),
		Speed:      req.Speed,
	})
	if err != nil {
		return ttsFailure(providerName, TTSEvidenceAPIError, redactTTSText(err.Error()))
	}
	filePath := firstNonEmptyTTS(providerResult.FilePath, outputPath)
	if validation := validateTTSOutputFile(providerName, filePath); validation.Evidence != "" {
		return validation
	}
	voiceCompatible := providerResult.VoiceCompatible || shouldTreatTTSAsVoiceCompatible(providerName, filePath, req.Platform)
	mediaTag := "MEDIA:" + filePath
	if voiceCompatible {
		mediaTag = "[[audio_as_voice]]\n" + mediaTag
	}
	return TTSResult{
		Success:         true,
		FilePath:        filePath,
		MediaTag:        mediaTag,
		Provider:        firstNonEmptyTTS(providerResult.Provider, providerName),
		VoiceCompatible: voiceCompatible,
		Truncated:       truncated,
		Evidence:        TTSEvidenceOK,
	}
}

func (c TTSConfig) maxTextLength(providerName string, provider TTSProvider) int {
	if c.MaxTextLength > 0 {
		return c.MaxTextLength
	}
	if limiter, ok := provider.(interface{ MaxTextLength() int }); ok {
		if maxLen := limiter.MaxTextLength(); maxLen > 0 {
			return maxLen
		}
	}
	return TTSProviderMaxTextLengthForConfig(providerName, c.ProviderConfig)
}

func (r *TTSRunner) selectProvider(ctx context.Context, requested string) (string, TTSProvider, TTSEvidence) {
	explicit := normalizeTTSProviderName(firstNonEmptyTTS(requested, r.cfg.Provider, defaultTTSProvider))
	if explicit == "local" {
		return r.selectAvailableProvider(ctx, []string{ProviderNameLocalGo, ProviderNameLocalFixture, "neutts", "kittentts", "piper"})
	}
	if explicit != "" && explicit != "auto" {
		provider := r.providers[explicit]
		if provider == nil || !provider.Available(ctx) {
			return "", nil, TTSEvidenceProviderUnavailable
		}
		return explicit, provider, ""
	}
	for _, name := range []string{"edge", "openai", "elevenlabs", "minimax", "mistral", "xai", "gemini", ProviderNameLocalGo, ProviderNameLocalFixture, "neutts", "kittentts", "piper"} {
		provider := r.providers[name]
		if provider != nil && provider.Available(ctx) {
			return name, provider, ""
		}
	}
	for name, provider := range r.providers {
		if isBuiltinTTSProviderName(name) {
			continue
		}
		if provider != nil && provider.Available(ctx) {
			return name, provider, ""
		}
	}
	return "", nil, TTSEvidenceProviderUnavailable
}

func (r *TTSRunner) selectAvailableProvider(ctx context.Context, names []string) (string, TTSProvider, TTSEvidence) {
	for _, name := range names {
		provider := r.providers[name]
		if provider != nil && provider.Available(ctx) {
			return name, provider, ""
		}
	}
	return "", nil, TTSEvidenceProviderUnavailable
}

func (r *TTSRunner) outputPath(raw, provider string, ttsProvider TTSProvider, platform string) (string, TTSResult) {
	value := strings.TrimSpace(raw)
	if strings.ContainsRune(value, 0) {
		return "", ttsFailure(provider, TTSEvidenceInvalidArguments, "output path contains NUL")
	}
	if value == "" {
		outputDir := strings.TrimSpace(r.cfg.OutputDir)
		if outputDir == "" {
			outputDir = filepath.Join(os.TempDir(), "gormes-audio-cache")
		}
		now := time.Now().UTC()
		if r.cfg.Now != nil {
			now = r.cfg.Now().UTC()
		}
		ext := preferredTTSAudioExt(ttsProvider)
		if ext == "" {
			ext = ".mp3"
		}
		if ext == ".mp3" && shouldPreferOpusForTTS(provider, platform) {
			ext = ".ogg"
		}
		value = filepath.Join(outputDir, "tts_"+now.Format("20060102_150405")+ext)
	}
	cleaned := filepath.Clean(value)
	if !supportedTTSAudioExt(filepath.Ext(cleaned)) {
		return "", ttsFailure(provider, TTSEvidenceUnsupportedAudioFormat, "unsupported audio format")
	}
	return cleaned, TTSResult{}
}

func preferredTTSAudioExt(provider TTSProvider) string {
	formatter, ok := provider.(interface{ PreferredOutputFormat() string })
	if !ok {
		return ""
	}
	format := strings.TrimPrefix(strings.ToLower(strings.TrimSpace(formatter.PreferredOutputFormat())), ".")
	if format == "" || !isSupportedCommandTTSOutputFormat(format) {
		return ""
	}
	return "." + format
}

func normalizeTTSText(text string, maxLen int) (string, bool) {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return "", false
	}
	runes := []rune(trimmed)
	if maxLen > 0 && len(runes) > maxLen {
		return string(runes[:maxLen]), true
	}
	return trimmed, false
}

func validateTTSOutputFile(provider, path string) TTSResult {
	info, err := os.Stat(path)
	if err != nil {
		return ttsFailure(provider, TTSEvidenceOutputMissing, "TTS generation produced no output")
	}
	if !info.Mode().IsRegular() || info.Size() == 0 {
		return ttsFailure(provider, TTSEvidenceOutputMissing, "TTS generation produced no output")
	}
	return TTSResult{}
}

func shouldPreferOpusForTTS(provider, platform string) bool {
	if !isTelegramTTSPlatform(platform) {
		return false
	}
	switch normalizeTTSProviderName(provider) {
	case "openai", "elevenlabs", "mistral", "gemini":
		return true
	default:
		return false
	}
}

func shouldTreatTTSAsVoiceCompatible(provider, path, platform string) bool {
	return isTelegramTTSPlatform(platform) && strings.EqualFold(filepath.Ext(path), ".ogg")
}

func isTelegramTTSPlatform(platform string) bool {
	platform = strings.ToLower(strings.TrimSpace(platform))
	return platform == "telegram" || strings.HasPrefix(platform, "telegram:")
}

func supportedTTSAudioExt(ext string) bool {
	switch strings.ToLower(strings.TrimSpace(ext)) {
	case ".mp3", ".ogg", ".opus", ".wav", ".m4a", ".aac", ".flac":
		return true
	default:
		return false
	}
}

func normalizeTTSProviderName(provider string) string {
	return strings.ToLower(strings.TrimSpace(provider))
}

func parseTTSRequestSpeed(raw json.RawMessage) float64 {
	if len(raw) == 0 || string(raw) == "null" {
		return 0
	}
	var numeric float64
	if err := json.Unmarshal(raw, &numeric); err == nil && numeric > 0 {
		return numeric
	}
	var text string
	if err := json.Unmarshal(raw, &text); err != nil {
		return 0
	}
	switch strings.ToLower(strings.TrimSpace(text)) {
	case "slow":
		return 0.75
	case "normal":
		return 1.0
	case "fast":
		return 1.25
	case "very-fast", "veryfast", "very_fast", "very fast":
		return 1.5
	default:
		return parseSpeed(text)
	}
}

func ttsFailure(provider string, evidence TTSEvidence, message string) TTSResult {
	return TTSResult{
		Success:  false,
		Provider: strings.TrimSpace(provider),
		Evidence: evidence,
		Error:    redactTTSText(message),
	}
}

var ttsSecretPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)\bBearer\s+[A-Za-z0-9._~+/=-]+`),
	regexp.MustCompile(`(?i)\b(sk|key|token|secret)[-_]?[A-Za-z0-9]*[=:]\s*["']?[^"'\s]+`),
	regexp.MustCompile(`\b[A-Za-z0-9_\-]{20,}\.[A-Za-z0-9_\-]{20,}\.[A-Za-z0-9_\-]{20,}\b`),
}

func redactTTSText(text string) string {
	redacted := strings.TrimSpace(text)
	for _, pattern := range ttsSecretPatterns {
		redacted = pattern.ReplaceAllString(redacted, "[redacted]")
	}
	if len(redacted) > 240 {
		redacted = redacted[:240] + "..."
	}
	if redacted == "" {
		return "redacted TTS error"
	}
	return redacted
}

func firstNonEmptyTTS(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

// TextToSpeechTool exposes the helper through the standard Go-native tool
// contract. Provider implementations stay injected through TTSRunner.
type TextToSpeechTool struct {
	runner *TTSRunner
}

func NewTextToSpeechTool(runner *TTSRunner) *TextToSpeechTool {
	return &TextToSpeechTool{runner: runner}
}

func (*TextToSpeechTool) Name() string { return "text_to_speech" }

func (*TextToSpeechTool) Description() string {
	return "Convert text to speech audio. Returns a MEDIA tag for platform-native delivery."
}

func (*TextToSpeechTool) Schema() json.RawMessage {
	return json.RawMessage(`{"type":"object","properties":{"text":{"type":"string","description":"The text to convert to speech. Provider-specific character caps apply and are enforced automatically; over-long input is truncated."},"output_path":{"type":"string","description":"Optional custom file path to save the audio. Defaults to the Gormes audio cache."}},"required":["text"]}`)
}

func (*TextToSpeechTool) Timeout() time.Duration { return 90 * time.Second }

func (t *TextToSpeechTool) Execute(ctx context.Context, args json.RawMessage) (json.RawMessage, error) {
	var in struct {
		Text       string          `json:"text"`
		OutputPath string          `json:"output_path"`
		Provider   string          `json:"provider"`
		Platform   string          `json:"platform"`
		Voice      string          `json:"voice"`
		Speed      json.RawMessage `json:"speed"`
	}
	if err := json.Unmarshal(args, &in); err != nil {
		result := ttsFailure("", TTSEvidenceInvalidArguments, "invalid TTS args: "+err.Error())
		return json.Marshal(result)
	}
	result := t.runner.Synthesize(ctx, TTSRequest{
		Text:       in.Text,
		OutputPath: in.OutputPath,
		Provider:   in.Provider,
		Platform:   in.Platform,
		Voice:      in.Voice,
		Speed:      parseTTSRequestSpeed(in.Speed),
	})
	return json.Marshal(result)
}

// Ensure compile-time tool conformance.
var _ toolkit.Tool = (*TextToSpeechTool)(nil)

func formatTTSCommandError(err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("TTS generation failed: %s", redactTTSText(err.Error()))
}
