package navivox

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway"
)

func TestNavivoxVoiceProfilesReadRedactsCredentialsAndValidatesProviders(t *testing.T) {
	home := filepath.Join(t.TempDir(), "gormes")
	t.Setenv("GORMES_HOME", home)
	t.Setenv("GORMES_NAVIVOX_TOKEN", "existing-secret-token")
	if err := os.MkdirAll(home, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(home, "config.toml"), []byte(`
config_version = 2

[navivox]
enabled = true
bind_host = "127.0.0.1"
port = 8765
exposure_mode = "local"
auth_mode = "pairing_token"

[profiles.main]
enabled = true
name = "Main Desk"

[profiles.main.voice_profile]
stt_provider = "local"
tts_provider = "piper"
voice_id = "amy"
language_policy = "match_user_language"
fallback_voice = "text_only"
stt_credential = "private-stt-ref"
tts_credential = "private-tts-ref"
`), 0o600); err != nil {
		t.Fatal(err)
	}

	ch := newTestChannel(t)
	server := httptest.NewServer(ch.Handler(make(chan gateway.InboundEvent, 1)))
	defer server.Close()
	httpc := newNavivoxHTTPContract(t, server.URL)

	var got struct {
		Action   string `json:"action"`
		Profiles []struct {
			ProfileID string `json:"profile_id"`
			Voice     struct {
				STTProvider    string `json:"stt_provider"`
				TTSProvider    string `json:"tts_provider"`
				VoiceID        string `json:"voice_id"`
				LanguagePolicy string `json:"language_policy"`
				FallbackVoice  string `json:"fallback_voice"`
			} `json:"voice_profile"`
			CredentialStatusRefs map[string]struct {
				Configured bool   `json:"configured"`
				Status     string `json:"status"`
			} `json:"credential_status_refs"`
			Valid bool `json:"valid"`
		} `json:"profiles"`
		ProviderMatrix struct {
			STT []string `json:"stt"`
			TTS []string `json:"tts"`
		} `json:"provider_matrix"`
	}
	httpc.JSON(http.MethodGet, "/v1/navivox/voice-profiles", "", http.StatusOK, &got)
	if got.Action != "voice_profiles.get" || len(got.Profiles) != 1 || got.Profiles[0].ProfileID != "main" {
		t.Fatalf("voice profile payload = %+v", got)
	}
	if got.Profiles[0].Voice.STTProvider != "local" || got.Profiles[0].Voice.TTSProvider != "piper" || got.Profiles[0].Voice.VoiceID != "amy" || !got.Profiles[0].Valid {
		t.Fatalf("profile voice state = %+v, want loaded valid profile", got.Profiles[0])
	}
	if !got.Profiles[0].CredentialStatusRefs["stt"].Configured || !got.Profiles[0].CredentialStatusRefs["tts"].Configured {
		t.Fatalf("credential status = %+v, want configured redacted refs", got.Profiles[0].CredentialStatusRefs)
	}
	if !stringListContains(got.ProviderMatrix.STT, "local") || !stringListContains(got.ProviderMatrix.TTS, "piper") {
		t.Fatalf("provider matrix = %+v, want local STT and piper TTS", got.ProviderMatrix)
	}
	raw, err := json.Marshal(got)
	if err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{"private-stt-ref", "private-tts-ref", "existing-secret-token"} {
		if strings.Contains(string(raw), forbidden) {
			t.Fatalf("voice profile payload leaked %q: %s", forbidden, raw)
		}
	}

	var invalid struct {
		Action string `json:"action"`
		Valid  bool   `json:"valid"`
		Errors []struct {
			Field string `json:"field"`
			Code  string `json:"code"`
		} `json:"errors"`
	}
	httpc.JSON(http.MethodPost, "/v1/navivox/voice-profiles/validate", `{"profile_id":"main","voice_profile":{"stt_provider":"bogus","tts_provider":"piper","fallback_voice":"text_only"}}`, http.StatusUnprocessableEntity, &invalid)
	if invalid.Action != "voice_profiles.validate" || invalid.Valid || len(invalid.Errors) == 0 || invalid.Errors[0].Field != "stt_provider" {
		t.Fatalf("invalid response = %+v, want field scoped stt error", invalid)
	}

	httpc.Raw(http.MethodPost, "/v1/navivox/voice-profiles/validate", `{"profile_id":"main","voice_profile":{"stt_provider":"local","tts_provider":"piper","voice_id":"amy","language_policy":"match_user_language","fallback_voice":"text_only"}}`, http.StatusOK)
}

func TestNavivoxVoiceTurnResolvesProfileVoiceFallbackEvidence(t *testing.T) {
	home := filepath.Join(t.TempDir(), "gormes")
	t.Setenv("GORMES_HOME", home)
	t.Setenv("GORMES_NAVIVOX_TOKEN", "existing-secret-token")
	if err := os.MkdirAll(home, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(home, "config.toml"), []byte(`
config_version = 2

[navivox]
enabled = true
bind_host = "127.0.0.1"
port = 8765
exposure_mode = "local"
auth_mode = "pairing_token"

[profiles.main]
enabled = true
name = "Main Desk"

[profiles.main.voice_profile]
stt_provider = "local"
tts_provider = "openai"
voice_id = "alloy"
language_policy = "match_user_language"
fallback_voice = "text_only"
`), 0o600); err != nil {
		t.Fatal(err)
	}

	ch := newTestChannel(t)
	inbox := make(chan gateway.InboundEvent, 1)
	server := httptest.NewServer(ch.Handler(inbox))
	defer server.Close()
	httpc := newNavivoxHTTPContract(t, server.URL)

	turn := `{"request_id":"req-profile-voice","session_id":"s-profile-voice","text":"hello by voice","metadata":{"input_kind":"voice","profile_id":"main","server_id":"local","audio_duration_ms":900}}`
	httpc.Raw(http.MethodPost, "/v1/navivox/turn", turn, http.StatusAccepted)
	select {
	case <-inbox:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for queued voice turn")
	}

	var got struct {
		Record struct {
			Voice struct {
				ServerSTT struct {
					Provider string `json:"provider"`
					Status   string `json:"status"`
				} `json:"server_stt"`
				TTS struct {
					Provider string `json:"provider"`
					VoiceID  string `json:"voice_id"`
					Status   string `json:"status"`
				} `json:"tts"`
			} `json:"voice"`
		} `json:"run_record"`
	}
	httpc.JSON(http.MethodGet, "/v1/navivox/run-records/req-profile-voice", "", http.StatusOK, &got)
	if got.Record.Voice.ServerSTT.Provider != "local" || got.Record.Voice.ServerSTT.Status != "available" {
		t.Fatalf("server STT evidence = %+v, want local available", got.Record.Voice.ServerSTT)
	}
	if got.Record.Voice.TTS.Provider != "text_only" || got.Record.Voice.TTS.Status != "fallback" || got.Record.Voice.TTS.VoiceID != "text_only" {
		t.Fatalf("TTS evidence = %+v, want text_only fallback", got.Record.Voice.TTS)
	}
}

func stringListContains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
