package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

// Default TTS configuration values
const (
	DefaultEdgeTTSVoice        = "en-US-AriaNeural"
	DefaultEdgeTTSRegion      = "eastus"
	DefaultOpenAIVoice        = "alloy"
	DefaultOpenAIModel        = "gpt-4o-mini-tts"
	DefaultOpenAIBaseURL      = "https://api.openai.com/v1"
	DefaultEdgeTTSBaseURL     = "https://%s.tts.speech.microsoft.com/cognitiveservices/v1"
	DefaultEdgeTTSContentType = "application/ssml+xml"
	DefaultOpenAIContentType  = "application/json"

	// Provider names (normalized)
	ProviderNameEdge   = "edge"
	ProviderNameOpenAI = "openai"

	// Max text lengths per provider
	MaxTextLengthEdge   = 5000
	MaxTextLengthOpenAI = 4096
)

// TTSProviderConfig holds HTTP provider-specific settings. It is passed to
// providers at construction time; zero value uses defaults.
type TTSProviderConfig struct {
	// Common settings
	Voice  string // Provider-specific voice identifier
	Speed  float64
	APIKey string // API key for HTTP providers
	Region string // Azure region for Edge TTS

	// OpenAI-specific
	OpenAIBaseURL string

	// Edge TTS-specific (alias for Region)
	EdgeTTSRegion string

	// Timeout for HTTP requests (0 = use default)
	Timeout time.Duration
}

// httpTimeout returns the effective request timeout.
func (c TTSProviderConfig) httpTimeout() time.Duration {
	if c.Timeout > 0 {
		return c.Timeout
	}
	return 60 * time.Second
}

// edgeTTSBaseURL returns the base URL for Edge TTS, allowing override for testing.
func (c TTSProviderConfig) edgeTTSBaseURL() string {
	if baseURL := os.Getenv("GORMES_TTS_EDGE_BASE_URL"); baseURL != "" {
		return baseURL
	}
	return fmt.Sprintf(DefaultEdgeTTSBaseURL, c.Region)
}

// TTSEdgeProvider is an HTTP-based Edge TTS provider using Azure Cognitive Services.
// It accepts plain text and synthesizes it via the Azure TTS REST API.
// Voice, speed, and API key are configured via TTSProviderConfig.
type TTSEdgeProvider struct {
	config TTSProviderConfig
	client *http.Client
}

// NewTTSEdgeProvider creates an Edge TTS provider from the environment or defaults.
// API key is read from GORMES_TTS_EDGE_KEY or GORMES_AZURE_TTS_KEY.
func NewTTSEdgeProvider(config TTSProviderConfig) *TTSEdgeProvider {
	cfg := config
	if cfg.Timeout == 0 {
		cfg.Timeout = 60 * time.Second
	}
	if cfg.Region == "" {
		cfg.Region = os.Getenv("GORMES_TTS_EDGE_REGION")
		if cfg.Region == "" {
			cfg.Region = DefaultEdgeTTSRegion
		}
	}
	if cfg.APIKey == "" {
		cfg.APIKey = os.Getenv("GORMES_TTS_EDGE_KEY")
		if cfg.APIKey == "" {
			cfg.APIKey = os.Getenv("GORMES_AZURE_TTS_KEY")
		}
	}
	if cfg.Voice == "" {
		cfg.Voice = DefaultEdgeTTSVoice
	}
	return &TTSEdgeProvider{
		config: cfg,
		client: &http.Client{Timeout: cfg.Timeout},
	}
}

// Available returns true when a non-empty API key is configured.
func (p *TTSEdgeProvider) Available(ctx context.Context) bool {
	return strings.TrimSpace(p.config.APIKey) != ""
}

// Synthesize converts text to speech via Azure Cognitive Services TTS API.
// It wraps the input text in SSML for voice and rate control.
func (p *TTSEdgeProvider) Synthesize(ctx context.Context, req TTSProviderRequest) (TTSProviderResult, error) {
	apiKey := strings.TrimSpace(p.config.APIKey)
	if apiKey == "" {
		return TTSProviderResult{}, errors.New("Edge TTS API key not configured")
	}

	voice := p.config.Voice
	if voice == "" {
		voice = DefaultEdgeTTSVoice
	}

	// Build SSML document
	ssml := buildEdgeTTSSSML(req.Text, voice, p.config.Speed)

	baseURL := p.config.edgeTTSBaseURL()
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL, strings.NewReader(ssml))
	if err != nil {
		return TTSProviderResult{}, fmt.Errorf("Edge TTS request: %w", err)
	}

	httpReq.Header.Set("Ocp-Apim-Subscription-Key", apiKey)
	httpReq.Header.Set("Content-Type", DefaultEdgeTTSContentType)
	httpReq.Header.Set("X-Microsoft-OutputFormat", "audio-24khz-48kbitrate-mono-mp3")

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return TTSProviderResult{}, fmt.Errorf("Edge TTS HTTP: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return TTSProviderResult{}, fmt.Errorf("Edge TTS HTTP %d: %s", resp.StatusCode, string(body))
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return TTSProviderResult{}, fmt.Errorf("Edge TTS read body: %w", err)
	}

	if err := os.WriteFile(req.OutputPath, data, 0o600); err != nil {
		return TTSProviderResult{}, fmt.Errorf("Edge TTS write file: %w", err)
	}

	return TTSProviderResult{
		FilePath:        req.OutputPath,
		Provider:        ProviderNameEdge,
		VoiceCompatible: false,
	}, nil
}

// buildEdgeTTSSSML creates an SSML document for Edge TTS.
func buildEdgeTTSSSML(text, voice string, speed float64) string {
	rate := "+0%"
	if speed != 1.0 && speed > 0 {
		pct := int((speed - 1.0) * 100)
		if pct > 0 {
			rate = fmt.Sprintf("+%d%%", pct)
		} else if pct < 0 {
			rate = fmt.Sprintf("%d%%", pct)
		}
	}
	return fmt.Sprintf(
		`<speak version='1.0' xmlns='http://www.w3.org/2001/10/synthesis' xml:lang='en-US'><voice name='%s'><prosody rate='%s'>%s</prosody></voice></speak>`,
		voice, rate, escapeSSML(text),
	)
}

// escapeSSML escapes special characters for SSML.
func escapeSSML(text string) string {
	text = strings.ReplaceAll(text, "&", "&amp;")
	text = strings.ReplaceAll(text, "<", "&lt;")
	text = strings.ReplaceAll(text, ">", "&gt;")
	text = strings.ReplaceAll(text, "\"", "&quot;")
	text = strings.ReplaceAll(text, "'", "&apos;")
	return text
}

// TTSOpenAIProvider is an HTTP-based OpenAI TTS provider.
type TTSOpenAIProvider struct {
	config TTSProviderConfig
	client *http.Client
}

// NewTTSOpenAIProvider creates an OpenAI TTS provider from the environment or defaults.
// API key is read from GORMES_TTS_OPENAI_KEY or OPENAI_API_KEY.
func NewTTSOpenAIProvider(config TTSProviderConfig) *TTSOpenAIProvider {
	cfg := config
	if cfg.Timeout == 0 {
		cfg.Timeout = 60 * time.Second
	}
	if cfg.APIKey == "" {
		cfg.APIKey = os.Getenv("GORMES_TTS_OPENAI_KEY")
		if cfg.APIKey == "" {
			cfg.APIKey = os.Getenv("OPENAI_API_KEY")
		}
	}
	if cfg.OpenAIBaseURL == "" {
		cfg.OpenAIBaseURL = os.Getenv("GORMES_TTS_OPENAI_BASE_URL")
		if cfg.OpenAIBaseURL == "" {
			cfg.OpenAIBaseURL = DefaultOpenAIBaseURL
		}
	}
	if cfg.Voice == "" {
		cfg.Voice = DefaultOpenAIVoice
	}
	return &TTSOpenAIProvider{
		config: cfg,
		client: &http.Client{Timeout: cfg.Timeout},
	}
}

// Available returns true when a non-empty API key is configured.
func (p *TTSOpenAIProvider) Available(ctx context.Context) bool {
	return strings.TrimSpace(p.config.APIKey) != ""
}

// Synthesize converts text to speech via OpenAI TTS API.
func (p *TTSOpenAIProvider) Synthesize(ctx context.Context, req TTSProviderRequest) (TTSProviderResult, error) {
	apiKey := strings.TrimSpace(p.config.APIKey)
	if apiKey == "" {
		return TTSProviderResult{}, errors.New("OpenAI TTS API key not configured")
	}

	model := DefaultOpenAIModel
	voice := p.config.Voice
	if voice == "" {
		voice = DefaultOpenAIVoice
	}

	// Determine output format from file extension
	format := "mp3"
	if strings.HasSuffix(strings.ToLower(req.OutputPath), ".ogg") ||
		strings.HasSuffix(strings.ToLower(req.OutputPath), ".opus") {
		format = "opus"
	}

	speed := p.config.Speed
	if speed <= 0 {
		speed = 1.0
	}
	// Clamp speed to OpenAI's valid range
	if speed < 0.25 {
		speed = 0.25
	}
	if speed > 4.0 {
		speed = 4.0
	}

	payload := map[string]any{
		"model":           model,
		"input":           req.Text,
		"voice":           voice,
		"response_format": format,
		"speed":           speed,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return TTSProviderResult{}, fmt.Errorf("OpenAI TTS payload: %w", err)
	}

	baseURL := strings.TrimSuffix(p.config.OpenAIBaseURL, "/")
	url := baseURL + "/v1/audio/speech"

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return TTSProviderResult{}, fmt.Errorf("OpenAI TTS request: %w", err)
	}

	httpReq.Header.Set("Authorization", "Bearer "+apiKey)
	httpReq.Header.Set("Content-Type", DefaultOpenAIContentType)

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return TTSProviderResult{}, fmt.Errorf("OpenAI TTS HTTP: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return TTSProviderResult{}, fmt.Errorf("OpenAI TTS HTTP %d: %s", resp.StatusCode, string(respBody))
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return TTSProviderResult{}, fmt.Errorf("OpenAI TTS read body: %w", err)
	}

	if err := os.WriteFile(req.OutputPath, data, 0o600); err != nil {
		return TTSProviderResult{}, fmt.Errorf("OpenAI TTS write file: %w", err)
	}

	return TTSProviderResult{
		FilePath:        req.OutputPath,
		Provider:        ProviderNameOpenAI,
		VoiceCompatible: format == "opus",
	}, nil
}

// RegisterTTSProviders registers the built-in HTTP TTS providers into a provider
// map. It skips nil providers (when API keys are absent).
func RegisterTTSProviders(into map[string]TTSProvider, cfg TTSProviderConfig) {
	edge := NewTTSEdgeProvider(cfg)
	if edge.Available(context.Background()) {
		into[ProviderNameEdge] = edge
	}

	openai := NewTTSOpenAIProvider(cfg)
	if openai.Available(context.Background()) {
		into[ProviderNameOpenAI] = openai
	}
}

// ValidateTTSProviderConfig checks that a provider name is valid and that required
// fields are present. Returns an error describing the problem, or nil if valid.
func ValidateTTSProviderConfig(provider string, cfg TTSProviderConfig) error {
	provider = normalizeTTSProviderName(provider)
	switch provider {
	case ProviderNameEdge:
		if cfg.APIKey == "" {
			// Check environment as fallback
			key := os.Getenv("GORMES_TTS_EDGE_KEY")
			if key == "" {
				key = os.Getenv("GORMES_AZURE_TTS_KEY")
			}
			if key == "" {
				return errors.New("Edge TTS requires GORMES_TTS_EDGE_KEY or GORMES_AZURE_TTS_KEY")
			}
		}
		return nil

	case ProviderNameOpenAI:
		if cfg.APIKey == "" {
			key := os.Getenv("GORMES_TTS_OPENAI_KEY")
			if key == "" {
				key = os.Getenv("OPENAI_API_KEY")
			}
			if key == "" {
				return errors.New("OpenAI TTS requires GORMES_TTS_OPENAI_KEY or OPENAI_API_KEY")
			}
		}
		return nil

	case "auto", "":
		return nil // Auto-selection is valid

	default:
		return fmt.Errorf("unknown TTS provider %q", provider)
	}
}

// TTSProviderMaxTextLength returns the maximum input length for a provider.
func TTSProviderMaxTextLength(provider string) int {
	switch normalizeTTSProviderName(provider) {
	case ProviderNameEdge:
		return MaxTextLengthEdge
	case ProviderNameOpenAI:
		return MaxTextLengthOpenAI
	default:
		return defaultTTSMaxTextLength
	}
}

// parseSpeed parses a speed value from string, returning 1.0 on error.
func parseSpeed(s string) float64 {
	v, err := strconv.ParseFloat(strings.TrimSpace(s), 64)
	if err != nil || v <= 0 {
		return 1.0
	}
	return v
}
