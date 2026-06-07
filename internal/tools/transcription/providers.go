//go:build !slim

package transcription

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/tools/transcription/cloud"
)

// Default STT provider configuration values
const (
	// Provider names (normalized) - reuse TTS constants where applicable
	ProviderNameDevice  = "device"
	ProviderNameLocal   = "local"
	ProviderNameGroq    = "groq"
	ProviderNameMistral = "mistral"
	ProviderNameXAI     = "xai"
	ProviderNameOpenAI  = "openai"

	// Default STT API endpoints
	DefaultOpenAIBaseURL  = "https://api.openai.com/v1"
	DefaultGroqBaseURL    = "https://api.groq.com/openai/v1"
	DefaultMistralBaseURL = "https://api.mistral.ai/v1"
	DefaultXAIBaseURL     = "https://api.x.ai/v1"

	// Default STT models
	DefaultGroqSTTModel    = "whisper-large-v3-turbo"
	DefaultMistralSTTModel = "voxtral-mini-latest"
	DefaultXAISTTModel     = "grok-stt"
	// DefaultOpenAISTTModel is defined in transcription_tool.go

	// STT request timeout
	DefaultSTTTimeout = 120 * time.Second
)

// TranscriptionProviderConfig holds HTTP provider-specific settings.
type TranscriptionProviderConfig struct {
	// Common settings
	APIKey  string
	BaseURL string
	Model   string
	Timeout time.Duration
}

// httpTimeout returns the effective request timeout.
var builtinTranscriptionProviderOrder = []string{
	ProviderNameDevice,
	ProviderNameLocal,
	ProviderNameOpenAI,
	ProviderNameGroq,
	ProviderNameMistral,
	ProviderNameXAI,
}

func BuiltinTranscriptionProviderNames() []string {
	out := make([]string, len(builtinTranscriptionProviderOrder))
	copy(out, builtinTranscriptionProviderOrder)
	return out
}

// ---------------------------------------------------------------------------
// OpenAI Whisper Provider
// ---------------------------------------------------------------------------

// TranscriptionOpenAIProvider transcribes audio using OpenAI's Whisper API.
type TranscriptionOpenAIProvider struct {
	config TranscriptionProviderConfig
	client *http.Client
}

// NewTranscriptionOpenAIProvider creates an OpenAI Whisper provider.
// API key is read from GORMES_STT_OPENAI_KEY, OPENAI_API_KEY, or VOICE_TOOLS_OPENAI_KEY.
func NewTranscriptionOpenAIProvider(config TranscriptionProviderConfig) *TranscriptionOpenAIProvider {
	cfg := config
	if cfg.Timeout == 0 {
		cfg.Timeout = DefaultSTTTimeout
	}
	if cfg.APIKey == "" {
		cfg.APIKey = os.Getenv("GORMES_STT_OPENAI_KEY")
		if cfg.APIKey == "" {
			cfg.APIKey = os.Getenv("OPENAI_API_KEY")
		}
	}
	if cfg.BaseURL == "" {
		cfg.BaseURL = os.Getenv("GORMES_STT_OPENAI_BASE_URL")
		if cfg.BaseURL == "" {
			cfg.BaseURL = DefaultOpenAIBaseURL // from tts_providers.go
		}
	}
	if cfg.Model == "" {
		cfg.Model = defaultOpenAISTTModel
	}
	return &TranscriptionOpenAIProvider{
		config: cfg,
		client: &http.Client{Timeout: cfg.Timeout},
	}
}

// Available returns true when a non-empty API key is configured.
func (p *TranscriptionOpenAIProvider) Available(ctx context.Context) bool {
	return strings.TrimSpace(p.config.APIKey) != ""
}

// Transcribe sends audio to OpenAI Whisper API and returns the transcript.
func (p *TranscriptionOpenAIProvider) Transcribe(ctx context.Context, req TranscriptionProviderRequest) (TranscriptionProviderResult, error) {
	apiKey := strings.TrimSpace(p.config.APIKey)
	if apiKey == "" {
		return TranscriptionProviderResult{}, errors.New("OpenAI STT API key not configured")
	}

	model := p.config.Model
	if req.Model != "" && req.Model != model {
		model = req.Model
	}

	fields := []cloud.Field{
		{Name: "model", Value: model},
		{Name: "response_format", Value: "text"},
	}
	if req.Language != "" {
		fields = append(fields, cloud.Field{Name: "language", Value: req.Language})
	}
	resp, err := cloud.PostBearerMultipart(ctx, p.client, "OpenAI STT", p.config.BaseURL, "/audio/transcriptions", apiKey, req.AudioPath, fields)
	if err != nil {
		return TranscriptionProviderResult{}, err
	}
	defer resp.Body.Close()
	if err := cloud.RequireOK("OpenAI STT", resp); err != nil {
		return TranscriptionProviderResult{}, err
	}

	// We requested response_format=text above, so the body is the raw
	// transcript, not JSON. Reading the body as plain text matches the
	// Content-Type OpenAI returns for text-format requests and avoids the
	// "invalid character ... looking for beginning of value" parse failure
	// the Groq provider hit before the same fix landed. Same defect class:
	// the comment "OpenAI returns {\"text\": \"...\"}" was true ONLY when
	// no response_format was set; with response_format=text the body is raw.
	transcript, err := cloud.ReadTrimmedText("OpenAI STT", resp.Body)
	if err != nil {
		return TranscriptionProviderResult{}, err
	}
	return TranscriptionProviderResult{
		Transcript: transcript,
		Provider:   ProviderNameOpenAI, // from tts_providers.go
		Model:      model,
		Language:   req.Language,
	}, nil
}

// ---------------------------------------------------------------------------
// Groq Whisper Provider
// ---------------------------------------------------------------------------

// TranscriptionGroqProvider transcribes audio using Groq's Whisper API (free tier).
type TranscriptionGroqProvider struct {
	config TranscriptionProviderConfig
	client *http.Client
}

// NewTranscriptionGroqProvider creates a Groq Whisper provider.
// API key is read from GROQ_API_KEY.
func NewTranscriptionGroqProvider(config TranscriptionProviderConfig) *TranscriptionGroqProvider {
	cfg := config
	if cfg.Timeout == 0 {
		cfg.Timeout = DefaultSTTTimeout
	}
	if cfg.APIKey == "" {
		cfg.APIKey = os.Getenv("GROQ_API_KEY")
	}
	if cfg.BaseURL == "" {
		cfg.BaseURL = os.Getenv("GROQ_STT_BASE_URL")
		if cfg.BaseURL == "" {
			cfg.BaseURL = DefaultGroqBaseURL
		}
	}
	if cfg.Model == "" {
		cfg.Model = DefaultGroqSTTModel
	}
	return &TranscriptionGroqProvider{
		config: cfg,
		client: &http.Client{Timeout: cfg.Timeout},
	}
}

// Available returns true when a non-empty API key is configured.
func (p *TranscriptionGroqProvider) Available(ctx context.Context) bool {
	return strings.TrimSpace(p.config.APIKey) != ""
}

// Transcribe sends audio to Groq Whisper API and returns the transcript.
func (p *TranscriptionGroqProvider) Transcribe(ctx context.Context, req TranscriptionProviderRequest) (TranscriptionProviderResult, error) {
	apiKey := strings.TrimSpace(p.config.APIKey)
	if apiKey == "" {
		return TranscriptionProviderResult{}, errors.New("Groq STT API key not configured")
	}

	model := p.config.Model
	if req.Model != "" && req.Model != model {
		model = req.Model
	}

	resp, err := cloud.PostBearerMultipart(ctx, p.client, "Groq STT", p.config.BaseURL, "/audio/transcriptions", apiKey, req.AudioPath, []cloud.Field{
		{Name: "model", Value: model},
		{Name: "response_format", Value: "text"},
	})
	if err != nil {
		return TranscriptionProviderResult{}, err
	}
	defer resp.Body.Close()
	if err := cloud.RequireOK("Groq STT", resp); err != nil {
		return TranscriptionProviderResult{}, err
	}

	// We requested response_format=text above, so the body is the raw
	// transcript, not JSON. Reading the body as plain text matches the
	// Content-Type Groq returns for text-format requests and avoids
	// "invalid character ... looking for beginning of value" JSON parse
	// errors that fire on the first non-{ character of a real transcript.
	transcript, err := cloud.ReadTrimmedText("Groq STT", resp.Body)
	if err != nil {
		return TranscriptionProviderResult{}, err
	}
	return TranscriptionProviderResult{
		Transcript: transcript,
		Provider:   ProviderNameGroq,
		Model:      model,
		Language:   req.Language,
	}, nil
}

// ---------------------------------------------------------------------------
// Mistral Voxtral Provider
// ---------------------------------------------------------------------------

// TranscriptionMistralProvider transcribes audio using Mistral's Voxtral API.
type TranscriptionMistralProvider struct {
	config TranscriptionProviderConfig
	client *http.Client
}

// NewTranscriptionMistralProvider creates a Mistral Voxtral provider.
// API key is read from MISTRAL_API_KEY.
func NewTranscriptionMistralProvider(config TranscriptionProviderConfig) *TranscriptionMistralProvider {
	cfg := config
	if cfg.Timeout == 0 {
		cfg.Timeout = DefaultSTTTimeout
	}
	if cfg.APIKey == "" {
		cfg.APIKey = os.Getenv("MISTRAL_API_KEY")
	}
	if cfg.BaseURL == "" {
		cfg.BaseURL = os.Getenv("MISTRAL_STT_BASE_URL")
		if cfg.BaseURL == "" {
			cfg.BaseURL = DefaultMistralBaseURL
		}
	}
	if cfg.Model == "" {
		cfg.Model = DefaultMistralSTTModel
	}
	return &TranscriptionMistralProvider{
		config: cfg,
		client: &http.Client{Timeout: cfg.Timeout},
	}
}

// Available returns true when a non-empty API key is configured.
func (p *TranscriptionMistralProvider) Available(ctx context.Context) bool {
	return strings.TrimSpace(p.config.APIKey) != ""
}

// Transcribe sends audio to Mistral Voxtral API and returns the transcript.
func (p *TranscriptionMistralProvider) Transcribe(ctx context.Context, req TranscriptionProviderRequest) (TranscriptionProviderResult, error) {
	apiKey := strings.TrimSpace(p.config.APIKey)
	if apiKey == "" {
		return TranscriptionProviderResult{}, errors.New("Mistral STT API key not configured")
	}

	model := p.config.Model
	if req.Model != "" && req.Model != model {
		model = req.Model
	}

	resp, err := cloud.PostBearerMultipart(ctx, p.client, "Mistral STT", p.config.BaseURL, "/audio/transcriptions", apiKey, req.AudioPath, []cloud.Field{
		{Name: "model", Value: model},
	})
	if err != nil {
		return TranscriptionProviderResult{}, err
	}
	defer resp.Body.Close()
	if err := cloud.RequireOK("Mistral STT", resp); err != nil {
		return TranscriptionProviderResult{}, err
	}

	// Mistral returns {"text": "..."} or similar structure
	var result struct {
		Text string `json:"text"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return TranscriptionProviderResult{}, fmt.Errorf("Mistral STT parse response: %w", err)
	}

	return TranscriptionProviderResult{
		Transcript: strings.TrimSpace(result.Text),
		Provider:   ProviderNameMistral,
		Model:      model,
		Language:   req.Language,
	}, nil
}

// ---------------------------------------------------------------------------
// xAI Grok STT Provider
// ---------------------------------------------------------------------------

// TranscriptionXAIProvider transcribes audio using xAI's Grok STT API.
type TranscriptionXAIProvider struct {
	config TranscriptionProviderConfig
	client *http.Client
}

// NewTranscriptionXAIProvider creates an xAI Grok STT provider.
// API key is read from XAI_API_KEY.
func NewTranscriptionXAIProvider(config TranscriptionProviderConfig) *TranscriptionXAIProvider {
	cfg := config
	if cfg.Timeout == 0 {
		cfg.Timeout = DefaultSTTTimeout
	}
	if cfg.APIKey == "" {
		cfg.APIKey = os.Getenv("XAI_API_KEY")
	}
	if cfg.BaseURL == "" {
		cfg.BaseURL = os.Getenv("XAI_STT_BASE_URL")
		if cfg.BaseURL == "" {
			cfg.BaseURL = DefaultXAIBaseURL
		}
	}
	// xAI Grok STT uses a fixed model name
	cfg.Model = DefaultXAISTTModel
	return &TranscriptionXAIProvider{
		config: cfg,
		client: &http.Client{Timeout: cfg.Timeout},
	}
}

// Available returns true when a non-empty API key is configured.
func (p *TranscriptionXAIProvider) Available(ctx context.Context) bool {
	return strings.TrimSpace(p.config.APIKey) != ""
}

// Transcribe sends audio to xAI Grok STT API and returns the transcript.
func (p *TranscriptionXAIProvider) Transcribe(ctx context.Context, req TranscriptionProviderRequest) (TranscriptionProviderResult, error) {
	apiKey := strings.TrimSpace(p.config.APIKey)
	if apiKey == "" {
		return TranscriptionProviderResult{}, errors.New("xAI STT API key not configured")
	}

	fields := []cloud.Field{{Name: "format", Value: "true"}}
	if req.Language != "" {
		fields = append(fields, cloud.Field{Name: "language", Value: req.Language})
	}
	resp, err := cloud.PostBearerMultipart(ctx, p.client, "xAI STT", p.config.BaseURL, "/v1/stt", apiKey, req.AudioPath, fields)
	if err != nil {
		return TranscriptionProviderResult{}, err
	}
	defer resp.Body.Close()
	if err := cloud.RequireOK("xAI STT", resp); err != nil {
		return TranscriptionProviderResult{}, err
	}

	// xAI returns {"text": "..."}
	var result struct {
		Text     string  `json:"text"`
		Language string  `json:"language,omitempty"`
		Duration float64 `json:"duration,omitempty"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return TranscriptionProviderResult{}, fmt.Errorf("xAI STT parse response: %w", err)
	}

	return TranscriptionProviderResult{
		Transcript: strings.TrimSpace(result.Text),
		Provider:   ProviderNameXAI,
		Model:      DefaultXAISTTModel,
		Language:   req.Language,
	}, nil
}

// ---------------------------------------------------------------------------
// Provider registration helpers
// ---------------------------------------------------------------------------

// RegisterTranscriptionProviders registers the built-in HTTP STT providers into a
// provider map. It skips nil providers (when API keys are absent).
func RegisterTranscriptionProviders(into map[string]TranscriptionProvider, cfg TranscriptionProviderConfig) {
	openai := NewTranscriptionOpenAIProvider(cfg)
	if openai.Available(context.Background()) {
		into[ProviderNameOpenAI] = openai
	}

	groq := NewTranscriptionGroqProvider(cfg)
	if groq.Available(context.Background()) {
		into[ProviderNameGroq] = groq
	}

	mistral := NewTranscriptionMistralProvider(cfg)
	if mistral.Available(context.Background()) {
		into[ProviderNameMistral] = mistral
	}

	xai := NewTranscriptionXAIProvider(cfg)
	if xai.Available(context.Background()) {
		into[ProviderNameXAI] = xai
	}
}

// ValidateTranscriptionProviderConfig checks that a provider name is valid and
// that required fields are present. Returns an error describing the problem, or
// nil if valid.
func ValidateTranscriptionProviderConfig(provider string, cfg TranscriptionProviderConfig) error {
	switch normalizeTranscriptionProviderName(provider) {
	case ProviderNameDevice, ProviderNameLocal:
		return nil
	case ProviderNameOpenAI: // from tts_providers.go
		if cfg.APIKey == "" {
			key := os.Getenv("GORMES_STT_OPENAI_KEY")
			if key == "" {
				key = os.Getenv("OPENAI_API_KEY")
			}
			if key == "" {
				key = os.Getenv("VOICE_TOOLS_OPENAI_KEY")
			}
			if key == "" {
				return errors.New("OpenAI STT requires GORMES_STT_OPENAI_KEY, OPENAI_API_KEY, or VOICE_TOOLS_OPENAI_KEY")
			}
		}
		return nil

	case ProviderNameGroq:
		if cfg.APIKey == "" {
			key := os.Getenv("GROQ_API_KEY")
			if key == "" {
				return errors.New("Groq STT requires GROQ_API_KEY")
			}
		}
		return nil

	case ProviderNameMistral:
		if cfg.APIKey == "" {
			key := os.Getenv("MISTRAL_API_KEY")
			if key == "" {
				return errors.New("Mistral STT requires MISTRAL_API_KEY")
			}
		}
		return nil

	case ProviderNameXAI:
		if cfg.APIKey == "" {
			key := os.Getenv("XAI_API_KEY")
			if key == "" {
				return errors.New("xAI STT requires XAI_API_KEY")
			}
		}
		return nil

	case "auto", "":
		return nil // Auto-selection is valid

	default:
		return fmt.Errorf("unknown STT provider %q", provider)
	}
}
