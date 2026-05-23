package navivox

import (
	"encoding/json"
	"net/http"
	"sort"
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tools"
)

type voiceProfileBackend struct {
	load func() (config.Config, error)
}

func defaultVoiceProfileBackend() voiceProfileBackend {
	return voiceProfileBackend{load: func() (config.Config, error) { return config.Load(nil) }}
}

type voiceProfileRequest struct {
	ProfileID    string                        `json:"profile_id"`
	VoiceProfile config.ProfileVoiceProfileCfg `json:"voice_profile"`
}

type voiceProfileResponse struct {
	Action         string                                 `json:"action"`
	ProviderMatrix config.ProfileVoiceProviderMatrix      `json:"provider_matrix"`
	Profiles       []voiceProfileView                     `json:"profiles,omitempty"`
	Validation     *config.ProfileVoiceProfileValidation  `json:"validation,omitempty"`
	Valid          bool                                   `json:"valid,omitempty"`
	Errors         []config.ProfileVoiceProfileFieldError `json:"errors,omitempty"`
}

type voiceProfileView struct {
	ProfileID            string                                         `json:"profile_id"`
	DisplayName          string                                         `json:"display_name,omitempty"`
	VoiceProfile         config.ProfileVoiceProfileCfg                  `json:"voice_profile"`
	CredentialStatusRefs map[string]config.ProfileVoiceCredentialStatus `json:"credential_status_refs,omitempty"`
	Valid                bool                                           `json:"valid"`
	Errors               []config.ProfileVoiceProfileFieldError         `json:"errors,omitempty"`
}

func (c *Channel) handleVoiceProfiles(w http.ResponseWriter, r *http.Request, _ string) {
	path := strings.TrimPrefix(r.URL.Path, "/v1/navivox/voice-profiles")
	path = strings.Trim(path, "/")
	switch {
	case r.Method == http.MethodGet && path == "":
		cfg, err := c.voiceProfiles.loadConfig()
		if err != nil {
			writeNavivoxError(w, http.StatusServiceUnavailable, "", "voice_profiles_unavailable", "Voice profiles are unavailable")
			return
		}
		writeNavivoxJSON(w, http.StatusOK, voiceProfileResponse{Action: "voice_profiles.get", ProviderMatrix: navivoxVoiceProviderMatrix(), Profiles: voiceProfileViews(cfg)})
	case r.Method == http.MethodPost && path == "validate":
		var req voiceProfileRequest
		dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
		dec.DisallowUnknownFields()
		if err := dec.Decode(&req); err != nil {
			writeNavivoxError(w, http.StatusBadRequest, "", "bad_request", "Invalid voice profile request")
			return
		}
		validation := config.ValidateProfileVoiceProfile(req.ProfileID, req.VoiceProfile, navivoxVoiceProviderMatrix())
		payload := voiceProfileResponse{Action: "voice_profiles.validate", ProviderMatrix: navivoxVoiceProviderMatrix(), Validation: &validation, Valid: validation.Valid, Errors: validation.Errors}
		if !validation.Valid {
			writeNavivoxJSON(w, http.StatusUnprocessableEntity, payload)
			return
		}
		writeNavivoxJSON(w, http.StatusOK, payload)
	case r.Method == http.MethodGet || r.Method == http.MethodPost:
		writeNavivoxError(w, http.StatusNotFound, "", "not_found", "Voice profile route not found")
	default:
		writeNavivoxError(w, http.StatusMethodNotAllowed, "", "bad_request", "Method not allowed")
	}
}

func (b voiceProfileBackend) loadConfig() (config.Config, error) {
	if b.load == nil {
		return config.Load(nil)
	}
	return b.load()
}

func voiceProfileViews(cfg config.Config) []voiceProfileView {
	ids := make([]string, 0, len(cfg.Profiles))
	for id, profile := range cfg.Profiles {
		if profile.Enabled {
			ids = append(ids, id)
		}
	}
	sort.Strings(ids)
	views := make([]voiceProfileView, 0, len(ids))
	matrix := navivoxVoiceProviderMatrix()
	for _, id := range ids {
		profile := cfg.Profiles[id]
		validation := config.ValidateProfileVoiceProfile(id, profile.VoiceProfile, matrix)
		displayName := strings.TrimSpace(profile.Name)
		if displayName == "" {
			displayName = id
		}
		views = append(views, voiceProfileView{ProfileID: id, DisplayName: displayName, VoiceProfile: validation.VoiceProfile, CredentialStatusRefs: validation.CredentialStatusRefs, Valid: validation.Valid, Errors: validation.Errors})
	}
	return views
}

func navivoxVoiceProviderMatrix() config.ProfileVoiceProviderMatrix {
	stt := tools.BuiltinTranscriptionProviderNames()
	tts := tools.BuiltinTTSProviderNames()
	return config.ProfileVoiceProviderMatrix{STTProviders: uniqueSortedVoiceProviders(stt), TTSProviders: uniqueSortedVoiceProviders(tts)}
}

func uniqueSortedVoiceProviders(values []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.ToLower(strings.TrimSpace(value))
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}

func (c *Channel) voiceProfileMetadataForTurn(metadata map[string]any, profileID string) map[string]any {
	if !navivoxTurnMetadataMarksVoice(metadata) || strings.TrimSpace(profileID) == "" {
		return metadata
	}
	cfg, err := c.voiceProfiles.loadConfig()
	if err != nil || len(cfg.Profiles) == 0 {
		return metadata
	}
	profile, ok := cfg.Profiles[profileID]
	if !ok || !profile.Enabled {
		return metadata
	}
	validation := config.ValidateProfileVoiceProfile(profileID, profile.VoiceProfile, navivoxVoiceProviderMatrix())
	if !validation.Valid {
		return metadata
	}
	out := cloneNavivoxMetadata(metadata)
	voice := validation.VoiceProfile
	if strings.TrimSpace(voice.STTProvider) != "" && strings.TrimSpace(anyString(out["server_stt_provider"])) == "" {
		out["server_stt_provider"] = voice.STTProvider
		status := "available"
		if ref := validation.CredentialStatusRefs["stt"]; ref.Required && !ref.Configured {
			status = "unavailable"
		}
		out["server_stt_status"] = status
	}
	if strings.TrimSpace(voice.TTSProvider) != "" && strings.TrimSpace(anyString(out["tts_provider"])) == "" {
		status := "available"
		provider := voice.TTSProvider
		voiceID := voice.VoiceID
		if ref := validation.CredentialStatusRefs["tts"]; ref.Required && !ref.Configured {
			status = "fallback"
			provider = strings.TrimSpace(voice.FallbackVoice)
			if provider == "" {
				provider = "text_only"
			}
			voiceID = provider
		}
		out["tts_provider"] = provider
		out["tts_status"] = status
		if strings.TrimSpace(voiceID) != "" {
			out["tts_voice_id"] = voiceID
		}
	}
	return out
}

func navivoxTurnMetadataMarksVoice(metadata map[string]any) bool {
	if len(metadata) == 0 {
		return false
	}
	if strings.EqualFold(anyString(metadata["input_kind"]), "voice") || boolFromNavivoxMetadata(metadata["voice"]) {
		return true
	}
	for key := range metadata {
		key = strings.ToLower(strings.TrimSpace(key))
		if strings.Contains(key, "stt") || strings.Contains(key, "tts") || strings.Contains(key, "audio_") {
			return true
		}
	}
	return false
}

func cloneNavivoxMetadata(metadata map[string]any) map[string]any {
	out := map[string]any{}
	for key, value := range metadata {
		out[key] = value
	}
	return out
}

func boolFromNavivoxMetadata(value any) bool {
	switch v := value.(type) {
	case bool:
		return v
	case string:
		return strings.EqualFold(strings.TrimSpace(v), "true") || strings.EqualFold(strings.TrimSpace(v), "yes") || strings.EqualFold(strings.TrimSpace(v), "voice")
	default:
		return false
	}
}
