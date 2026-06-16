package profile

import (
	"sort"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config/credentials"
)

func ProfilesDocument(profiles map[string]Config) map[string]any {
	out := map[string]any{}
	for _, id := range SortedKeys(profiles) {
		profile := profiles[id]
		entry := map[string]any{
			"enabled": profile.Enabled,
			"name":    profile.Name,
		}
		if profile.Description != "" {
			entry["description"] = profile.Description
		}
		if profile.Workspace != "" {
			entry["workspace"] = profile.Workspace
		}
		if len(profile.Workspaces) > 0 {
			entry["workspaces"] = append([]string(nil), profile.Workspaces...)
		}
		if len(profile.AllowedPaths) > 0 {
			entry["allowed_paths"] = append([]string(nil), profile.AllowedPaths...)
		}
		if len(profile.AllowedPathRules) > 0 {
			entry["allowed_path"] = AllowedPathRulesDocument(profile.AllowedPathRules)
		}
		if len(profile.Tags) > 0 {
			entry["tags"] = append([]string(nil), profile.Tags...)
		}
		if len(profile.Settings) > 0 {
			entry["settings"] = profile.Settings
		}
		if runtime := RuntimeDocument(profile.Runtime); len(runtime) > 0 {
			entry["runtime"] = runtime
		}
		if voice := VoiceProfileDocument(profile.VoiceProfile); len(voice) > 0 {
			entry["voice_profile"] = voice
		}
		if providers := ProvidersDocument(profile.Providers); len(providers) > 0 {
			entry["providers"] = providers
		}
		if channels := ChannelsDocument(profile.Channels); len(channels) > 0 {
			entry["channels"] = channels
		}
		out[id] = entry
	}
	return out
}

func AllowedPathRulesDocument(rules []AllowedPathConfig) []map[string]any {
	out := make([]map[string]any, 0, len(rules))
	for _, rule := range rules {
		entry := map[string]any{"path": rule.Path}
		if rule.Access != "" {
			entry["access"] = rule.Access
		}
		out = append(out, entry)
	}
	return out
}

func RuntimeDocument(runtime RuntimeConfig) map[string]any {
	out := map[string]any{}
	if runtime.MaxTurns != 0 {
		out["max_turns"] = int64(runtime.MaxTurns)
	}
	if runtime.SessionResetPolicy != "" {
		out["session_reset_policy"] = runtime.SessionResetPolicy
	}
	if runtime.SessionResetAfterMins != 0 {
		out["session_reset_after_minutes"] = int64(runtime.SessionResetAfterMins)
	}
	if runtime.GonchoWorkspace != "" {
		out["goncho_workspace"] = runtime.GonchoWorkspace
	}
	return out
}

func VoiceProfileDocument(voice VoiceProfileConfig) map[string]any {
	voice = NormalizeVoiceProfile(voice)
	out := map[string]any{}
	if voice.STTProvider != "" {
		out["stt_provider"] = voice.STTProvider
	}
	if voice.TTSProvider != "" {
		out["tts_provider"] = voice.TTSProvider
	}
	if voice.VoiceID != "" {
		out["voice_id"] = voice.VoiceID
	}
	if voice.LanguagePolicy != "" {
		out["language_policy"] = voice.LanguagePolicy
	}
	if voice.FallbackVoice != "" {
		out["fallback_voice"] = voice.FallbackVoice
	}
	if voice.STTCredential != "" {
		out["stt_credential"] = voice.STTCredential
	}
	if voice.TTSCredential != "" {
		out["tts_credential"] = voice.TTSCredential
	}
	return out
}

func ProvidersDocument(providers map[string]ProviderConfig) map[string]any {
	out := map[string]any{}
	for _, id := range SortedKeys(providers) {
		provider := providers[id]
		entry := map[string]any{"enabled": provider.Enabled}
		if provider.Credential != "" {
			entry["credential"] = provider.Credential
		}
		if provider.DefaultModel != "" {
			entry["default_model"] = provider.DefaultModel
		}
		if len(provider.AllowedModels) > 0 {
			entry["allowed_models"] = append([]string(nil), provider.AllowedModels...)
		}
		if provider.Endpoint != "" {
			entry["endpoint"] = provider.Endpoint
		}
		out[id] = entry
	}
	return out
}

func ChannelsDocument(channels map[string]ChannelConfig) map[string]any {
	out := map[string]any{}
	for _, id := range SortedKeys(channels) {
		channel := channels[id]
		entry := map[string]any{"enabled": channel.Enabled}
		if channel.Credential != "" {
			entry["credential"] = channel.Credential
		}
		if len(channel.AllowedChats) > 0 {
			entry["allowed_chats"] = append([]string(nil), channel.AllowedChats...)
		}
		if len(channel.AllowedUsers) > 0 {
			entry["allowed_users"] = append([]string(nil), channel.AllowedUsers...)
		}
		if channel.RequireMention {
			entry["require_mention"] = channel.RequireMention
		}
		if channel.ToolProgress != "" {
			entry["tool_progress"] = channel.ToolProgress
		}
		if len(channel.Servers) > 0 {
			entry["servers"] = append([]string(nil), channel.Servers...)
		}
		if channel.VoiceProfile != "" {
			entry["voice_profile"] = channel.VoiceProfile
		}
		out[id] = entry
	}
	return out
}

func CredentialsDocument(credentialsByID map[string]CredentialConfig) map[string]any {
	out := map[string]any{}
	for _, id := range SortedKeys(credentialsByID) {
		credential := credentialsByID[id]
		entry := map[string]any{}
		if credential.Kind != "" {
			entry["kind"] = credential.Kind
		}
		if credential.Provider != "" {
			entry["provider"] = credential.Provider
		}
		if credential.Channel != "" {
			entry["channel"] = credential.Channel
		}
		if credential.OwnerProfile != "" {
			entry["owner_profile"] = credential.OwnerProfile
		}
		if credential.SecretRef != nil {
			entry["secret_ref"] = SecretRefDocument(*credential.SecretRef)
		}
		out[id] = entry
	}
	return out
}

func SecretRefDocument(ref credentials.SecretRef) map[string]any {
	out := map[string]any{"source": string(ref.Source), "id": ref.ID}
	if ref.Provider != "" {
		out["provider"] = ref.Provider
	}
	return out
}

func SortedKeys[T any](values map[string]T) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
