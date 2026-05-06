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

func TestMiniMaxTTSDefaultPayloadAndRawAudio(t *testing.T) {
	var gotPath string
	var gotAuth string
	var gotContentType string
	var gotPayload map[string]any

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotAuth = r.Header.Get("Authorization")
		gotContentType = r.Header.Get("Content-Type")
		if err := json.NewDecoder(r.Body).Decode(&gotPayload); err != nil {
			t.Fatalf("decode payload: %v", err)
		}
		w.Header().Set("Content-Type", "audio/mpeg")
		_, _ = w.Write([]byte{0x00, 0x01, 0x02, 0x03})
	}))
	defer server.Close()

	output := filepath.Join(t.TempDir(), "out.mp3")
	provider := NewTTSMiniMaxProvider(TTSProviderConfig{
		APIKey:  "mxp-test-key",
		Timeout: 10 * time.Second,
		ProviderConfig: map[string]any{
			"minimax": map[string]any{
				"base_url": server.URL + "/v1/text_to_speech",
			},
		},
	})

	result, err := provider.Synthesize(context.Background(), TTSProviderRequest{
		Text:       "Hello from MiniMax",
		OutputPath: output,
		Provider:   ProviderNameMiniMax,
	})
	if err != nil {
		t.Fatalf("synthesize: %v", err)
	}
	if result.FilePath != output {
		t.Fatalf("file path = %q, want %q", result.FilePath, output)
	}
	if gotPath != "/v1/text_to_speech" {
		t.Fatalf("path = %q, want /v1/text_to_speech", gotPath)
	}
	if gotAuth != "Bearer mxp-test-key" {
		t.Fatalf("authorization header = %q", gotAuth)
	}
	if gotContentType != "application/json" {
		t.Fatalf("content type = %q, want application/json", gotContentType)
	}
	if gotPayload["model"] != DefaultMiniMaxTTSModel {
		t.Fatalf("model = %v, want %s", gotPayload["model"], DefaultMiniMaxTTSModel)
	}
	if gotPayload["voice_id"] != DefaultMiniMaxTTSVoiceID {
		t.Fatalf("voice_id = %v, want %s", gotPayload["voice_id"], DefaultMiniMaxTTSVoiceID)
	}
	if gotPayload["text"] != "Hello from MiniMax" {
		t.Fatalf("text = %v", gotPayload["text"])
	}
	for _, legacy := range []string{"voice_setting", "audio_setting", "stream"} {
		if _, ok := gotPayload[legacy]; ok {
			t.Fatalf("payload contains legacy field %q: %#v", legacy, gotPayload)
		}
	}
	data, err := os.ReadFile(output)
	if err != nil {
		t.Fatalf("read output: %v", err)
	}
	if string(data) != string([]byte{0x00, 0x01, 0x02, 0x03}) {
		t.Fatalf("raw audio = %v", data)
	}
}

func TestMiniMaxTTSLegacyHexJSONFallback(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"base_resp":{"status_code":0},"data":{"audio":"00010203"}}`))
	}))
	defer server.Close()

	output := filepath.Join(t.TempDir(), "legacy.mp3")
	provider := NewTTSMiniMaxProvider(TTSProviderConfig{
		APIKey: "mxp-test-key",
		ProviderConfig: map[string]any{
			"minimax": map[string]any{"base_url": server.URL},
		},
	})

	if _, err := provider.Synthesize(context.Background(), TTSProviderRequest{
		Text:       "legacy",
		OutputPath: output,
		Provider:   ProviderNameMiniMax,
	}); err != nil {
		t.Fatalf("synthesize legacy: %v", err)
	}
	data, err := os.ReadFile(output)
	if err != nil {
		t.Fatalf("read output: %v", err)
	}
	if string(data) != string([]byte{0x00, 0x01, 0x02, 0x03}) {
		t.Fatalf("legacy audio = %v", data)
	}
}

func TestMiniMaxTTSFailuresUseTTSEvidenceAndRedaction(t *testing.T) {
	tests := []struct {
		name        string
		status      int
		contentType string
		body        string
		want        string
	}{
		{
			name:        "http api error",
			status:      http.StatusUnauthorized,
			contentType: "application/json",
			body:        `{"base_resp":{"status_code":1001,"status_msg":"invalid key: mxp-secret-12345678901234567890"}}`,
			want:        "MiniMax TTS API error",
		},
		{
			name:        "missing audio",
			status:      http.StatusOK,
			contentType: "application/json",
			body:        `{"base_resp":{"status_code":0},"data":{}}`,
			want:        "empty audio data",
		},
		{
			name:        "malformed body",
			status:      http.StatusOK,
			contentType: "text/plain",
			body:        `not json`,
			want:        "unexpected Content-Type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", tt.contentType)
				w.WriteHeader(tt.status)
				_, _ = w.Write([]byte(tt.body))
			}))
			defer server.Close()

			runner := NewTTSRunner(TTSConfig{
				Provider:  ProviderNameMiniMax,
				OutputDir: t.TempDir(),
			}, map[string]TTSProvider{
				ProviderNameMiniMax: NewTTSMiniMaxProvider(TTSProviderConfig{
					APIKey: "mxp-secret-12345678901234567890",
					ProviderConfig: map[string]any{
						"minimax": map[string]any{"base_url": server.URL},
					},
				}),
			})

			result := runner.Synthesize(context.Background(), TTSRequest{
				Text:     "private prompt that must not appear",
				Provider: ProviderNameMiniMax,
			})
			if result.Success {
				t.Fatalf("expected failure, got %+v", result)
			}
			if result.Evidence != TTSEvidenceAPIError {
				t.Fatalf("evidence = %s, want %s", result.Evidence, TTSEvidenceAPIError)
			}
			if !strings.Contains(result.Error, tt.want) {
				t.Fatalf("error = %q, want substring %q", result.Error, tt.want)
			}
			if strings.Contains(result.Error, "mxp-secret-12345678901234567890") {
				t.Fatalf("error leaked API key: %q", result.Error)
			}
			if strings.Contains(result.Error, "private prompt") {
				t.Fatalf("error leaked prompt text: %q", result.Error)
			}
		})
	}
}
