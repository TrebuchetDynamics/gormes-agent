package profile

import (
	"github.com/TrebuchetDynamics/gormes-agent/internal/config/credentials"
	"github.com/TrebuchetDynamics/gormes-agent/internal/config/profilestorage"
)

const DefaultID = profilestorage.DefaultProfileID

type Config struct {
	Enabled      bool                      `toml:"enabled" yaml:"enabled"`
	Name         string                    `toml:"name" yaml:"name"`
	Description  string                    `toml:"description" yaml:"description"`
	Workspaces   []string                  `toml:"workspaces" yaml:"workspaces"`
	Tags         []string                  `toml:"tags" yaml:"tags"`
	Settings     map[string]any            `toml:"settings" yaml:"settings"`
	Runtime      RuntimeConfig             `toml:"runtime" yaml:"runtime"`
	Providers    map[string]ProviderConfig `toml:"providers" yaml:"providers"`
	Channels     map[string]ChannelConfig  `toml:"channels" yaml:"channels"`
	VoiceProfile VoiceProfileConfig        `toml:"voice_profile" yaml:"voice_profile" json:"voice_profile,omitempty"`
}

type RuntimeConfig struct {
	MaxTurns              int    `toml:"max_turns" yaml:"max_turns"`
	SessionResetPolicy    string `toml:"session_reset_policy" yaml:"session_reset_policy"`
	SessionResetAfterMins int    `toml:"session_reset_after_minutes" yaml:"session_reset_after_minutes"`
	GonchoWorkspace       string `toml:"goncho_workspace" yaml:"goncho_workspace"`
}

type ProviderConfig struct {
	Enabled       bool     `toml:"enabled" yaml:"enabled"`
	Credential    string   `toml:"credential" yaml:"credential"`
	DefaultModel  string   `toml:"default_model" yaml:"default_model"`
	AllowedModels []string `toml:"allowed_models" yaml:"allowed_models"`
	Endpoint      string   `toml:"endpoint" yaml:"endpoint"`
}

type ChannelConfig struct {
	Enabled        bool     `toml:"enabled" yaml:"enabled"`
	Credential     string   `toml:"credential" yaml:"credential"`
	AllowedChats   []string `toml:"allowed_chats" yaml:"allowed_chats"`
	AllowedUsers   []string `toml:"allowed_users" yaml:"allowed_users"`
	RequireMention bool     `toml:"require_mention" yaml:"require_mention"`
	ToolProgress   string   `toml:"tool_progress" yaml:"tool_progress"`
	Servers        []string `toml:"servers" yaml:"servers"`
	VoiceProfile   string   `toml:"voice_profile" yaml:"voice_profile"`
}

type VoiceProfileConfig struct {
	STTProvider    string `toml:"stt_provider" yaml:"stt_provider" json:"stt_provider,omitempty"`
	TTSProvider    string `toml:"tts_provider" yaml:"tts_provider" json:"tts_provider,omitempty"`
	VoiceID        string `toml:"voice_id" yaml:"voice_id" json:"voice_id,omitempty"`
	LanguagePolicy string `toml:"language_policy" yaml:"language_policy" json:"language_policy,omitempty"`
	FallbackVoice  string `toml:"fallback_voice" yaml:"fallback_voice" json:"fallback_voice,omitempty"`
	STTCredential  string `toml:"stt_credential" yaml:"stt_credential" json:"-"`
	TTSCredential  string `toml:"tts_credential" yaml:"tts_credential" json:"-"`
}

type VoiceProviderMatrix struct {
	STTProviders []string `json:"stt"`
	TTSProviders []string `json:"tts"`
}

type VoiceProfileValidation struct {
	ProfileID            string                           `json:"profile_id"`
	VoiceProfile         VoiceProfileConfig               `json:"voice_profile"`
	Valid                bool                             `json:"valid"`
	Errors               []VoiceProfileFieldError         `json:"errors,omitempty"`
	CredentialStatusRefs map[string]VoiceCredentialStatus `json:"credential_status_refs,omitempty"`
}

type VoiceProfileFieldError struct {
	Field   string `json:"field"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

type VoiceCredentialStatus struct {
	Configured bool   `json:"configured"`
	Required   bool   `json:"required"`
	Status     string `json:"status"`
	Source     string `json:"source,omitempty"`
}

type CredentialConfig struct {
	Kind         string                 `toml:"kind" yaml:"kind"`
	Provider     string                 `toml:"provider" yaml:"provider"`
	Channel      string                 `toml:"channel" yaml:"channel"`
	OwnerProfile string                 `toml:"owner_profile" yaml:"owner_profile"`
	SecretRef    *credentials.SecretRef `toml:"secret_ref" yaml:"secret_ref"`
}

type Service struct {
	ID      string
	Profile Config
}

type NavivoxRoutingReport struct {
	Profiles []NavivoxRoute       `json:"profiles,omitempty"`
	Servers  []NavivoxServerRoute `json:"servers,omitempty"`
}

type NavivoxServerRoute struct {
	ServerID     string                `json:"server_id"`
	Bind         string                `json:"bind,omitempty"`
	Transports   []string              `json:"transports,omitempty"`
	Capabilities []string              `json:"capabilities,omitempty"`
	Profiles     []NavivoxRoute        `json:"profiles,omitempty"`
	Warnings     []NavivoxRouteWarning `json:"warnings,omitempty"`
}

type NavivoxRoute struct {
	ProfileID              string                `json:"profile_id"`
	DisplayName            string                `json:"display_name,omitempty"`
	Workspaces             []string              `json:"workspaces,omitempty"`
	Providers              []string              `json:"providers,omitempty"`
	Channels               []string              `json:"channels,omitempty"`
	ServerIDs              []string              `json:"server_ids,omitempty"`
	CredentialConfigured   bool                  `json:"credential_configured,omitempty"`
	VoiceProfileConfigured bool                  `json:"voice_profile_configured,omitempty"`
	Ready                  bool                  `json:"ready,omitempty"`
	Warnings               []NavivoxRouteWarning `json:"warnings,omitempty"`
}

type NavivoxRouteWarning struct {
	Code      string `json:"code"`
	ProfileID string `json:"profile_id,omitempty"`
	Message   string `json:"message,omitempty"`
}
