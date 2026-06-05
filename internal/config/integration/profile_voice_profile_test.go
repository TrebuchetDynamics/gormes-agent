package integration_test

import (
	. "github.com/TrebuchetDynamics/gormes-agent/internal/config"

	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestProfileVoiceProfileConfigLoadsAndValidatesProviderMatrix(t *testing.T) {
	home := t.TempDir()
	t.Setenv("GORMES_HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, "xdg"))
	if err := os.MkdirAll(home, 0o700); err != nil {
		t.Fatal(err)
	}
	body := []byte(`
config_version = 2

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
`)
	if err := os.WriteFile(ConfigPath(), body, 0o600); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(nil)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	voice := cfg.Profiles["main"].VoiceProfile
	if voice.STTProvider != "local" || voice.TTSProvider != "piper" || voice.VoiceID != "amy" || voice.LanguagePolicy != "match_user_language" || voice.FallbackVoice != "text_only" {
		t.Fatalf("voice profile = %+v, want loaded per-profile fields", voice)
	}
	validation := ValidateProfileVoiceProfile("main", voice, ProfileVoiceProviderMatrix{
		STTProviders: []string{"device", "local", "openai"},
		TTSProviders: []string{"piper", "openai"},
	})
	if !validation.Valid || len(validation.Errors) != 0 {
		t.Fatalf("validation = %+v, want valid", validation)
	}
	if !validation.CredentialStatusRefs["stt"].Configured || !validation.CredentialStatusRefs["tts"].Configured {
		t.Fatalf("credential status = %+v, want configured refs", validation.CredentialStatusRefs)
	}
	raw, err := json.Marshal(validation)
	if err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{"private-stt-ref", "private-tts-ref"} {
		if strings.Contains(string(raw), forbidden) {
			t.Fatalf("validation leaked credential ref %q: %s", forbidden, raw)
		}
	}
}

func TestProfileVoiceProfileValidationRejectsUnknownProviders(t *testing.T) {
	validation := ValidateProfileVoiceProfile("main", ProfileVoiceProfileCfg{
		STTProvider:    "bogus-stt",
		TTSProvider:    "openai",
		VoiceID:        "alloy",
		LanguagePolicy: "match_user_language",
		FallbackVoice:  "text_only",
	}, ProfileVoiceProviderMatrix{
		STTProviders: []string{"device", "local", "openai"},
		TTSProviders: []string{"piper", "openai"},
	})
	if validation.Valid || len(validation.Errors) == 0 {
		t.Fatalf("validation = %+v, want invalid provider error", validation)
	}
	if validation.Errors[0].Field != "stt_provider" || validation.Errors[0].Code != "unknown_provider" {
		t.Fatalf("first error = %+v, want stt unknown_provider", validation.Errors[0])
	}
}

func TestWriteProfileConfigV2PersistsVoiceProfile(t *testing.T) {
	home := t.TempDir()
	path := filepath.Join(home, "config.toml")
	cfg := Config{Profiles: map[string]ProfileCfg{
		"main": {
			Enabled: true,
			Name:    "Main Desk",
			VoiceProfile: ProfileVoiceProfileCfg{
				STTProvider:    "local",
				TTSProvider:    "piper",
				VoiceID:        "amy",
				LanguagePolicy: "match_user_language",
				FallbackVoice:  "text_only",
			},
		},
	}}
	if err := WriteProfileConfigV2(path, cfg); err != nil {
		t.Fatalf("WriteProfileConfigV2: %v", err)
	}
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	text := string(body)
	for _, want := range []string{"[profiles.main.voice_profile]", `stt_provider = 'local'`, `tts_provider = 'piper'`, `voice_id = 'amy'`, `fallback_voice = 'text_only'`} {
		if !strings.Contains(text, want) {
			t.Fatalf("config body missing %q:\n%s", want, text)
		}
	}
}
