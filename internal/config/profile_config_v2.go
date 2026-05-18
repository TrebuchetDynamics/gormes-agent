package config

import (
	"fmt"
	"sort"
	"strings"
)

const DefaultProfileID = "main"

type ProfileCfg struct {
	Enabled     bool                          `toml:"enabled" yaml:"enabled"`
	Name        string                        `toml:"name" yaml:"name"`
	Description string                        `toml:"description" yaml:"description"`
	Workspaces  []string                      `toml:"workspaces" yaml:"workspaces"`
	Tags        []string                      `toml:"tags" yaml:"tags"`
	Settings    map[string]any                `toml:"settings" yaml:"settings"`
	Runtime     ProfileRuntimeCfg             `toml:"runtime" yaml:"runtime"`
	Providers   map[string]ProfileProviderCfg `toml:"providers" yaml:"providers"`
	Channels    map[string]ProfileChannelCfg  `toml:"channels" yaml:"channels"`
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

func DefaultConfigDocumentV2() map[string]any {
	return map[string]any{
		"config_version": int64(CurrentConfigVersion),
		"profiles": map[string]any{
			DefaultProfileID: map[string]any{
				"enabled": true,
				"name":    "",
			},
		},
	}
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
