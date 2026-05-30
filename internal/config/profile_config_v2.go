package config

import (
	"fmt"
	"sort"
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config/profilestorage"
)

const DefaultProfileID = profilestorage.DefaultProfileID

type ProfileCfg struct {
	Enabled      bool                          `toml:"enabled" yaml:"enabled"`
	Name         string                        `toml:"name" yaml:"name"`
	Description  string                        `toml:"description" yaml:"description"`
	Workspaces   []string                      `toml:"workspaces" yaml:"workspaces"`
	Tags         []string                      `toml:"tags" yaml:"tags"`
	Settings     map[string]any                `toml:"settings" yaml:"settings"`
	Runtime      ProfileRuntimeCfg             `toml:"runtime" yaml:"runtime"`
	Providers    map[string]ProfileProviderCfg `toml:"providers" yaml:"providers"`
	Channels     map[string]ProfileChannelCfg  `toml:"channels" yaml:"channels"`
	VoiceProfile ProfileVoiceProfileCfg        `toml:"voice_profile" yaml:"voice_profile" json:"voice_profile,omitempty"`
}

type ProfileRuntimeCfg struct {
	MaxTurns              int    `toml:"max_turns" yaml:"max_turns"`
	SessionResetPolicy    string `toml:"session_reset_policy" yaml:"session_reset_policy"`
	SessionResetAfterMins int    `toml:"session_reset_after_minutes" yaml:"session_reset_after_minutes"`
	GonchoWorkspace       string `toml:"goncho_workspace" yaml:"goncho_workspace"`
}

type ProfileProviderCfg struct {
	Enabled       bool     `toml:"enabled" yaml:"enabled"`
	Credential    string   `toml:"credential" yaml:"credential"`
	DefaultModel  string   `toml:"default_model" yaml:"default_model"`
	AllowedModels []string `toml:"allowed_models" yaml:"allowed_models"`
	Endpoint      string   `toml:"endpoint" yaml:"endpoint"`
}

type ProfileChannelCfg struct {
	Enabled        bool     `toml:"enabled" yaml:"enabled"`
	Credential     string   `toml:"credential" yaml:"credential"`
	AllowedChats   []string `toml:"allowed_chats" yaml:"allowed_chats"`
	AllowedUsers   []string `toml:"allowed_users" yaml:"allowed_users"`
	RequireMention bool     `toml:"require_mention" yaml:"require_mention"`
	ToolProgress   string   `toml:"tool_progress" yaml:"tool_progress"`
	Servers        []string `toml:"servers" yaml:"servers"`
	VoiceProfile   string   `toml:"voice_profile" yaml:"voice_profile"`
}

type ProfileVoiceProfileCfg struct {
	STTProvider    string `toml:"stt_provider" yaml:"stt_provider" json:"stt_provider,omitempty"`
	TTSProvider    string `toml:"tts_provider" yaml:"tts_provider" json:"tts_provider,omitempty"`
	VoiceID        string `toml:"voice_id" yaml:"voice_id" json:"voice_id,omitempty"`
	LanguagePolicy string `toml:"language_policy" yaml:"language_policy" json:"language_policy,omitempty"`
	FallbackVoice  string `toml:"fallback_voice" yaml:"fallback_voice" json:"fallback_voice,omitempty"`
	STTCredential  string `toml:"stt_credential" yaml:"stt_credential" json:"-"`
	TTSCredential  string `toml:"tts_credential" yaml:"tts_credential" json:"-"`
}

type ProfileVoiceProviderMatrix struct {
	STTProviders []string `json:"stt"`
	TTSProviders []string `json:"tts"`
}

type ProfileVoiceProfileValidation struct {
	ProfileID            string                                  `json:"profile_id"`
	VoiceProfile         ProfileVoiceProfileCfg                  `json:"voice_profile"`
	Valid                bool                                    `json:"valid"`
	Errors               []ProfileVoiceProfileFieldError         `json:"errors,omitempty"`
	CredentialStatusRefs map[string]ProfileVoiceCredentialStatus `json:"credential_status_refs,omitempty"`
}

type ProfileVoiceProfileFieldError struct {
	Field   string `json:"field"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

type ProfileVoiceCredentialStatus struct {
	Configured bool   `json:"configured"`
	Required   bool   `json:"required"`
	Status     string `json:"status"`
	Source     string `json:"source,omitempty"`
}

type CredentialCfg struct {
	Kind         string     `toml:"kind" yaml:"kind"`
	Provider     string     `toml:"provider" yaml:"provider"`
	Channel      string     `toml:"channel" yaml:"channel"`
	OwnerProfile string     `toml:"owner_profile" yaml:"owner_profile"`
	SecretRef    *SecretRef `toml:"secret_ref" yaml:"secret_ref"`
}

type ProfileService struct {
	ID      string
	Profile ProfileCfg
}

type NavivoxProfileRoutingReport struct {
	Profiles []NavivoxProfileRoute `json:"profiles,omitempty"`
	Servers  []NavivoxServerRoute  `json:"servers,omitempty"`
}

type NavivoxServerRoute struct {
	ServerID     string                       `json:"server_id"`
	Bind         string                       `json:"bind,omitempty"`
	Transports   []string                     `json:"transports,omitempty"`
	Capabilities []string                     `json:"capabilities,omitempty"`
	Profiles     []NavivoxProfileRoute        `json:"profiles,omitempty"`
	Warnings     []NavivoxProfileRouteWarning `json:"warnings,omitempty"`
}

type NavivoxProfileRoute struct {
	ProfileID              string                       `json:"profile_id"`
	DisplayName            string                       `json:"display_name,omitempty"`
	Workspaces             []string                     `json:"workspaces,omitempty"`
	Providers              []string                     `json:"providers,omitempty"`
	Channels               []string                     `json:"channels,omitempty"`
	ServerIDs              []string                     `json:"server_ids,omitempty"`
	CredentialConfigured   bool                         `json:"credential_configured,omitempty"`
	VoiceProfileConfigured bool                         `json:"voice_profile_configured,omitempty"`
	Ready                  bool                         `json:"ready,omitempty"`
	Warnings               []NavivoxProfileRouteWarning `json:"warnings,omitempty"`
}

type NavivoxProfileRouteWarning struct {
	Code      string `json:"code"`
	ProfileID string `json:"profile_id,omitempty"`
	Message   string `json:"message,omitempty"`
}

func (c Config) ProfileConfigV2Available() bool {
	return len(c.Profiles) > 0
}

func (c Config) EnabledProfileServices() []ProfileService {
	ids := make([]string, 0, len(c.Profiles))
	for id, profile := range c.Profiles {
		if profile.Enabled {
			ids = append(ids, id)
		}
	}
	sort.Strings(ids)
	out := make([]ProfileService, 0, len(ids))
	for _, id := range ids {
		out = append(out, ProfileService{ID: id, Profile: c.Profiles[id]})
	}
	return out
}

func (c Config) NavivoxProfileRouting() NavivoxProfileRoutingReport {
	if len(c.Navivox.Servers) > 0 {
		return c.navivoxServerScopedProfileRouting()
	}
	services := c.EnabledProfileServices()
	routes := make([]NavivoxProfileRoute, 0, len(services))
	for _, service := range services {
		routes = append(routes, navivoxProfileRouteFromProfile(service.ID, service.Profile))
	}
	return NavivoxProfileRoutingReport{Profiles: routes}
}

func (c Config) navivoxServerScopedProfileRouting() NavivoxProfileRoutingReport {
	serverIDs := profileConfigV2SortedKeys(c.Navivox.Servers)
	servers := make([]NavivoxServerRoute, 0, len(serverIDs))
	routedProfiles := map[string]NavivoxProfileRoute{}
	for _, serverID := range serverIDs {
		server := c.Navivox.Servers[serverID]
		if !server.Enabled {
			continue
		}
		serverRoute := NavivoxServerRoute{
			ServerID:     serverID,
			Bind:         strings.TrimSpace(server.Bind),
			Transports:   navivoxRoutingStrings(server.Transports),
			Capabilities: navivoxRoutingStrings(server.Capabilities),
		}
		for _, profileID := range server.Profiles {
			profile, ok := c.Profiles[profileID]
			if !ok {
				serverRoute.Warnings = append(serverRoute.Warnings, navivoxProfileUnavailableWarning(profileID, "profile is not configured"))
				continue
			}
			if !profile.Enabled {
				serverRoute.Warnings = append(serverRoute.Warnings, navivoxProfileUnavailableWarning(profileID, "profile is disabled"))
				continue
			}
			navivoxChannel, ok := profile.Channels["navivox"]
			if !ok || !navivoxChannel.Enabled || !navivoxChannelReferencesServer(navivoxChannel, serverID) {
				serverRoute.Warnings = append(serverRoute.Warnings, navivoxProfileUnavailableWarning(profileID, "profile is not opted into this Navivox server"))
				continue
			}
			route := navivoxProfileRouteFromProfile(profileID, profile)
			route.ServerIDs = []string{serverID}
			route.CredentialConfigured = strings.TrimSpace(navivoxChannel.Credential) != ""
			route.VoiceProfileConfigured = strings.TrimSpace(navivoxChannel.VoiceProfile) != ""
			route.Ready = true
			serverRoute.Profiles = append(serverRoute.Profiles, route)

			union := routedProfiles[profileID]
			if union.ProfileID == "" {
				union = navivoxProfileRouteFromProfile(profileID, profile)
			}
			union.ServerIDs = appendNavivoxRoutingString(union.ServerIDs, serverID)
			union.CredentialConfigured = union.CredentialConfigured || route.CredentialConfigured
			union.VoiceProfileConfigured = union.VoiceProfileConfigured || route.VoiceProfileConfigured
			union.Ready = union.Ready || route.Ready
			routedProfiles[profileID] = union
		}
		servers = append(servers, serverRoute)
	}

	profileIDs := profileConfigV2SortedKeys(routedProfiles)
	profiles := make([]NavivoxProfileRoute, 0, len(profileIDs))
	for _, profileID := range profileIDs {
		profiles = append(profiles, routedProfiles[profileID])
	}
	return NavivoxProfileRoutingReport{Profiles: profiles, Servers: servers}
}

func navivoxProfileRouteFromProfile(profileID string, profile ProfileCfg) NavivoxProfileRoute {
	displayName := strings.TrimSpace(profile.Name)
	if displayName == "" {
		displayName = profileID
	}
	return NavivoxProfileRoute{
		ProfileID:   profileID,
		DisplayName: displayName,
		Workspaces:  navivoxRoutingStrings(profile.Workspaces),
		Providers:   navivoxRoutingProviderIDs(profile.Providers),
		Channels:    navivoxRoutingChannelIDs(profile.Channels),
	}
}

func navivoxChannelReferencesServer(channel ProfileChannelCfg, serverID string) bool {
	serverID = strings.TrimSpace(serverID)
	if serverID == "" {
		return false
	}
	for _, candidate := range channel.Servers {
		if strings.EqualFold(strings.TrimSpace(candidate), serverID) {
			return true
		}
	}
	return false
}

func navivoxProfileUnavailableWarning(profileID, message string) NavivoxProfileRouteWarning {
	return NavivoxProfileRouteWarning{Code: "navivox_profile_unavailable", ProfileID: profileID, Message: message}
}

func appendNavivoxRoutingString(values []string, value string) []string {
	value = strings.TrimSpace(value)
	if value == "" {
		return values
	}
	for _, existing := range values {
		if existing == value {
			return values
		}
	}
	return append(values, value)
}

func navivoxRoutingProviderIDs(providers map[string]ProfileProviderCfg) []string {
	ids := make([]string, 0, len(providers))
	for id, provider := range providers {
		id = strings.ToLower(strings.TrimSpace(id))
		if id != "" && provider.Enabled {
			ids = append(ids, id)
		}
	}
	sort.Strings(ids)
	return navivoxRoutingStrings(ids)
}

func navivoxRoutingChannelIDs(channels map[string]ProfileChannelCfg) []string {
	ids := make([]string, 0, len(channels))
	for id, channel := range channels {
		id = strings.ToLower(strings.TrimSpace(id))
		if id != "" && channel.Enabled {
			ids = append(ids, id)
		}
	}
	sort.Strings(ids)
	return navivoxRoutingStrings(ids)
}

func navivoxRoutingStrings(values []string) []string {
	values = cleanStringSlice(values)
	if len(values) == 0 {
		return nil
	}
	return values
}

// WriteProfileConfigV2 writes the canonical profile-service and credential
// registry portions of cfg into path as one root TOML transaction. Existing
// non-profile sections are preserved in the root document; profile data never
// lands in per-profile config.toml files and raw secret values are represented
// only as SecretRef metadata.
func WriteProfileConfigV2(path string, cfg Config) error {
	cfg.ConfigVersion = CurrentConfigVersion
	if err := normalizeProfileConfigV2(&cfg); err != nil {
		return err
	}
	doc, err := readTOMLDoc(path)
	if err != nil {
		return err
	}
	doc["config_version"] = int64(CurrentConfigVersion)
	if len(cfg.Profiles) == 0 {
		delete(doc, "profiles")
	} else {
		doc["profiles"] = profileConfigV2ProfilesDocument(cfg.Profiles)
	}
	if len(cfg.Credentials) == 0 {
		delete(doc, "credentials")
	} else {
		doc["credentials"] = profileConfigV2CredentialsDocument(cfg.Credentials)
	}
	return writeTOMLDoc(path, doc)
}

func profileConfigV2ProfilesDocument(profiles map[string]ProfileCfg) map[string]any {
	out := map[string]any{}
	for _, id := range profileConfigV2SortedKeys(profiles) {
		profile := profiles[id]
		entry := map[string]any{
			"enabled": profile.Enabled,
			"name":    profile.Name,
		}
		if profile.Description != "" {
			entry["description"] = profile.Description
		}
		if len(profile.Workspaces) > 0 {
			entry["workspaces"] = append([]string(nil), profile.Workspaces...)
		}
		if len(profile.Tags) > 0 {
			entry["tags"] = append([]string(nil), profile.Tags...)
		}
		if len(profile.Settings) > 0 {
			entry["settings"] = profile.Settings
		}
		if runtime := profileConfigV2RuntimeDocument(profile.Runtime); len(runtime) > 0 {
			entry["runtime"] = runtime
		}
		if voice := profileConfigV2VoiceProfileDocument(profile.VoiceProfile); len(voice) > 0 {
			entry["voice_profile"] = voice
		}
		if providers := profileConfigV2ProvidersDocument(profile.Providers); len(providers) > 0 {
			entry["providers"] = providers
		}
		if channels := profileConfigV2ChannelsDocument(profile.Channels); len(channels) > 0 {
			entry["channels"] = channels
		}
		out[id] = entry
	}
	return out
}

func profileConfigV2RuntimeDocument(runtime ProfileRuntimeCfg) map[string]any {
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

func profileConfigV2VoiceProfileDocument(voice ProfileVoiceProfileCfg) map[string]any {
	voice = normalizeProfileVoiceProfile(voice)
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

func profileConfigV2ProvidersDocument(providers map[string]ProfileProviderCfg) map[string]any {
	out := map[string]any{}
	for _, id := range profileConfigV2SortedKeys(providers) {
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

func profileConfigV2ChannelsDocument(channels map[string]ProfileChannelCfg) map[string]any {
	out := map[string]any{}
	for _, id := range profileConfigV2SortedKeys(channels) {
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

func profileConfigV2CredentialsDocument(credentials map[string]CredentialCfg) map[string]any {
	out := map[string]any{}
	for _, id := range profileConfigV2SortedKeys(credentials) {
		credential := credentials[id]
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
			entry["secret_ref"] = profileConfigV2SecretRefDocument(*credential.SecretRef)
		}
		out[id] = entry
	}
	return out
}

func profileConfigV2SecretRefDocument(ref SecretRef) map[string]any {
	out := map[string]any{"source": string(ref.Source), "id": ref.ID}
	if ref.Provider != "" {
		out["provider"] = ref.Provider
	}
	return out
}

func profileConfigV2SortedKeys[T any](values map[string]T) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func normalizeProfileConfigV2(cfg *Config) error {
	if len(cfg.Profiles) == 0 {
		cfg.Profiles = nil
	} else {
		profiles := make(map[string]ProfileCfg, len(cfg.Profiles))
		for id, profile := range cfg.Profiles {
			normalizedID := strings.TrimSpace(id)
			if normalizedID != id || !agentIDPattern.MatchString(normalizedID) {
				return fmt.Errorf("config: profile id %q is invalid", id)
			}
			profile.Name = strings.TrimSpace(profile.Name)
			profile.Description = strings.TrimSpace(profile.Description)
			profile.Workspaces = cleanStringSlice(profile.Workspaces)
			profile.Tags = cleanStringSlice(profile.Tags)
			profile.Runtime.SessionResetPolicy = strings.TrimSpace(profile.Runtime.SessionResetPolicy)
			profile.Runtime.GonchoWorkspace = strings.TrimSpace(profile.Runtime.GonchoWorkspace)
			profile.Providers = normalizeProfileProviders(profile.Providers)
			profile.Channels = normalizeProfileChannels(profile.Channels)
			profile.VoiceProfile = normalizeProfileVoiceProfile(profile.VoiceProfile)
			profiles[normalizedID] = profile
		}
		cfg.Profiles = profiles
	}

	if len(cfg.Credentials) == 0 {
		cfg.Credentials = nil
		return nil
	}
	credentials := make(map[string]CredentialCfg, len(cfg.Credentials))
	for id, credential := range cfg.Credentials {
		normalizedID := strings.TrimSpace(id)
		if normalizedID != id || !agentIDPattern.MatchString(normalizedID) {
			return fmt.Errorf("config: credential id %q is invalid", id)
		}
		credential.Kind = strings.ToLower(strings.TrimSpace(credential.Kind))
		credential.Provider = strings.ToLower(strings.TrimSpace(credential.Provider))
		credential.Channel = strings.ToLower(strings.TrimSpace(credential.Channel))
		credential.OwnerProfile = strings.TrimSpace(credential.OwnerProfile)
		if credential.OwnerProfile != "" {
			if !agentIDPattern.MatchString(credential.OwnerProfile) {
				return fmt.Errorf("config: credentials.%s.owner_profile %q is invalid", normalizedID, credential.OwnerProfile)
			}
			if len(cfg.Profiles) > 0 {
				if _, ok := cfg.Profiles[credential.OwnerProfile]; !ok {
					return fmt.Errorf("config: credentials.%s.owner_profile %q does not match a configured profile", normalizedID, credential.OwnerProfile)
				}
			}
		}
		if credential.SecretRef != nil {
			ref := normalizeSecretRef(*credential.SecretRef)
			switch ref.Source {
			case SecretRefSourceEnv, SecretRefSourceFile, SecretRefSourceExec:
			default:
				return fmt.Errorf("config: credentials.%s.secret_ref.source %q is invalid", normalizedID, ref.Source)
			}
			if strings.TrimSpace(ref.ID) == "" {
				return fmt.Errorf("config: credentials.%s.secret_ref.id is required", normalizedID)
			}
			credential.SecretRef = &ref
		}
		credentials[normalizedID] = credential
	}
	cfg.Credentials = credentials
	return nil
}

func normalizeProfileVoiceProfile(voice ProfileVoiceProfileCfg) ProfileVoiceProfileCfg {
	voice.STTProvider = strings.ToLower(strings.TrimSpace(voice.STTProvider))
	voice.TTSProvider = strings.ToLower(strings.TrimSpace(voice.TTSProvider))
	voice.VoiceID = strings.TrimSpace(voice.VoiceID)
	voice.LanguagePolicy = strings.ToLower(strings.TrimSpace(voice.LanguagePolicy))
	voice.FallbackVoice = strings.TrimSpace(voice.FallbackVoice)
	voice.STTCredential = strings.TrimSpace(voice.STTCredential)
	voice.TTSCredential = strings.TrimSpace(voice.TTSCredential)
	return voice
}

func ValidateProfileVoiceProfile(profileID string, voice ProfileVoiceProfileCfg, matrix ProfileVoiceProviderMatrix) ProfileVoiceProfileValidation {
	voice = normalizeProfileVoiceProfile(voice)
	validation := ProfileVoiceProfileValidation{
		ProfileID:            strings.TrimSpace(profileID),
		VoiceProfile:         voice,
		Valid:                true,
		CredentialStatusRefs: map[string]ProfileVoiceCredentialStatus{},
	}
	sttProviders := normalizedProviderSet(matrix.STTProviders)
	ttsProviders := normalizedProviderSet(matrix.TTSProviders)
	if voice.STTProvider != "" && !sttProviders[voice.STTProvider] {
		validation.Errors = append(validation.Errors, ProfileVoiceProfileFieldError{Field: "stt_provider", Code: "unknown_provider", Message: fmt.Sprintf("unknown STT provider %q", voice.STTProvider)})
	}
	if voice.TTSProvider != "" && !ttsProviders[voice.TTSProvider] {
		validation.Errors = append(validation.Errors, ProfileVoiceProfileFieldError{Field: "tts_provider", Code: "unknown_provider", Message: fmt.Sprintf("unknown TTS provider %q", voice.TTSProvider)})
	}
	validation.CredentialStatusRefs["stt"] = profileVoiceCredentialStatus("stt", voice.STTProvider, voice.STTCredential)
	validation.CredentialStatusRefs["tts"] = profileVoiceCredentialStatus("tts", voice.TTSProvider, voice.TTSCredential)
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

func profileVoiceCredentialStatus(kind, provider, credential string) ProfileVoiceCredentialStatus {
	provider = strings.ToLower(strings.TrimSpace(provider))
	credential = strings.TrimSpace(credential)
	required := profileVoiceProviderRequiresCredential(kind, provider)
	switch {
	case provider == "":
		return ProfileVoiceCredentialStatus{Configured: false, Required: false, Status: "unset"}
	case credential != "":
		return ProfileVoiceCredentialStatus{Configured: true, Required: required, Status: "configured", Source: "profile_voice_profile." + kind + "_credential"}
	case !required:
		return ProfileVoiceCredentialStatus{Configured: true, Required: false, Status: "not_required", Source: provider}
	default:
		return ProfileVoiceCredentialStatus{Configured: false, Required: true, Status: "missing"}
	}
}

func profileVoiceProviderRequiresCredential(kind, provider string) bool {
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
	case "piper", "neutts", "kittentts", "local", "text_only":
		return false
	default:
		return true
	}
}

func normalizeProfileProviders(in map[string]ProfileProviderCfg) map[string]ProfileProviderCfg {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]ProfileProviderCfg, len(in))
	for provider, cfg := range in {
		key := strings.ToLower(strings.TrimSpace(provider))
		if key == "" {
			continue
		}
		cfg.Credential = strings.TrimSpace(cfg.Credential)
		cfg.DefaultModel = strings.TrimSpace(cfg.DefaultModel)
		cfg.AllowedModels = cleanStringSlice(cfg.AllowedModels)
		cfg.Endpoint = strings.TrimSpace(cfg.Endpoint)
		out[key] = cfg
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func normalizeProfileChannels(in map[string]ProfileChannelCfg) map[string]ProfileChannelCfg {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]ProfileChannelCfg, len(in))
	for channel, cfg := range in {
		key := strings.ToLower(strings.TrimSpace(channel))
		if key == "" {
			continue
		}
		cfg.Credential = strings.TrimSpace(cfg.Credential)
		cfg.AllowedChats = cleanStringSlice(cfg.AllowedChats)
		cfg.AllowedUsers = cleanStringSlice(cfg.AllowedUsers)
		cfg.ToolProgress = strings.ToLower(strings.TrimSpace(cfg.ToolProgress))
		cfg.Servers = cleanStringSlice(cfg.Servers)
		cfg.VoiceProfile = strings.TrimSpace(cfg.VoiceProfile)
		out[key] = cfg
	}
	if len(out) == 0 {
		return nil
	}
	return out
}
