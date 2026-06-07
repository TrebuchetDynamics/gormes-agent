//go:build !slim

package transcription

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

// ---------------------------------------------------------------------------
// TranscriptionOpenAIProvider
// ---------------------------------------------------------------------------

func TestTranscriptionOpenAIProviderAvailable(t *testing.T) {
	orig := os.Getenv("GORMES_STT_OPENAI_KEY")
	defer os.Setenv("GORMES_STT_OPENAI_KEY", orig)

	t.Run("available with API key", func(t *testing.T) {
		os.Setenv("GORMES_STT_OPENAI_KEY", "sk-test-key")
		p := NewTranscriptionOpenAIProvider(TranscriptionProviderConfig{})
		if !p.Available(context.Background()) {
			t.Fatal("expected available with API key")
		}
	})

	t.Run("available with OPENAI_API_KEY fallback", func(t *testing.T) {
		os.Unsetenv("GORMES_STT_OPENAI_KEY")
		os.Setenv("OPENAI_API_KEY", "sk-fallback-key")
		p := NewTranscriptionOpenAIProvider(TranscriptionProviderConfig{})
		if !p.Available(context.Background()) {
			t.Fatal("expected available with OPENAI_API_KEY fallback")
		}
	})

	t.Run("not available without API key", func(t *testing.T) {
		os.Unsetenv("GORMES_STT_OPENAI_KEY")
		os.Unsetenv("OPENAI_API_KEY")
		p := NewTranscriptionOpenAIProvider(TranscriptionProviderConfig{})
		if p.Available(context.Background()) {
			t.Fatal("expected not available without API key")
		}
	})
}

func TestTranscriptionOpenAIProviderTranscribe(t *testing.T) {
	t.Run("transcribes successfully (response_format=text returns raw text body)", func(t *testing.T) {
		// Same defect class as the Groq provider hit on 2026-05-10:
		// the request sends response_format=text but the old code tried to
		// JSON-decode the body. Pin the actual OpenAI contract: when we ask
		// for text, the body is the raw transcript, not JSON.
		var reqBody map[string]any
		var authHeader string
		var contentType string
		var gotPath string
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			gotPath = r.URL.Path
			authHeader = r.Header.Get("Authorization")
			contentType = r.Header.Get("Content-Type")
			if err := r.ParseMultipartForm(1024 * 1024); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			reqBody = map[string]any{
				"model":           r.FormValue("model"),
				"language":        r.FormValue("language"),
				"response_format": r.FormValue("response_format"),
			}
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			// Real OpenAI behavior under response_format=text: raw transcript,
			// no JSON envelope. Leading 'H' specifically reproduces the
			// production failure that gave us "invalid character ..." on Groq.
			_, _ = w.Write([]byte("Hello world\n"))
		}))
		defer server.Close()

		audio := writeTestAudioFile(t, "test.ogg", []byte("fake audio content"))
		provider := NewTranscriptionOpenAIProvider(TranscriptionProviderConfig{
			APIKey:  "sk-test-key",
			BaseURL: server.URL + "/v1",
			Model:   "whisper-1",
			Timeout: 10 * time.Second,
		})

		result, err := provider.Transcribe(context.Background(), TranscriptionProviderRequest{
			AudioPath: audio,
			Provider:  "openai",
			Model:     "whisper-1",
			Language:  "en",
		})

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.Transcript != "Hello world" {
			t.Fatalf("expected raw text transcript, got %q", result.Transcript)
		}
		if result.Provider != "openai" {
			t.Fatalf("expected provider 'openai', got %q", result.Provider)
		}
		if !strings.HasPrefix(authHeader, "Bearer ") {
			t.Fatalf("expected Bearer auth header, got: %s", authHeader)
		}
		if reqBody["model"] != "whisper-1" {
			t.Fatalf("expected model whisper-1, got: %v", reqBody["model"])
		}
		if reqBody["language"] != "en" {
			t.Fatalf("expected language en, got: %v", reqBody["language"])
		}
		if reqBody["response_format"] != "text" {
			t.Fatalf("expected response_format=text (raw body contract), got: %v", reqBody["response_format"])
		}
		if gotPath != "/v1/audio/transcriptions" {
			t.Fatalf("request path = %q, want single /v1 prefix", gotPath)
		}
		_ = contentType // multipart form-data
	})

	t.Run("handles HTTP error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`{"error":{"message":"Invalid API key"}}`))
		}))
		defer server.Close()

		audio := writeTestAudioFile(t, "test.ogg", []byte("fake audio"))
		provider := NewTranscriptionOpenAIProvider(TranscriptionProviderConfig{
			APIKey:  "bad-key",
			BaseURL: server.URL,
			Timeout: 10 * time.Second,
		})

		_, err := provider.Transcribe(context.Background(), TranscriptionProviderRequest{
			AudioPath: audio,
		})

		if err == nil {
			t.Fatal("expected error for HTTP 401")
		}
		if !strings.Contains(err.Error(), "401") {
			t.Fatalf("error should contain status code: %v", err)
		}
	})

	t.Run("returns error without API key", func(t *testing.T) {
		provider := NewTranscriptionOpenAIProvider(TranscriptionProviderConfig{})
		audio := writeTestAudioFile(t, "test.ogg", []byte("fake audio"))

		_, err := provider.Transcribe(context.Background(), TranscriptionProviderRequest{
			AudioPath: audio,
		})

		if err == nil {
			t.Fatal("expected error without API key")
		}
		if !strings.Contains(err.Error(), "not configured") {
			t.Fatalf("error should mention API key not configured: %v", err)
		}
	})

	t.Run("returns error when audio file not found", func(t *testing.T) {
		provider := NewTranscriptionOpenAIProvider(TranscriptionProviderConfig{
			APIKey:  "sk-test-key",
			BaseURL: "https://api.openai.com/v1",
		})

		_, err := provider.Transcribe(context.Background(), TranscriptionProviderRequest{
			AudioPath: "/nonexistent/path/audio.ogg",
		})

		if err == nil {
			t.Fatal("expected error for missing audio file")
		}
		if !strings.Contains(err.Error(), "open audio") {
			t.Fatalf("error should mention open audio: %v", err)
		}
	})
}

// ---------------------------------------------------------------------------
// TranscriptionGroqProvider
// ---------------------------------------------------------------------------

func TestTranscriptionGroqProviderAvailable(t *testing.T) {
	orig := os.Getenv("GROQ_API_KEY")
	defer os.Setenv("GROQ_API_KEY", orig)

	t.Run("available with API key", func(t *testing.T) {
		os.Setenv("GROQ_API_KEY", "gsk-test-key")
		p := NewTranscriptionGroqProvider(TranscriptionProviderConfig{})
		if !p.Available(context.Background()) {
			t.Fatal("expected available with GROQ_API_KEY")
		}
	})

	t.Run("not available without API key", func(t *testing.T) {
		os.Unsetenv("GROQ_API_KEY")
		p := NewTranscriptionGroqProvider(TranscriptionProviderConfig{})
		if p.Available(context.Background()) {
			t.Fatal("expected not available without API key")
		}
	})
}

func TestTranscriptionGroqProviderTranscribe(t *testing.T) {
	t.Run("transcribes successfully (response_format=text returns raw text body)", func(t *testing.T) {
		// Live regression 2026-05-10: provider was sending
		// response_format=text but trying to JSON-decode the response,
		// producing "invalid character 'T' looking for beginning of value"
		// the moment a real transcript started with a non-{ character.
		// Pin the actual Groq contract: when we ask for text, the body
		// is the raw transcript, not JSON.
		var reqBody map[string]any
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if err := r.ParseMultipartForm(1024 * 1024); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			reqBody = map[string]any{
				"model":           r.FormValue("model"),
				"response_format": r.FormValue("response_format"),
			}
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			// Real Groq behavior under response_format=text: raw transcript,
			// no JSON envelope. Leading 'T' specifically reproduces the
			// production failure that gave us "invalid character 'T'".
			_, _ = w.Write([]byte("Testing the voice path end to end.\n"))
		}))
		defer server.Close()

		audio := writeTestAudioFile(t, "test.ogg", []byte("fake audio"))
		provider := NewTranscriptionGroqProvider(TranscriptionProviderConfig{
			APIKey:  "gsk-test-key",
			BaseURL: server.URL,
			Model:   "whisper-large-v3-turbo",
			Timeout: 10 * time.Second,
		})

		result, err := provider.Transcribe(context.Background(), TranscriptionProviderRequest{
			AudioPath: audio,
			Provider:  "groq",
		})

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.Transcript != "Testing the voice path end to end." {
			t.Fatalf("expected raw text transcript, got %q", result.Transcript)
		}
		if result.Provider != "groq" {
			t.Fatalf("expected provider 'groq', got %q", result.Provider)
		}
		if reqBody["model"] != "whisper-large-v3-turbo" {
			t.Fatalf("expected model whisper-large-v3-turbo, got: %v", reqBody["model"])
		}
		if reqBody["response_format"] != "text" {
			t.Fatalf("expected response_format=text (raw body contract), got: %v", reqBody["response_format"])
		}
	})

	t.Run("handles HTTP error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusTooManyRequests)
			w.Write([]byte(`{"error":"Rate limit exceeded"}`))
		}))
		defer server.Close()

		audio := writeTestAudioFile(t, "test.ogg", []byte("fake audio"))
		provider := NewTranscriptionGroqProvider(TranscriptionProviderConfig{
			APIKey:  "gsk-test-key",
			BaseURL: server.URL,
			Timeout: 10 * time.Second,
		})

		_, err := provider.Transcribe(context.Background(), TranscriptionProviderRequest{
			AudioPath: audio,
		})

		if err == nil {
			t.Fatal("expected error for HTTP 429")
		}
		if !strings.Contains(err.Error(), "429") {
			t.Fatalf("error should contain status code: %v", err)
		}
	})
}

// ---------------------------------------------------------------------------
// TranscriptionMistralProvider
// ---------------------------------------------------------------------------

func TestTranscriptionMistralProviderAvailable(t *testing.T) {
	orig := os.Getenv("MISTRAL_API_KEY")
	defer os.Setenv("MISTRAL_API_KEY", orig)

	t.Run("available with API key", func(t *testing.T) {
		os.Setenv("MISTRAL_API_KEY", "mistral-test-key")
		p := NewTranscriptionMistralProvider(TranscriptionProviderConfig{})
		if !p.Available(context.Background()) {
			t.Fatal("expected available with MISTRAL_API_KEY")
		}
	})

	t.Run("not available without API key", func(t *testing.T) {
		os.Unsetenv("MISTRAL_API_KEY")
		p := NewTranscriptionMistralProvider(TranscriptionProviderConfig{})
		if p.Available(context.Background()) {
			t.Fatal("expected not available without API key")
		}
	})
}

func TestTranscriptionMistralProviderTranscribe(t *testing.T) {
	t.Run("transcribes successfully", func(t *testing.T) {
		var receivedModel string
		var gotPath string
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			gotPath = r.URL.Path
			if err := r.ParseMultipartForm(1024 * 1024); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			receivedModel = r.FormValue("model")
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]string{"text": "Mistral transcript"})
		}))
		defer server.Close()

		audio := writeTestAudioFile(t, "test.mp3", []byte("fake audio"))
		provider := NewTranscriptionMistralProvider(TranscriptionProviderConfig{
			APIKey:  "mistral-test-key",
			BaseURL: server.URL + "/v1",
			Model:   "voxtral-mini-latest",
			Timeout: 10 * time.Second,
		})

		result, err := provider.Transcribe(context.Background(), TranscriptionProviderRequest{
			AudioPath: audio,
			Provider:  "mistral",
		})

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.Transcript != "Mistral transcript" {
			t.Fatalf("expected transcript 'Mistral transcript', got %q", result.Transcript)
		}
		if result.Provider != "mistral" {
			t.Fatalf("expected provider 'mistral', got %q", result.Provider)
		}
		if receivedModel != "voxtral-mini-latest" {
			t.Fatalf("expected model voxtral-mini-latest, got: %s", receivedModel)
		}
		if gotPath != "/v1/audio/transcriptions" {
			t.Fatalf("request path = %q, want single /v1 prefix", gotPath)
		}
	})
}

// ---------------------------------------------------------------------------
// TranscriptionXAIProvider
// ---------------------------------------------------------------------------

func TestTranscriptionXAIProviderAvailable(t *testing.T) {
	orig := os.Getenv("XAI_API_KEY")
	defer os.Setenv("XAI_API_KEY", orig)

	t.Run("available with API key", func(t *testing.T) {
		os.Setenv("XAI_API_KEY", "xai-test-key")
		p := NewTranscriptionXAIProvider(TranscriptionProviderConfig{})
		if !p.Available(context.Background()) {
			t.Fatal("expected available with XAI_API_KEY")
		}
	})

	t.Run("not available without API key", func(t *testing.T) {
		os.Unsetenv("XAI_API_KEY")
		p := NewTranscriptionXAIProvider(TranscriptionProviderConfig{})
		if p.Available(context.Background()) {
			t.Fatal("expected not available without API key")
		}
	})
}

func TestTranscriptionXAIProviderTranscribe(t *testing.T) {
	t.Run("transcribes successfully", func(t *testing.T) {
		var languageField string
		var formatField string
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if err := r.ParseMultipartForm(1024 * 1024); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			languageField = r.FormValue("language")
			formatField = r.FormValue("format")
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]any{
				"text":     "xAI Grok transcript",
				"language": "en",
				"duration": 3.5,
			})
		}))
		defer server.Close()

		audio := writeTestAudioFile(t, "test.ogg", []byte("fake audio"))
		provider := NewTranscriptionXAIProvider(TranscriptionProviderConfig{
			APIKey:  "xai-test-key",
			BaseURL: server.URL,
			Timeout: 10 * time.Second,
		})

		result, err := provider.Transcribe(context.Background(), TranscriptionProviderRequest{
			AudioPath: audio,
			Provider:  "xai",
			Language:  "en",
		})

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.Transcript != "xAI Grok transcript" {
			t.Fatalf("expected transcript 'xAI Grok transcript', got %q", result.Transcript)
		}
		if result.Provider != "xai" {
			t.Fatalf("expected provider 'xai', got %q", result.Provider)
		}
		if result.Model != "grok-stt" {
			t.Fatalf("expected model grok-stt, got %q", result.Model)
		}
		if languageField != "en" {
			t.Fatalf("expected language en, got: %s", languageField)
		}
		if formatField != "true" {
			t.Fatalf("expected format true, got: %s", formatField)
		}
	})

	t.Run("handles HTTP error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`{"error":{"message":"Invalid request"}}`))
		}))
		defer server.Close()

		audio := writeTestAudioFile(t, "test.ogg", []byte("fake audio"))
		provider := NewTranscriptionXAIProvider(TranscriptionProviderConfig{
			APIKey:  "xai-test-key",
			BaseURL: server.URL,
			Timeout: 10 * time.Second,
		})

		_, err := provider.Transcribe(context.Background(), TranscriptionProviderRequest{
			AudioPath: audio,
		})

		if err == nil {
			t.Fatal("expected error for HTTP 400")
		}
		if !strings.Contains(err.Error(), "400") {
			t.Fatalf("error should contain status code: %v", err)
		}
	})
}

// ---------------------------------------------------------------------------
// RegisterTranscriptionProviders
// ---------------------------------------------------------------------------

func TestRegisterTranscriptionProviders(t *testing.T) {
	origOpenAI := os.Getenv("GORMES_STT_OPENAI_KEY")
	origGroq := os.Getenv("GROQ_API_KEY")
	origMistral := os.Getenv("MISTRAL_API_KEY")
	origXAI := os.Getenv("XAI_API_KEY")
	defer func() {
		os.Setenv("GORMES_STT_OPENAI_KEY", origOpenAI)
		os.Setenv("GROQ_API_KEY", origGroq)
		os.Setenv("MISTRAL_API_KEY", origMistral)
		os.Setenv("XAI_API_KEY", origXAI)
	}()

	t.Run("registers all available providers", func(t *testing.T) {
		os.Setenv("GORMES_STT_OPENAI_KEY", "openai-key")
		os.Setenv("GROQ_API_KEY", "groq-key")
		os.Setenv("MISTRAL_API_KEY", "mistral-key")
		os.Setenv("XAI_API_KEY", "xai-key")

		into := make(map[string]TranscriptionProvider)
		RegisterTranscriptionProviders(into, TranscriptionProviderConfig{})

		if _, ok := into["openai"]; !ok {
			t.Error("expected openai provider to be registered")
		}
		if _, ok := into["groq"]; !ok {
			t.Error("expected groq provider to be registered")
		}
		if _, ok := into["mistral"]; !ok {
			t.Error("expected mistral provider to be registered")
		}
		if _, ok := into["xai"]; !ok {
			t.Error("expected xai provider to be registered")
		}
	})

	t.Run("skips unavailable providers", func(t *testing.T) {
		os.Unsetenv("GORMES_STT_OPENAI_KEY")
		os.Unsetenv("GROQ_API_KEY")
		os.Unsetenv("MISTRAL_API_KEY")
		os.Unsetenv("XAI_API_KEY")

		into := make(map[string]TranscriptionProvider)
		RegisterTranscriptionProviders(into, TranscriptionProviderConfig{})

		if len(into) != 0 {
			t.Fatalf("expected no providers registered without keys, got: %v", mapKeysTranscription(into))
		}
	})

	t.Run("partial registration when only one key set", func(t *testing.T) {
		os.Unsetenv("GORMES_STT_OPENAI_KEY")
		os.Setenv("GROQ_API_KEY", "groq-key")
		os.Unsetenv("MISTRAL_API_KEY")
		os.Unsetenv("XAI_API_KEY")

		into := make(map[string]TranscriptionProvider)
		RegisterTranscriptionProviders(into, TranscriptionProviderConfig{})

		if _, ok := into["openai"]; ok {
			t.Error("openai should not be registered without key")
		}
		if _, ok := into["groq"]; !ok {
			t.Error("expected groq provider registered")
		}
		if _, ok := into["mistral"]; ok {
			t.Error("mistral should not be registered without key")
		}
		if _, ok := into["xai"]; ok {
			t.Error("xai should not be registered without key")
		}
	})
}

// ---------------------------------------------------------------------------
// ValidateTranscriptionProviderConfig
// ---------------------------------------------------------------------------

func TestBuiltinTranscriptionProviderNamesIncludesLocalAndCloudMatrix(t *testing.T) {
	names := BuiltinTranscriptionProviderNames()
	for _, want := range []string{"device", "local", "openai", "groq", "mistral", "xai"} {
		if !transcriptionProviderNamePresent(names, want) {
			t.Fatalf("BuiltinTranscriptionProviderNames() = %#v, missing %q", names, want)
		}
	}
	if err := ValidateTranscriptionProviderConfig("local", TranscriptionProviderConfig{}); err != nil {
		t.Fatalf("ValidateTranscriptionProviderConfig(local): %v", err)
	}
}

func transcriptionProviderNamePresent(names []string, want string) bool {
	for _, name := range names {
		if name == want {
			return true
		}
	}
	return false
}

func TestValidateTranscriptionProviderConfig(t *testing.T) {
	// Save and restore env vars
	origOpenAI := os.Getenv("GORMES_STT_OPENAI_KEY")
	origOpenAIEnv := os.Getenv("OPENAI_API_KEY")
	origGroq := os.Getenv("GROQ_API_KEY")
	origMistral := os.Getenv("MISTRAL_API_KEY")
	origXAI := os.Getenv("XAI_API_KEY")
	defer func() {
		os.Setenv("GORMES_STT_OPENAI_KEY", origOpenAI)
		os.Setenv("OPENAI_API_KEY", origOpenAIEnv)
		os.Setenv("GROQ_API_KEY", origGroq)
		os.Setenv("MISTRAL_API_KEY", origMistral)
		os.Setenv("XAI_API_KEY", origXAI)
	}()

	t.Run("openai requires API key", func(t *testing.T) {
		os.Unsetenv("GORMES_STT_OPENAI_KEY")
		os.Unsetenv("OPENAI_API_KEY")
		os.Unsetenv("VOICE_TOOLS_OPENAI_KEY")
		err := ValidateTranscriptionProviderConfig("openai", TranscriptionProviderConfig{})
		if err == nil {
			t.Fatal("expected error when no OpenAI API key configured")
		}
		if !strings.Contains(err.Error(), "OpenAI") {
			t.Fatalf("error should mention OpenAI: %v", err)
		}
	})

	t.Run("openai accepts GORMES_STT_OPENAI_KEY", func(t *testing.T) {
		os.Setenv("GORMES_STT_OPENAI_KEY", "test-key")
		os.Unsetenv("OPENAI_API_KEY")
		err := ValidateTranscriptionProviderConfig("openai", TranscriptionProviderConfig{})
		if err != nil {
			t.Fatalf("expected no error with GORMES_STT_OPENAI_KEY set: %v", err)
		}
	})

	t.Run("openai accepts OPENAI_API_KEY", func(t *testing.T) {
		os.Unsetenv("GORMES_STT_OPENAI_KEY")
		os.Setenv("OPENAI_API_KEY", "openai-key")
		err := ValidateTranscriptionProviderConfig("openai", TranscriptionProviderConfig{})
		if err != nil {
			t.Fatalf("expected no error with OPENAI_API_KEY set: %v", err)
		}
	})

	t.Run("groq requires API key", func(t *testing.T) {
		os.Unsetenv("GROQ_API_KEY")
		err := ValidateTranscriptionProviderConfig("groq", TranscriptionProviderConfig{})
		if err == nil {
			t.Fatal("expected error when no Groq API key configured")
		}
		if !strings.Contains(err.Error(), "Groq") {
			t.Fatalf("error should mention Groq: %v", err)
		}
	})

	t.Run("groq accepts GROQ_API_KEY", func(t *testing.T) {
		os.Setenv("GROQ_API_KEY", "groq-key")
		err := ValidateTranscriptionProviderConfig("groq", TranscriptionProviderConfig{})
		if err != nil {
			t.Fatalf("expected no error with GROQ_API_KEY set: %v", err)
		}
	})

	t.Run("mistral requires API key", func(t *testing.T) {
		os.Unsetenv("MISTRAL_API_KEY")
		err := ValidateTranscriptionProviderConfig("mistral", TranscriptionProviderConfig{})
		if err == nil {
			t.Fatal("expected error when no Mistral API key configured")
		}
		if !strings.Contains(err.Error(), "Mistral") {
			t.Fatalf("error should mention Mistral: %v", err)
		}
	})

	t.Run("mistral accepts MISTRAL_API_KEY", func(t *testing.T) {
		os.Setenv("MISTRAL_API_KEY", "mistral-key")
		err := ValidateTranscriptionProviderConfig("mistral", TranscriptionProviderConfig{})
		if err != nil {
			t.Fatalf("expected no error with MISTRAL_API_KEY set: %v", err)
		}
	})

	t.Run("xai requires API key", func(t *testing.T) {
		os.Unsetenv("XAI_API_KEY")
		err := ValidateTranscriptionProviderConfig("xai", TranscriptionProviderConfig{})
		if err == nil {
			t.Fatal("expected error when no xAI API key configured")
		}
		if !strings.Contains(err.Error(), "xAI") {
			t.Fatalf("error should mention xAI: %v", err)
		}
	})

	t.Run("xai accepts XAI_API_KEY", func(t *testing.T) {
		os.Setenv("XAI_API_KEY", "xai-key")
		err := ValidateTranscriptionProviderConfig("xai", TranscriptionProviderConfig{})
		if err != nil {
			t.Fatalf("expected no error with XAI_API_KEY set: %v", err)
		}
	})

	t.Run("auto is always valid", func(t *testing.T) {
		err := ValidateTranscriptionProviderConfig("auto", TranscriptionProviderConfig{})
		if err != nil {
			t.Fatalf("auto selection should be valid: %v", err)
		}
	})

	t.Run("empty provider is valid", func(t *testing.T) {
		err := ValidateTranscriptionProviderConfig("", TranscriptionProviderConfig{})
		if err != nil {
			t.Fatalf("empty provider should be valid: %v", err)
		}
	})

	t.Run("unknown provider", func(t *testing.T) {
		err := ValidateTranscriptionProviderConfig("unknown", TranscriptionProviderConfig{})
		if err == nil {
			t.Fatal("expected error for unknown provider")
		}
		if !strings.Contains(err.Error(), "unknown") {
			t.Fatalf("error should mention unknown provider: %v", err)
		}
	})
}

// ---------------------------------------------------------------------------
// Integration with TranscriptionRunner
// ---------------------------------------------------------------------------

func mapKeysTranscription(m map[string]TranscriptionProvider) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}
