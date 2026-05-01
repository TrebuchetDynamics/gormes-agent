//go:build !slim

package tools

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// ---------------------------------------------------------------------------
// Provider selection logic
// ---------------------------------------------------------------------------

func TestProviderSelection(t *testing.T) {
	ctx := context.Background()

	t.Run("selects edge when requested and available", func(t *testing.T) {
		providers := map[string]TTSProvider{
			"edge":    &fakeTTSProvider{available: true},
			"openai": &fakeTTSProvider{available: true},
		}
		runner := NewTTSRunner(TTSConfig{Provider: "edge"}, providers)
		result := runner.Synthesize(ctx, TTSRequest{Text: "hello"})
		if !result.Success {
			t.Fatalf("expected success, got %+v", result)
		}
	})

	t.Run("falls back when requested provider unavailable", func(t *testing.T) {
		providers := map[string]TTSProvider{
			"edge":    &fakeTTSProvider{available: false},
			"openai": &fakeTTSProvider{available: true},
		}
		runner := NewTTSRunner(TTSConfig{Provider: "edge"}, providers)
		result := runner.Synthesize(ctx, TTSRequest{Text: "hello"})
		if result.Success {
			t.Fatalf("expected failure for unavailable provider, got %+v", result)
		}
		if result.Evidence != TTSEvidenceProviderUnavailable {
			t.Fatalf("expected provider unavailable evidence, got %+v", result)
		}
	})

	t.Run("selects first available when provider is auto", func(t *testing.T) {
		providers := map[string]TTSProvider{
			"edge":    &fakeTTSProvider{available: false},
			"openai": &fakeTTSProvider{available: true},
		}
		runner := NewTTSRunner(TTSConfig{Provider: "auto"}, providers)
		result := runner.Synthesize(ctx, TTSRequest{Text: "hello"})
		if !result.Success {
			t.Fatalf("expected success with fallback, got %+v", result)
		}
	})

	t.Run("returns unavailable when no providers available", func(t *testing.T) {
		providers := map[string]TTSProvider{
			"edge":    &fakeTTSProvider{available: false},
			"openai": &fakeTTSProvider{available: false},
		}
		runner := NewTTSRunner(TTSConfig{}, providers)
		result := runner.Synthesize(ctx, TTSRequest{Text: "hello"})
		if result.Success {
			t.Fatalf("expected failure when no providers available, got %+v", result)
		}
		if result.Evidence != TTSEvidenceProviderUnavailable {
			t.Fatalf("expected provider unavailable evidence, got %+v", result)
		}
	})

	t.Run("case insensitive provider names", func(t *testing.T) {
		providers := map[string]TTSProvider{
			"EDGE":    &fakeTTSProvider{available: true},
			"OpenAI": &fakeTTSProvider{available: true},
		}
		runner := NewTTSRunner(TTSConfig{Provider: "EDGE"}, providers)
		result := runner.Synthesize(ctx, TTSRequest{Text: "hello"})
		if !result.Success {
			t.Fatalf("expected success with case-insensitive match, got %+v", result)
		}
	})
}

// ---------------------------------------------------------------------------
// Config validation
// ---------------------------------------------------------------------------

func TestValidateTTSProviderConfig(t *testing.T) {
	// Save and restore env vars
	origEdgeKey := os.Getenv("GORMES_TTS_EDGE_KEY")
	origAzureKey := os.Getenv("GORMES_AZURE_TTS_KEY")
	origOpenAIKey := os.Getenv("GORMES_TTS_OPENAI_KEY")
	origOpenAIEnv := os.Getenv("OPENAI_API_KEY")
	defer func() {
		os.Setenv("GORMES_TTS_EDGE_KEY", origEdgeKey)
		os.Setenv("GORMES_AZURE_TTS_KEY", origAzureKey)
		os.Setenv("GORMES_TTS_OPENAI_KEY", origOpenAIKey)
		os.Setenv("OPENAI_API_KEY", origOpenAIEnv)
	}()

	t.Run("edge requires API key", func(t *testing.T) {
		os.Unsetenv("GORMES_TTS_EDGE_KEY")
		os.Unsetenv("GORMES_AZURE_TTS_KEY")
		err := ValidateTTSProviderConfig("edge", TTSProviderConfig{})
		if err == nil {
			t.Fatal("expected error when no Edge API key configured")
		}
		if !strings.Contains(err.Error(), "Edge TTS") {
			t.Fatalf("error should mention Edge TTS: %v", err)
		}
	})

	t.Run("edge accepts GORMES_TTS_EDGE_KEY", func(t *testing.T) {
		os.Setenv("GORMES_TTS_EDGE_KEY", "test-key")
		os.Unsetenv("GORMES_AZURE_TTS_KEY")
		err := ValidateTTSProviderConfig("edge", TTSProviderConfig{})
		if err != nil {
			t.Fatalf("expected no error with GORMES_TTS_EDGE_KEY set: %v", err)
		}
	})

	t.Run("edge accepts GORMES_AZURE_TTS_KEY", func(t *testing.T) {
		os.Setenv("GORMES_AZURE_TTS_KEY", "azure-key")
		os.Unsetenv("GORMES_TTS_EDGE_KEY")
		err := ValidateTTSProviderConfig("edge", TTSProviderConfig{})
		if err != nil {
			t.Fatalf("expected no error with GORMES_AZURE_TTS_KEY set: %v", err)
		}
	})

	t.Run("edge accepts config API key", func(t *testing.T) {
		os.Unsetenv("GORMES_TTS_EDGE_KEY")
		os.Unsetenv("GORMES_AZURE_TTS_KEY")
		err := ValidateTTSProviderConfig("edge", TTSProviderConfig{APIKey: "cfg-key"})
		if err != nil {
			t.Fatalf("expected no error with config API key: %v", err)
		}
	})

	t.Run("openai requires API key", func(t *testing.T) {
		os.Unsetenv("GORMES_TTS_OPENAI_KEY")
		os.Unsetenv("OPENAI_API_KEY")
		err := ValidateTTSProviderConfig("openai", TTSProviderConfig{})
		if err == nil {
			t.Fatal("expected error when no OpenAI API key configured")
		}
		if !strings.Contains(err.Error(), "OpenAI") {
			t.Fatalf("error should mention OpenAI: %v", err)
		}
	})

	t.Run("openai accepts GORMES_TTS_OPENAI_KEY", func(t *testing.T) {
		os.Setenv("GORMES_TTS_OPENAI_KEY", "test-key")
		os.Unsetenv("OPENAI_API_KEY")
		err := ValidateTTSProviderConfig("openai", TTSProviderConfig{})
		if err != nil {
			t.Fatalf("expected no error with GORMES_TTS_OPENAI_KEY set: %v", err)
		}
	})

	t.Run("openai accepts OPENAI_API_KEY", func(t *testing.T) {
		os.Setenv("OPENAI_API_KEY", "openai-key")
		os.Unsetenv("GORMES_TTS_OPENAI_KEY")
		err := ValidateTTSProviderConfig("openai", TTSProviderConfig{})
		if err != nil {
			t.Fatalf("expected no error with OPENAI_API_KEY set: %v", err)
		}
	})

	t.Run("auto is always valid", func(t *testing.T) {
		err := ValidateTTSProviderConfig("auto", TTSProviderConfig{})
		if err != nil {
			t.Fatalf("auto selection should be valid: %v", err)
		}
	})

	t.Run("empty provider is valid", func(t *testing.T) {
		err := ValidateTTSProviderConfig("", TTSProviderConfig{})
		if err != nil {
			t.Fatalf("empty provider should be valid: %v", err)
		}
	})

	t.Run("unknown provider", func(t *testing.T) {
		err := ValidateTTSProviderConfig("unknown", TTSProviderConfig{})
		if err == nil {
			t.Fatal("expected error for unknown provider")
		}
		if !strings.Contains(err.Error(), "unknown") {
			t.Fatalf("error should mention unknown provider: %v", err)
		}
	})
}

// ---------------------------------------------------------------------------
// TTSEdgeProvider
// ---------------------------------------------------------------------------

func TestTTSEdgeProviderAvailable(t *testing.T) {
	orig := os.Getenv("GORMES_TTS_EDGE_KEY")
	defer os.Setenv("GORMES_TTS_EDGE_KEY", orig)

	t.Run("available with API key", func(t *testing.T) {
		os.Setenv("GORMES_TTS_EDGE_KEY", "test-key")
		p := NewTTSEdgeProvider(TTSProviderConfig{})
		if !p.Available(context.Background()) {
			t.Fatal("expected available with API key")
		}
	})

	t.Run("not available without API key", func(t *testing.T) {
		os.Unsetenv("GORMES_TTS_EDGE_KEY")
		p := NewTTSEdgeProvider(TTSProviderConfig{})
		if p.Available(context.Background()) {
			t.Fatal("expected not available without API key")
		}
	})
}

func TestTTSEdgeProviderSynthesize(t *testing.T) {
	// Save and restore env var for base URL
	origBaseURL := os.Getenv("GORMES_TTS_EDGE_BASE_URL")
	defer func() {
		if origBaseURL == "" {
			os.Unsetenv("GORMES_TTS_EDGE_BASE_URL")
		} else {
			os.Setenv("GORMES_TTS_EDGE_BASE_URL", origBaseURL)
		}
	}()

	t.Run("synthesizes successfully", func(t *testing.T) {
		var receivedAuth string
		var receivedContentType string
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			receivedAuth = r.Header.Get("Ocp-Apim-Subscription-Key")
			receivedContentType = r.Header.Get("Content-Type")
			w.Header().Set("Content-Type", "audio/mpeg")
			w.Write([]byte("fake audio data"))
		}))
		defer server.Close()

		// Set the base URL to our mock server
		os.Setenv("GORMES_TTS_EDGE_BASE_URL", server.URL)

		output := filepath.Join(t.TempDir(), "test.mp3")
		provider := NewTTSEdgeProvider(TTSProviderConfig{
			APIKey:  "test-edge-key",
			Voice:   "en-US-JennyNeural",
			Region:  "eastus",
			Timeout: 10 * time.Second,
		})

		result, err := provider.Synthesize(context.Background(), TTSProviderRequest{
			Text:       "Hello world",
			OutputPath: output,
			Provider:   "edge",
		})

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.FilePath == "" {
			t.Fatalf("expected file path in result: %+v", result)
		}
		if receivedAuth != "test-edge-key" {
			t.Fatalf("expected auth header with API key")
		}
		if receivedContentType != "application/ssml+xml" {
			t.Fatalf("expected SSML content type, got: %s", receivedContentType)
		}
	})

	t.Run("handles HTTP error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte("Invalid API key"))
		}))
		defer server.Close()

		os.Setenv("GORMES_TTS_EDGE_BASE_URL", server.URL)

		output := filepath.Join(t.TempDir(), "test.mp3")
		provider := NewTTSEdgeProvider(TTSProviderConfig{
			APIKey:  "bad-key",
			Region:  "eastus",
			Timeout: 10 * time.Second,
		})

		_, err := provider.Synthesize(context.Background(), TTSProviderRequest{
			Text:       "Hello",
			OutputPath: output,
			Provider:   "edge",
		})

		if err == nil {
			t.Fatal("expected error for HTTP 401")
		}
		if !strings.Contains(err.Error(), "401") {
			t.Fatalf("error should contain status code: %v", err)
		}
	})

	t.Run("returns error without API key", func(t *testing.T) {
		os.Unsetenv("GORMES_TTS_EDGE_BASE_URL")
		provider := NewTTSEdgeProvider(TTSProviderConfig{})

		_, err := provider.Synthesize(context.Background(), TTSProviderRequest{
			Text:       "Hello",
			OutputPath: filepath.Join(t.TempDir(), "test.mp3"),
			Provider:   "edge",
		})

		if err == nil {
			t.Fatal("expected error without API key")
		}
	})
}

func TestBuildEdgeTTSSSML(t *testing.T) {
	t.Run("escapes special characters", func(t *testing.T) {
		ssml := buildEdgeTTSSSML("Hello <world> & \"test\"", "en-US-AriaNeural", 1.0)
		if strings.Contains(ssml, "<world>") {
			t.Fatal("SSML should escape < and >")
		}
		if !strings.Contains(ssml, "&amp;") {
			t.Fatal("SSML should escape & to &amp;")
		}
	})

	t.Run("sets rate correctly", func(t *testing.T) {
		ssmlFast := buildEdgeTTSSSML("Hello", "en-US-AriaNeural", 1.5)
		if !strings.Contains(ssmlFast, `rate='+50%'`) {
			t.Fatalf("expected +50%% rate for 1.5x speed, got: %s", ssmlFast)
		}

		ssmlSlow := buildEdgeTTSSSML("Hello", "en-US-AriaNeural", 0.5)
		if !strings.Contains(ssmlSlow, `rate='-50%'`) {
			t.Fatalf("expected -50%% rate for 0.5x speed, got: %s", ssmlSlow)
		}
	})
}

// ---------------------------------------------------------------------------
// TTSOpenAIProvider
// ---------------------------------------------------------------------------

func TestTTSOpenAIProviderAvailable(t *testing.T) {
	orig := os.Getenv("GORMES_TTS_OPENAI_KEY")
	defer os.Setenv("GORMES_TTS_OPENAI_KEY", orig)

	t.Run("available with API key", func(t *testing.T) {
		os.Setenv("GORMES_TTS_OPENAI_KEY", "sk-test-key")
		p := NewTTSOpenAIProvider(TTSProviderConfig{})
		if !p.Available(context.Background()) {
			t.Fatal("expected available with API key")
		}
	})

	t.Run("not available without API key", func(t *testing.T) {
		os.Unsetenv("GORMES_TTS_OPENAI_KEY")
		p := NewTTSOpenAIProvider(TTSProviderConfig{})
		if p.Available(context.Background()) {
			t.Fatal("expected not available without API key")
		}
	})
}

func TestTTSOpenAIProviderSynthesize(t *testing.T) {
	t.Run("synthesizes successfully with MP3", func(t *testing.T) {
		var reqBody map[string]any
		var authHeader string
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader = r.Header.Get("Authorization")
			json.NewDecoder(r.Body).Decode(&reqBody)
			w.Header().Set("Content-Type", "audio/mpeg")
			w.Write([]byte("fake audio data"))
		}))
		defer server.Close()

		output := filepath.Join(t.TempDir(), "test.mp3")
		provider := NewTTSOpenAIProvider(TTSProviderConfig{
			APIKey:      "sk-test-key",
			Voice:       "alloy",
			OpenAIBaseURL: server.URL,
			Timeout:     10 * time.Second,
		})

		result, err := provider.Synthesize(context.Background(), TTSProviderRequest{
			Text:       "Hello world",
			OutputPath: output,
			Provider:   "openai",
		})

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.FilePath == "" {
			t.Fatalf("expected file path in result: %+v", result)
		}
		if !strings.HasPrefix(authHeader, "Bearer ") {
			t.Fatalf("expected Bearer auth header, got: %s", authHeader)
		}
		if reqBody["model"] != "gpt-4o-mini-tts" {
			t.Fatalf("expected gpt-4o-mini-tts model, got: %v", reqBody["model"])
		}
		if reqBody["voice"] != "alloy" {
			t.Fatalf("expected alloy voice, got: %v", reqBody["voice"])
		}
		if reqBody["input"] != "Hello world" {
			t.Fatalf("expected input text, got: %v", reqBody["input"])
		}
	})

	t.Run("uses opus format for OGG output", func(t *testing.T) {
		var reqBody map[string]any
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			json.NewDecoder(r.Body).Decode(&reqBody)
			w.Header().Set("Content-Type", "audio/ogg")
			w.Write([]byte("fake opus data"))
		}))
		defer server.Close()

		output := filepath.Join(t.TempDir(), "test.ogg")
		provider := NewTTSOpenAIProvider(TTSProviderConfig{
			APIKey:      "sk-test-key",
			OpenAIBaseURL: server.URL,
			Timeout:     10 * time.Second,
		})

		result, err := provider.Synthesize(context.Background(), TTSProviderRequest{
			Text:       "Hello",
			OutputPath: output,
			Provider:   "openai",
		})

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if reqBody["response_format"] != "opus" {
			t.Fatalf("expected opus format for .ogg output, got: %v", reqBody["response_format"])
		}
		if !result.VoiceCompatible {
			t.Fatal("expected voice compatible for opus format")
		}
	})

	t.Run("clamps speed to valid range", func(t *testing.T) {
		var reqBody map[string]any
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			json.NewDecoder(r.Body).Decode(&reqBody)
			w.Write([]byte("fake audio"))
		}))
		defer server.Close()

		provider := NewTTSOpenAIProvider(TTSProviderConfig{
			APIKey:      "sk-test-key",
			Speed:       10.0, // exceeds max of 4.0
			OpenAIBaseURL: server.URL,
			Timeout:     10 * time.Second,
		})

		_, _ = provider.Synthesize(context.Background(), TTSProviderRequest{
			Text:       "Hello",
			OutputPath: filepath.Join(t.TempDir(), "test.mp3"),
			Provider:   "openai",
		})

		speed := reqBody["speed"].(float64)
		if speed != 4.0 {
			t.Fatalf("expected speed clamped to 4.0, got: %v", speed)
		}
	})

	t.Run("handles HTTP error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`{"error":{"message":"Invalid API key"}}`))
		}))
		defer server.Close()

		provider := NewTTSOpenAIProvider(TTSProviderConfig{
			APIKey:      "bad-key",
			OpenAIBaseURL: server.URL,
			Timeout:     10 * time.Second,
		})

		_, err := provider.Synthesize(context.Background(), TTSProviderRequest{
			Text:       "Hello",
			OutputPath: filepath.Join(t.TempDir(), "test.mp3"),
			Provider:   "openai",
		})

		if err == nil {
			t.Fatal("expected error for HTTP 401")
		}
		if !strings.Contains(err.Error(), "401") {
			t.Fatalf("error should contain status code: %v", err)
		}
	})

	t.Run("returns error without API key", func(t *testing.T) {
		provider := NewTTSOpenAIProvider(TTSProviderConfig{})

		_, err := provider.Synthesize(context.Background(), TTSProviderRequest{
			Text:       "Hello",
			OutputPath: filepath.Join(t.TempDir(), "test.mp3"),
			Provider:   "openai",
		})

		if err == nil {
			t.Fatal("expected error without API key")
		}
	})
}

// ---------------------------------------------------------------------------
// Max text length
// ---------------------------------------------------------------------------

func TestTTSProviderMaxTextLength(t *testing.T) {
	tests := []struct {
		provider string
		want     int
	}{
		{"edge", MaxTextLengthEdge},
		{"openai", MaxTextLengthOpenAI},
		{"EDGE", MaxTextLengthEdge},     // case insensitive
		{"OPENAI", MaxTextLengthOpenAI}, // case insensitive
		{"unknown", defaultTTSMaxTextLength},
		{"", defaultTTSMaxTextLength},
	}

	for _, tt := range tests {
		t.Run(tt.provider, func(t *testing.T) {
			got := TTSProviderMaxTextLength(tt.provider)
			if got != tt.want {
				t.Errorf("TTSProviderMaxTextLength(%q) = %d, want %d", tt.provider, got, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// RegisterTTSProviders
// ---------------------------------------------------------------------------

func TestRegisterTTSProviders(t *testing.T) {
	origEdge := os.Getenv("GORMES_TTS_EDGE_KEY")
	origOpenAI := os.Getenv("GORMES_TTS_OPENAI_KEY")
	defer func() {
		os.Setenv("GORMES_TTS_EDGE_KEY", origEdge)
		os.Setenv("GORMES_TTS_OPENAI_KEY", origOpenAI)
	}()

	t.Run("registers available providers", func(t *testing.T) {
		os.Setenv("GORMES_TTS_EDGE_KEY", "edge-key")
		os.Setenv("GORMES_TTS_OPENAI_KEY", "openai-key")

		into := make(map[string]TTSProvider)
		RegisterTTSProviders(into, TTSProviderConfig{})

		if _, ok := into["edge"]; !ok {
			t.Error("expected edge provider to be registered")
		}
		if _, ok := into["openai"]; !ok {
			t.Error("expected openai provider to be registered")
		}
	})

	t.Run("skips unavailable providers", func(t *testing.T) {
		os.Unsetenv("GORMES_TTS_EDGE_KEY")
		os.Unsetenv("GORMES_TTS_OPENAI_KEY")

		into := make(map[string]TTSProvider)
		RegisterTTSProviders(into, TTSProviderConfig{})

		if len(into) != 0 {
			t.Fatalf("expected no providers registered without keys, got: %v", mapKeys(into))
		}
	})

	t.Run("partial registration when only one key set", func(t *testing.T) {
		os.Setenv("GORMES_TTS_EDGE_KEY", "edge-key")
		os.Unsetenv("GORMES_TTS_OPENAI_KEY")

		into := make(map[string]TTSProvider)
		RegisterTTSProviders(into, TTSProviderConfig{})

		if _, ok := into["edge"]; !ok {
			t.Error("expected edge provider registered")
		}
		if _, ok := into["openai"]; ok {
			t.Error("openai should not be registered without key")
		}
	})
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func mapKeys(m map[string]TTSProvider) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}
