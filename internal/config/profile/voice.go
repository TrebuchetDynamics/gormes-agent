package profile

import (
	"fmt"
	"strings"
)

func NormalizeVoiceProfile(voice VoiceProfileConfig) VoiceProfileConfig {
	voice.STTProvider = strings.ToLower(strings.TrimSpace(voice.STTProvider))
	voice.TTSProvider = strings.ToLower(strings.TrimSpace(voice.TTSProvider))
	voice.VoiceID = strings.TrimSpace(voice.VoiceID)
	voice.LanguagePolicy = strings.ToLower(strings.TrimSpace(voice.LanguagePolicy))
	voice.FallbackVoice = strings.TrimSpace(voice.FallbackVoice)
	voice.STTCredential = strings.TrimSpace(voice.STTCredential)
	voice.TTSCredential = strings.TrimSpace(voice.TTSCredential)
	return voice
}

func ValidateVoiceProfile(profileID string, voice VoiceProfileConfig, matrix VoiceProviderMatrix) VoiceProfileValidation {
	voice = NormalizeVoiceProfile(voice)
	validation := VoiceProfileValidation{
		ProfileID:            strings.TrimSpace(profileID),
		VoiceProfile:         voice,
		Valid:                true,
		CredentialStatusRefs: map[string]VoiceCredentialStatus{},
	}
	sttProviders := normalizedProviderSet(matrix.STTProviders)
	ttsProviders := normalizedProviderSet(matrix.TTSProviders)
	if voice.STTProvider != "" && !sttProviders[voice.STTProvider] {
		validation.Errors = append(validation.Errors, VoiceProfileFieldError{Field: "stt_provider", Code: "unknown_provider", Message: fmt.Sprintf("unknown STT provider %q", voice.STTProvider)})
	}
	if voice.TTSProvider != "" && !ttsProviders[voice.TTSProvider] {
		validation.Errors = append(validation.Errors, VoiceProfileFieldError{Field: "tts_provider", Code: "unknown_provider", Message: fmt.Sprintf("unknown TTS provider %q", voice.TTSProvider)})
	}
	validation.CredentialStatusRefs["stt"] = voiceCredentialStatus("stt", voice.STTProvider, voice.STTCredential)
	validation.CredentialStatusRefs["tts"] = voiceCredentialStatus("tts", voice.TTSProvider, voice.TTSCredential)
	validation.Valid = len(validation.Errors) == 0
	return validation
}

func normalizedProviderSet(values []string) map[string]bool {
	out := map[string]bool{}
	for _, value := range values {
		value = strings.ToLower(strings.TrimSpace(value))
		if value != "" {
			out[value] = true
		}
	}
	return out
}

func voiceCredentialStatus(kind, provider, credential string) VoiceCredentialStatus {
	provider = strings.ToLower(strings.TrimSpace(provider))
	credential = strings.TrimSpace(credential)
	required := voiceProviderRequiresCredential(kind, provider)
	switch {
	case provider == "":
		return VoiceCredentialStatus{Configured: false, Required: false, Status: "unset"}
	case credential != "":
		return VoiceCredentialStatus{Configured: true, Required: required, Status: "configured", Source: "profile_voice_profile." + kind + "_credential"}
	case !required:
		return VoiceCredentialStatus{Configured: true, Required: false, Status: "not_required", Source: provider}
	default:
		return VoiceCredentialStatus{Configured: false, Required: true, Status: "missing"}
	}
}

func voiceProviderRequiresCredential(kind, provider string) bool {
	provider = strings.ToLower(strings.TrimSpace(provider))
	if provider == "" {
		return false
	}
	if kind == "stt" {
		switch provider {
		case "device", "local":
			return false
		default:
			return true
		}
	}
	switch provider {
	case "local_go", "local_fixture", "piper", "neutts", "kittentts", "local", "text_only":
		return false
	default:
		return true
	}
}
