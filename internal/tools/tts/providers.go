//go:build !slim

package tts

import (
	"bufio"
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// Default TTS configuration values
const (
	DefaultEdgeTTSVoice       = "en-US-AriaNeural"
	DefaultEdgeTTSRegion      = "eastus"
	DefaultOpenAIVoice        = "alloy"
	DefaultOpenAIModel        = "gpt-4o-mini-tts"
	DefaultOpenAIBaseURL      = "https://api.openai.com/v1"
	DefaultEdgeTTSBaseURL     = "https://%s.tts.speech.microsoft.com/cognitiveservices/v1"
	DefaultEdgeTTSContentType = "application/ssml+xml"
	DefaultOpenAIContentType  = "application/json"
	DefaultMiniMaxTTSModel    = "speech-01"
	DefaultMiniMaxTTSVoiceID  = "female-shaonv"
	DefaultMiniMaxTTSBaseURL  = "https://api.minimax.chat/v1/text_to_speech"

	// Provider names (normalized)
	ProviderNameEdge         = "edge"
	ProviderNameOpenAI       = "openai"
	ProviderNameElevenLabs   = "elevenlabs"
	ProviderNameMiniMax      = "minimax"
	ProviderNameXAI          = "xai"
	ProviderNameMistral      = "mistral"
	ProviderNameGemini       = "gemini"
	ProviderNameLocalGo      = "local_go"
	ProviderNameLocalFixture = "local_fixture"
	ProviderNameNeuTTS       = "neutts"
	ProviderNameKittenTTS    = "kittentts"
	ProviderNamePiper        = "piper"

	// Max text lengths per provider
	MaxTextLengthEdge       = 5000
	MaxTextLengthOpenAI     = 4096
	MaxTextLengthXAI        = 15000
	MaxTextLengthMiniMax    = 10000
	MaxTextLengthMistral    = 4000
	MaxTextLengthGemini     = 5000
	MaxTextLengthElevenLabs = 10000
	MaxTextLengthLocalGo    = 2000
	MaxTextLengthNeuTTS     = 2000
	MaxTextLengthKittenTTS  = 2000
	MaxTextLengthPiper      = 5000
)

var elevenLabsModelMaxTextLength = map[string]int{
	"eleven_v3":              5000,
	"eleven_ttv_v3":          5000,
	"eleven_multilingual_v2": 10000,
	"eleven_multilingual_v1": 10000,
	"eleven_english_sts_v2":  10000,
	"eleven_english_sts_v1":  10000,
	"eleven_flash_v2":        30000,
	"eleven_flash_v2_5":      40000,
}

var builtinTTSProviderOrder = []string{
	ProviderNameEdge,
	ProviderNameElevenLabs,
	ProviderNameOpenAI,
	ProviderNameMiniMax,
	ProviderNameXAI,
	ProviderNameMistral,
	ProviderNameGemini,
	ProviderNameLocalGo,
	ProviderNameLocalFixture,
	ProviderNameNeuTTS,
	ProviderNameKittenTTS,
	ProviderNamePiper,
}

// TTSProviderConfig holds HTTP provider-specific settings. It is passed to
// providers at construction time; zero value uses defaults.
type TTSProviderConfig struct {
	// Common settings
	Voice          string // Provider-specific voice identifier
	Speed          float64
	APIKey         string // API key for HTTP providers
	Region         string // Azure region for Edge TTS
	EnvLookup      func(string) (string, bool)
	ProviderConfig map[string]any

	// OpenAI-specific
	OpenAIBaseURL string

	// Edge TTS-specific (alias for Region)
	EdgeTTSRegion string

	// Timeout for HTTP requests (0 = use default)
	Timeout time.Duration
}

type TTSProviderCredential struct {
	Provider   string
	APIKey     string
	SourceName string
}

func BuiltinTTSProviderNames() []string {
	out := make([]string, len(builtinTTSProviderOrder))
	copy(out, builtinTTSProviderOrder)
	return out
}

func isBuiltinTTSProviderName(provider string) bool {
	provider = normalizeTTSProviderName(provider)
	for _, name := range builtinTTSProviderOrder {
		if provider == name {
			return true
		}
	}
	return false
}

func ResolveTTSProviderCredential(provider string, cfg TTSProviderConfig) TTSProviderCredential {
	provider = normalizeTTSProviderName(provider)
	if strings.TrimSpace(cfg.APIKey) != "" {
		return TTSProviderCredential{Provider: provider, APIKey: strings.TrimSpace(cfg.APIKey), SourceName: "config"}
	}
	names := ttsCredentialEnvNames(provider)
	value, source := lookupTTSProviderEnv(cfg, names...)
	return TTSProviderCredential{Provider: provider, APIKey: value, SourceName: source}
}

func ttsCredentialEnvNames(provider string) []string {
	switch normalizeTTSProviderName(provider) {
	case ProviderNameEdge:
		return []string{"GORMES_TTS_EDGE_KEY", "GORMES_AZURE_TTS_KEY"}
	case ProviderNameOpenAI:
		return []string{"GORMES_TTS_OPENAI_KEY", "OPENAI_API_KEY"}
	case ProviderNameElevenLabs:
		return []string{"ELEVENLABS_API_KEY"}
	case ProviderNameXAI:
		return []string{"XAI_API_KEY"}
	case ProviderNameMiniMax:
		return []string{"MINIMAX_API_KEY"}
	case ProviderNameMistral:
		return []string{"MISTRAL_API_KEY"}
	case ProviderNameGemini:
		return []string{"GEMINI_API_KEY", "GOOGLE_API_KEY"}
	default:
		return nil
	}
}

func lookupTTSProviderEnv(cfg TTSProviderConfig, names ...string) (string, string) {
	filtered := make([]string, 0, len(names))
	for _, name := range names {
		if strings.TrimSpace(name) != "" {
			filtered = append(filtered, strings.TrimSpace(name))
		}
	}
	if len(filtered) == 0 {
		return "", ""
	}
	if cfg.EnvLookup != nil {
		for _, name := range filtered {
			if value, ok := cfg.EnvLookup(name); ok && strings.TrimSpace(value) != "" {
				return strings.TrimSpace(value), name
			}
		}
		return "", ""
	}
	for _, name := range filtered {
		if value, ok := os.LookupEnv(name); ok && strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value), name
		}
	}
	dotenv := readGormesDotenvValues()
	for _, name := range filtered {
		if value := strings.TrimSpace(dotenv[name]); value != "" {
			return value, name
		}
	}
	return "", ""
}

func readGormesDotenvValues() map[string]string {
	path := filepath.Join(gormesHomeForTTS(), ".env")
	file, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer file.Close()

	values := map[string]string{}
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimLeft(scanner.Text(), " \t")
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "export ") || strings.HasPrefix(line, "export\t") {
			line = strings.TrimLeft(line[len("export"):], " \t")
		}
		eq := strings.IndexByte(line, '=')
		if eq <= 0 {
			continue
		}
		key := strings.TrimSpace(line[:eq])
		if key == "" {
			continue
		}
		values[key] = unquoteTTSProviderDotenvValue(line[eq+1:])
	}
	return values
}

func gormesHomeForTTS() string {
	if value := strings.TrimSpace(os.Getenv("GORMES_HOME")); value != "" {
		return value
	}
	if home, err := os.UserHomeDir(); err == nil && strings.TrimSpace(home) != "" {
		return filepath.Join(home, ".gormes")
	}
	return ".gormes"
}

func unquoteTTSProviderDotenvValue(raw string) string {
	raw = strings.TrimLeft(raw, " \t")
	if raw == "" {
		return ""
	}
	switch raw[0] {
	case '"':
		end := strings.LastIndex(raw[1:], `"`)
		if end < 0 {
			return raw[1:]
		}
		value := raw[1 : 1+end]
		replacer := strings.NewReplacer(`\n`, "\n", `\t`, "\t", `\r`, "\r", `\"`, `"`, `\\`, `\`)
		return replacer.Replace(value)
	case '\'':
		end := strings.IndexByte(raw[1:], '\'')
		if end < 0 {
			return raw[1:]
		}
		return raw[1 : 1+end]
	default:
		return strings.TrimRight(raw, " \t\r")
	}
}

// httpTimeout returns the effective request timeout.
// edgeTTSBaseURL returns the base URL for Edge TTS, allowing override for testing.
func (c TTSProviderConfig) edgeTTSBaseURL() string {
	if baseURL, _ := lookupTTSProviderEnv(c, "GORMES_TTS_EDGE_BASE_URL"); baseURL != "" {
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
		cfg.APIKey = ResolveTTSProviderCredential(ProviderNameEdge, cfg).APIKey
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

	voice := firstNonEmptyTTS(req.Voice, p.config.Voice, DefaultEdgeTTSVoice)
	speed := req.Speed
	if speed <= 0 {
		speed = p.config.Speed
	}

	// Build SSML document
	ssml := buildEdgeTTSSSML(req.Text, voice, speed)

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
	escaped := escapeSSML(text)
	rate := EdgeTTSRateString(speed)
	if rate == "" {
		return fmt.Sprintf(
			`<speak version='1.0' xmlns='http://www.w3.org/2001/10/synthesis' xml:lang='en-US'><voice name='%s'>%s</voice></speak>`,
			voice, escaped,
		)
	}
	return fmt.Sprintf(
		`<speak version='1.0' xmlns='http://www.w3.org/2001/10/synthesis' xml:lang='en-US'><voice name='%s'><prosody rate='%s'>%s</prosody></voice></speak>`,
		voice, rate, escaped,
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
		cfg.APIKey = ResolveTTSProviderCredential(ProviderNameOpenAI, cfg).APIKey
	}
	if cfg.OpenAIBaseURL == "" {
		cfg.OpenAIBaseURL, _ = lookupTTSProviderEnv(cfg, "GORMES_TTS_OPENAI_BASE_URL")
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
	voice := firstNonEmptyTTS(req.Voice, p.config.Voice, DefaultOpenAIVoice)

	// Determine output format from file extension
	format := "mp3"
	if strings.HasSuffix(strings.ToLower(req.OutputPath), ".ogg") ||
		strings.HasSuffix(strings.ToLower(req.OutputPath), ".opus") {
		format = "opus"
	}

	payload := map[string]any{
		"model":           model,
		"input":           req.Text,
		"voice":           voice,
		"response_format": format,
	}
	speed := req.Speed
	if speed <= 0 {
		speed = p.config.Speed
	}
	speed = OpenAITTSSpeed(speed)
	if speed != 1.0 {
		payload["speed"] = speed
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return TTSProviderResult{}, fmt.Errorf("OpenAI TTS payload: %w", err)
	}

	url := openAITTSSpeechURL(p.config.OpenAIBaseURL)

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

func openAITTSSpeechURL(baseURL string) string {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if strings.HasSuffix(baseURL, "/v1") {
		return baseURL + "/audio/speech"
	}
	return baseURL + "/v1/audio/speech"
}

type miniMaxTTSSettings struct {
	Model   string
	VoiceID string
	BaseURL string
}

func (c TTSProviderConfig) miniMaxTTSSettings() miniMaxTTSSettings {
	section := mapFromAny(lookupCaseInsensitiveAny(c.ProviderConfig, ProviderNameMiniMax))
	envBaseURL, _ := lookupTTSProviderEnv(c, "GORMES_TTS_MINIMAX_BASE_URL", "MINIMAX_TTS_BASE_URL")
	return miniMaxTTSSettings{
		Model: firstNonEmptyTTS(
			stringFromAny(lookupCaseInsensitiveAny(section, "model")),
			DefaultMiniMaxTTSModel,
		),
		VoiceID: firstNonEmptyTTS(
			stringFromAny(lookupCaseInsensitiveAny(section, "voice_id")),
			stringFromAny(lookupCaseInsensitiveAny(section, "voice")),
			c.Voice,
			DefaultMiniMaxTTSVoiceID,
		),
		BaseURL: firstNonEmptyTTS(
			stringFromAny(lookupCaseInsensitiveAny(section, "base_url")),
			envBaseURL,
			DefaultMiniMaxTTSBaseURL,
		),
	}
}

// TTSMiniMaxProvider is an HTTP-based MiniMax TTS provider.
type TTSMiniMaxProvider struct {
	config TTSProviderConfig
	client *http.Client
}

func NewTTSMiniMaxProvider(config TTSProviderConfig) *TTSMiniMaxProvider {
	cfg := config
	if cfg.Timeout == 0 {
		cfg.Timeout = 60 * time.Second
	}
	if cfg.APIKey == "" {
		cfg.APIKey = ResolveTTSProviderCredential(ProviderNameMiniMax, cfg).APIKey
	}
	return &TTSMiniMaxProvider{
		config: cfg,
		client: &http.Client{Timeout: cfg.Timeout},
	}
}

func (p *TTSMiniMaxProvider) Available(ctx context.Context) bool {
	return strings.TrimSpace(p.config.APIKey) != ""
}

func (*TTSMiniMaxProvider) PreferredOutputFormat() string { return "mp3" }

func (p *TTSMiniMaxProvider) Synthesize(ctx context.Context, req TTSProviderRequest) (TTSProviderResult, error) {
	apiKey := strings.TrimSpace(p.config.APIKey)
	if apiKey == "" {
		return TTSProviderResult{}, errors.New("MiniMax TTS API key not configured")
	}

	settings := p.config.miniMaxTTSSettings()
	payload := map[string]any{
		"model":    settings.Model,
		"text":     req.Text,
		"voice_id": firstNonEmptyTTS(req.Voice, settings.VoiceID),
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return TTSProviderResult{}, fmt.Errorf("MiniMax TTS payload: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, settings.BaseURL, bytes.NewReader(body))
	if err != nil {
		return TTSProviderResult{}, fmt.Errorf("MiniMax TTS request: %w", err)
	}
	httpReq.Header.Set("Authorization", "Bearer "+apiKey)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return TTSProviderResult{}, fmt.Errorf("MiniMax TTS HTTP: %w", err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return TTSProviderResult{}, fmt.Errorf("MiniMax TTS read body: %w", err)
	}

	contentType := strings.ToLower(resp.Header.Get("Content-Type"))
	if strings.Contains(contentType, "audio/") {
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return TTSProviderResult{}, fmt.Errorf("MiniMax TTS HTTP %d: %s", resp.StatusCode, string(data))
		}
		if err := os.WriteFile(req.OutputPath, data, 0o600); err != nil {
			return TTSProviderResult{}, fmt.Errorf("MiniMax TTS write file: %w", err)
		}
		return TTSProviderResult{FilePath: req.OutputPath, Provider: ProviderNameMiniMax}, nil
	}

	audio, parsedLegacy, err := parseMiniMaxTTSLegacyAudio(data)
	if parsedLegacy {
		if err != nil {
			return TTSProviderResult{}, err
		}
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return TTSProviderResult{}, fmt.Errorf("MiniMax TTS HTTP %d", resp.StatusCode)
		}
		if err := os.WriteFile(req.OutputPath, audio, 0o600); err != nil {
			return TTSProviderResult{}, fmt.Errorf("MiniMax TTS write file: %w", err)
		}
		return TTSProviderResult{FilePath: req.OutputPath, Provider: ProviderNameMiniMax}, nil
	}
	if err != nil {
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return TTSProviderResult{}, fmt.Errorf("MiniMax TTS HTTP %d: %s", resp.StatusCode, string(data))
		}
		if strings.TrimSpace(contentType) == "" {
			contentType = "unknown"
		}
		return TTSProviderResult{}, fmt.Errorf("MiniMax TTS returned unexpected Content-Type %q (%d bytes)", contentType, len(data))
	}
	return TTSProviderResult{}, fmt.Errorf("MiniMax TTS returned unexpected empty response")
}

func parseMiniMaxTTSLegacyAudio(data []byte) ([]byte, bool, error) {
	var result struct {
		BaseResp struct {
			StatusCode int    `json:"status_code"`
			StatusMsg  string `json:"status_msg"`
		} `json:"base_resp"`
		Data struct {
			Audio string `json:"audio"`
		} `json:"data"`
	}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, false, err
	}
	if result.BaseResp.StatusCode != 0 {
		statusMsg := strings.TrimSpace(result.BaseResp.StatusMsg)
		if statusMsg == "" {
			statusMsg = "unknown error"
		}
		return nil, true, fmt.Errorf("MiniMax TTS API error (code %d): %s", result.BaseResp.StatusCode, statusMsg)
	}
	hexAudio := strings.TrimSpace(result.Data.Audio)
	if hexAudio == "" {
		return nil, true, errors.New("MiniMax TTS returned empty audio data")
	}
	audio, err := hex.DecodeString(hexAudio)
	if err != nil {
		return nil, true, fmt.Errorf("MiniMax TTS decode audio: %w", err)
	}
	return audio, true, nil
}

// RegisterTTSProviders registers the built-in HTTP TTS providers into a provider
// map. It skips nil providers (when API keys are absent).
func RegisterTTSProviders(into map[string]TTSProvider, cfg TTSProviderConfig) {
	localGo := NewGoNativeTTSProvider(GoNativeTTSProviderConfig{})
	into[ProviderNameLocalGo] = localGo
	into[ProviderNameLocalFixture] = localGo

	edge := NewTTSEdgeProvider(cfg)
	if edge.Available(context.Background()) {
		into[ProviderNameEdge] = edge
	}

	openai := NewTTSOpenAIProvider(cfg)
	if openai.Available(context.Background()) {
		into[ProviderNameOpenAI] = openai
	}
	minimax := NewTTSMiniMaxProvider(cfg)
	if minimax.Available(context.Background()) {
		into[ProviderNameMiniMax] = minimax
	}
}

// ValidateTTSProviderConfig checks that a provider name is valid and that required
// fields are present. Returns an error describing the problem, or nil if valid.
func ValidateTTSProviderConfig(provider string, cfg TTSProviderConfig) error {
	provider = normalizeTTSProviderName(provider)
	switch provider {
	case ProviderNameEdge:
		if cfg.APIKey == "" {
			if ResolveTTSProviderCredential(provider, cfg).APIKey == "" {
				return errors.New("Edge TTS requires GORMES_TTS_EDGE_KEY or GORMES_AZURE_TTS_KEY")
			}
		}
		return nil

	case ProviderNameOpenAI:
		if cfg.APIKey == "" {
			if ResolveTTSProviderCredential(provider, cfg).APIKey == "" {
				return errors.New("OpenAI TTS requires GORMES_TTS_OPENAI_KEY or OPENAI_API_KEY")
			}
		}
		return nil

	case ProviderNameElevenLabs, ProviderNameMiniMax, ProviderNameXAI, ProviderNameMistral, ProviderNameGemini:
		if cfg.APIKey == "" {
			if ResolveTTSProviderCredential(provider, cfg).APIKey == "" {
				return fmt.Errorf("%s TTS API key not configured", provider)
			}
		}
		return nil

	case ProviderNameLocalGo, ProviderNameLocalFixture, ProviderNameNeuTTS, ProviderNameKittenTTS, ProviderNamePiper, "local":
		return nil

	case "auto", "":
		return nil // Auto-selection is valid

	default:
		return fmt.Errorf("unknown TTS provider %q", provider)
	}
}

// TTSProviderMaxTextLength returns the maximum input length for a provider.
func TTSProviderMaxTextLength(provider string) int {
	return TTSProviderMaxTextLengthForConfig(provider, nil)
}

func TTSProviderMaxTextLengthForConfig(provider string, ttsConfig map[string]any) int {
	key := normalizeTTSProviderName(provider)
	if key == "" {
		return defaultTTSMaxTextLength
	}
	if section := mapFromAny(lookupCaseInsensitiveAny(ttsConfig, key)); section != nil {
		if override := positiveIntFromAny(section["max_text_length"]); override > 0 {
			return override
		}
	}
	if !isBuiltinTTSProviderName(key) {
		if named := namedTTSProviderConfig(ttsConfig, key); isTTSCommandProviderConfig(named) {
			if override := positiveIntFromAny(named["max_text_length"]); override > 0 {
				return override
			}
			return defaultCommandTTSMaxTextLength
		}
	}
	if key == ProviderNameElevenLabs {
		section := mapFromAny(lookupCaseInsensitiveAny(ttsConfig, key))
		modelID := "eleven_multilingual_v2"
		if section != nil {
			if configured := stringFromAny(section["model_id"]); configured != "" {
				modelID = configured
			}
		}
		if maxLen := elevenLabsModelMaxTextLength[strings.TrimSpace(modelID)]; maxLen > 0 {
			return maxLen
		}
		return MaxTextLengthElevenLabs
	}
	switch normalizeTTSProviderName(provider) {
	case ProviderNameEdge:
		return MaxTextLengthEdge
	case ProviderNameOpenAI:
		return MaxTextLengthOpenAI
	case ProviderNameXAI:
		return MaxTextLengthXAI
	case ProviderNameMiniMax:
		return MaxTextLengthMiniMax
	case ProviderNameMistral:
		return MaxTextLengthMistral
	case ProviderNameGemini:
		return MaxTextLengthGemini
	case ProviderNameLocalGo, ProviderNameLocalFixture:
		return MaxTextLengthLocalGo
	case ProviderNameNeuTTS:
		return MaxTextLengthNeuTTS
	case ProviderNameKittenTTS:
		return MaxTextLengthKittenTTS
	case ProviderNamePiper:
		return MaxTextLengthPiper
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

func TTSSpeedForProvider(provider string, ttsConfig map[string]any) float64 {
	provider = normalizeTTSProviderName(provider)
	speed := floatFromAny(lookupCaseInsensitiveAny(ttsConfig, "speed"))
	if section := mapFromAny(lookupCaseInsensitiveAny(ttsConfig, provider)); section != nil {
		if providerSpeed := floatFromAny(lookupCaseInsensitiveAny(section, "speed")); providerSpeed > 0 {
			speed = providerSpeed
		}
	}
	if speed <= 0 {
		return 1.0
	}
	return speed
}

func EdgeTTSRateString(speed float64) string {
	if speed <= 0 || speed == 1.0 {
		return ""
	}
	pct := int((speed - 1.0) * 100)
	if pct == 0 {
		return ""
	}
	if pct > 0 {
		return fmt.Sprintf("+%d%%", pct)
	}
	return fmt.Sprintf("%d%%", pct)
}

func OpenAITTSSpeed(speed float64) float64 {
	if speed <= 0 {
		return 1.0
	}
	if speed < 0.25 {
		return 0.25
	}
	if speed > 4.0 {
		return 4.0
	}
	return speed
}

type TTSProviderStatus struct {
	Provider  string
	Available bool
	Evidence  TTSEvidence
	Error     string
}

type LazyLocalTTSProvider struct {
	provider string
	probe    func(context.Context) error
}

func NewLazyLocalTTSProvider(provider string, probe func(context.Context) error) *LazyLocalTTSProvider {
	return &LazyLocalTTSProvider{
		provider: normalizeTTSProviderName(provider),
		probe:    probe,
	}
}

func (p *LazyLocalTTSProvider) Available(ctx context.Context) bool {
	return p.DependencyStatus(ctx).Available
}

func (p *LazyLocalTTSProvider) DependencyStatus(ctx context.Context) TTSProviderStatus {
	if p == nil {
		return TTSProviderStatus{Evidence: TTSEvidenceProviderUnavailable, Error: "no local TTS provider configured"}
	}
	provider := firstNonEmptyTTS(p.provider, "local")
	if p.probe == nil {
		return TTSProviderStatus{Provider: provider, Available: true, Evidence: TTSEvidenceOK}
	}
	if err := p.probe(ctx); err != nil {
		return TTSProviderStatus{
			Provider:  provider,
			Available: false,
			Evidence:  TTSEvidenceProviderUnavailable,
			Error:     redactTTSText(err.Error()),
		}
	}
	return TTSProviderStatus{Provider: provider, Available: true, Evidence: TTSEvidenceOK}
}

func (p *LazyLocalTTSProvider) Synthesize(ctx context.Context, req TTSProviderRequest) (TTSProviderResult, error) {
	status := p.DependencyStatus(ctx)
	if !status.Available {
		return TTSProviderResult{}, errors.New(status.Error)
	}
	return TTSProviderResult{}, fmt.Errorf("%s TTS synthesis provider is not wired in this build", firstNonEmptyTTS(status.Provider, req.Provider))
}
