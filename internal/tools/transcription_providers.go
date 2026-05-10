//go:build !slim

package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Default STT provider configuration values
const (
	// Provider names (normalized) - reuse TTS constants where applicable
	ProviderNameGroq    = "groq"
	ProviderNameMistral = "mistral"
	ProviderNameXAI     = "xai"
	// ProviderNameOpenAI and DefaultOpenAIBaseURL are defined in tts_providers.go

	// Default STT API endpoints
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
func (c TranscriptionProviderConfig) httpTimeout() time.Duration {
	if c.Timeout > 0 {
		return c.Timeout
	}
	return DefaultSTTTimeout
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

	// Build multipart form request
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// Add model field
	if err := writer.WriteField("model", model); err != nil {
		return TranscriptionProviderResult{}, fmt.Errorf("OpenAI STT field: %w", err)
	}

	// Add language hint if provided
	if req.Language != "" {
		if err := writer.WriteField("language", req.Language); err != nil {
			return TranscriptionProviderResult{}, fmt.Errorf("OpenAI STT language field: %w", err)
		}
	}

	// Add response_format - always text for simplicity
	if err := writer.WriteField("response_format", "text"); err != nil {
		return TranscriptionProviderResult{}, fmt.Errorf("OpenAI STT format field: %w", err)
	}

	// Add audio file
	f, err := os.Open(req.AudioPath)
	if err != nil {
		return TranscriptionProviderResult{}, fmt.Errorf("OpenAI STT open audio: %w", err)
	}
	defer f.Close()

	part, err := writer.CreateFormFile("file", filepath.Base(req.AudioPath))
	if err != nil {
		return TranscriptionProviderResult{}, fmt.Errorf("OpenAI STT create form file: %w", err)
	}
	if _, err := io.Copy(part, f); err != nil {
		return TranscriptionProviderResult{}, fmt.Errorf("OpenAI STT copy audio: %w", err)
	}
	if err := writer.Close(); err != nil {
		return TranscriptionProviderResult{}, fmt.Errorf("OpenAI STT close writer: %w", err)
	}

	baseURL := strings.TrimSuffix(p.config.BaseURL, "/")
	url := baseURL + "/v1/audio/transcriptions"

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, body)
	if err != nil {
		return TranscriptionProviderResult{}, fmt.Errorf("OpenAI STT request: %w", err)
	}

	httpReq.Header.Set("Authorization", "Bearer "+apiKey)
	httpReq.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return TranscriptionProviderResult{}, fmt.Errorf("OpenAI STT HTTP: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return TranscriptionProviderResult{}, fmt.Errorf("OpenAI STT HTTP %d: %s", resp.StatusCode, string(respBody))
	}

	// We requested response_format=text above, so the body is the raw
	// transcript, not JSON. Reading the body as plain text matches the
	// Content-Type OpenAI returns for text-format requests and avoids the
	// "invalid character ... looking for beginning of value" parse failure
	// the Groq provider hit before the same fix landed. Same defect class:
	// the comment "OpenAI returns {\"text\": \"...\"}" was true ONLY when
	// no response_format was set; with response_format=text the body is raw.
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return TranscriptionProviderResult{}, fmt.Errorf("OpenAI STT read response: %w", err)
	}
	return TranscriptionProviderResult{
		Transcript: strings.TrimSpace(string(bodyBytes)),
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

	// Build multipart form request
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	if err := writer.WriteField("model", model); err != nil {
		return TranscriptionProviderResult{}, fmt.Errorf("Groq STT field: %w", err)
	}
	if err := writer.WriteField("response_format", "text"); err != nil {
		return TranscriptionProviderResult{}, fmt.Errorf("Groq STT format field: %w", err)
	}

	f, err := os.Open(req.AudioPath)
	if err != nil {
		return TranscriptionProviderResult{}, fmt.Errorf("Groq STT open audio: %w", err)
	}
	defer f.Close()

	part, err := writer.CreateFormFile("file", filepath.Base(req.AudioPath))
	if err != nil {
		return TranscriptionProviderResult{}, fmt.Errorf("Groq STT create form file: %w", err)
	}
	if _, err := io.Copy(part, f); err != nil {
		return TranscriptionProviderResult{}, fmt.Errorf("Groq STT copy audio: %w", err)
	}
	if err := writer.Close(); err != nil {
		return TranscriptionProviderResult{}, fmt.Errorf("Groq STT close writer: %w", err)
	}

	baseURL := strings.TrimSuffix(p.config.BaseURL, "/")
	url := baseURL + "/audio/transcriptions"

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, body)
	if err != nil {
		return TranscriptionProviderResult{}, fmt.Errorf("Groq STT request: %w", err)
	}

	httpReq.Header.Set("Authorization", "Bearer "+apiKey)
	httpReq.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return TranscriptionProviderResult{}, fmt.Errorf("Groq STT HTTP: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return TranscriptionProviderResult{}, fmt.Errorf("Groq STT HTTP %d: %s", resp.StatusCode, string(respBody))
	}

	// We requested response_format=text above, so the body is the raw
	// transcript, not JSON. Reading the body as plain text matches the
	// Content-Type Groq returns for text-format requests and avoids
	// "invalid character ... looking for beginning of value" JSON parse
	// errors that fire on the first non-{ character of a real transcript.
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return TranscriptionProviderResult{}, fmt.Errorf("Groq STT read response: %w", err)
	}
	return TranscriptionProviderResult{
		Transcript: strings.TrimSpace(string(bodyBytes)),
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

	// Build multipart form request
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	if err := writer.WriteField("model", model); err != nil {
		return TranscriptionProviderResult{}, fmt.Errorf("Mistral STT field: %w", err)
	}

	f, err := os.Open(req.AudioPath)
	if err != nil {
		return TranscriptionProviderResult{}, fmt.Errorf("Mistral STT open audio: %w", err)
	}
	defer f.Close()

	part, err := writer.CreateFormFile("file", filepath.Base(req.AudioPath))
	if err != nil {
		return TranscriptionProviderResult{}, fmt.Errorf("Mistral STT create form file: %w", err)
	}
	if _, err := io.Copy(part, f); err != nil {
		return TranscriptionProviderResult{}, fmt.Errorf("Mistral STT copy audio: %w", err)
	}
	if err := writer.Close(); err != nil {
		return TranscriptionProviderResult{}, fmt.Errorf("Mistral STT close writer: %w", err)
	}

	baseURL := strings.TrimSuffix(p.config.BaseURL, "/")
	url := baseURL + "/v1/audio/transcriptions"

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, body)
	if err != nil {
		return TranscriptionProviderResult{}, fmt.Errorf("Mistral STT request: %w", err)
	}

	httpReq.Header.Set("Authorization", "Bearer "+apiKey)
	httpReq.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return TranscriptionProviderResult{}, fmt.Errorf("Mistral STT HTTP: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return TranscriptionProviderResult{}, fmt.Errorf("Mistral STT HTTP %d: %s", resp.StatusCode, string(respBody))
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

	// Build multipart form request
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// xAI supports optional language hint
	if req.Language != "" {
		if err := writer.WriteField("language", req.Language); err != nil {
			return TranscriptionProviderResult{}, fmt.Errorf("xAI STT language field: %w", err)
		}
	}

	// xAI supports format and diarize options (default true/false respectively)
	if err := writer.WriteField("format", "true"); err != nil {
		return TranscriptionProviderResult{}, fmt.Errorf("xAI STT format field: %w", err)
	}

	f, err := os.Open(req.AudioPath)
	if err != nil {
		return TranscriptionProviderResult{}, fmt.Errorf("xAI STT open audio: %w", err)
	}
	defer f.Close()

	part, err := writer.CreateFormFile("file", filepath.Base(req.AudioPath))
	if err != nil {
		return TranscriptionProviderResult{}, fmt.Errorf("xAI STT create form file: %w", err)
	}
	if _, err := io.Copy(part, f); err != nil {
		return TranscriptionProviderResult{}, fmt.Errorf("xAI STT copy audio: %w", err)
	}
	if err := writer.Close(); err != nil {
		return TranscriptionProviderResult{}, fmt.Errorf("xAI STT close writer: %w", err)
	}

	baseURL := strings.TrimSuffix(p.config.BaseURL, "/")
	url := baseURL + "/v1/stt"

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, body)
	if err != nil {
		return TranscriptionProviderResult{}, fmt.Errorf("xAI STT request: %w", err)
	}

	httpReq.Header.Set("Authorization", "Bearer "+apiKey)
	httpReq.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return TranscriptionProviderResult{}, fmt.Errorf("xAI STT HTTP: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return TranscriptionProviderResult{}, fmt.Errorf("xAI STT HTTP %d: %s", resp.StatusCode, string(respBody))
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
