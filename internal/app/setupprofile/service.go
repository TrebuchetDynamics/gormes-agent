package setupprofile

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/pelletier/go-toml/v2"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
)

// ProfileID resolves the active setup profile ID from a GORMES_HOME-like path.
func ProfileID(home, defaultProfileID string) string {
	home = filepath.Clean(strings.TrimSpace(home))
	if home != "" && filepath.Base(filepath.Dir(home)) == "profiles" {
		if name := strings.ToLower(strings.TrimSpace(filepath.Base(home))); name != "" {
			return name
		}
	}
	return defaultProfileID
}

// RegistryPath returns the shared profile registry path under the base home.
func RegistryPath(baseHome string) string {
	return filepath.Join(baseHome, "config.toml")
}

// CredentialID returns the channel credential ID for a profile/channel pair.
func CredentialID(profileID, channelID string) string {
	profileID = strings.ToLower(strings.TrimSpace(profileID))
	channelID = strings.ToLower(strings.TrimSpace(channelID))
	channelID = strings.NewReplacer(".", "_", "/", "_", " ", "_").Replace(channelID)
	return profileID + "-" + channelID
}

// CompactStrings trims, drops blanks, and de-duplicates values while preserving order.
func CompactStrings(values []string) []string {
	out := make([]string, 0, len(values))
	seen := map[string]struct{}{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}

// Int64Strings formats int64 identifiers for profile channel allow-lists.
func Int64Strings(values []int64) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		out = append(out, fmt.Sprintf("%d", value))
	}
	return out
}

type ChannelBinding struct {
	ProfileID     string
	ChannelID     string
	CredentialID  string
	SecretEnvName string
	RegistryPath  string
}

type ChannelOptions struct {
	ChannelID      string
	AllowedChats   []string
	AllowedUsers   []string
	RequireMention bool
	ToolProgress   string
}

func ChannelBindingPreview(home, baseHome, defaultProfileID, channelID string) ChannelBinding {
	channelID = strings.ToLower(strings.TrimSpace(channelID))
	profileID := ProfileID(home, defaultProfileID)
	return ChannelBinding{
		ProfileID:     profileID,
		ChannelID:     channelID,
		CredentialID:  CredentialID(profileID, channelID),
		SecretEnvName: ChannelEnv(profileID, channelID),
		RegistryPath:  RegistryPath(baseHome),
	}
}

func WriteChannelBinding(home, baseHome, defaultProfileID string, opts ChannelOptions) (ChannelBinding, error) {
	channelID := strings.ToLower(strings.TrimSpace(opts.ChannelID))
	if channelID == "" {
		return ChannelBinding{}, fmt.Errorf("setup gateway profile channel: channel id is required")
	}
	preview := ChannelBindingPreview(home, baseHome, defaultProfileID, channelID)
	cfg, err := LoadRegistry(preview.RegistryPath)
	if err != nil {
		return ChannelBinding{}, err
	}
	if cfg.Profiles == nil {
		cfg.Profiles = map[string]config.ProfileCfg{}
	}
	profile := cfg.Profiles[preview.ProfileID]
	profile.Enabled = true
	if profile.Channels == nil {
		profile.Channels = map[string]config.ProfileChannelCfg{}
	}
	channelCfg := profile.Channels[channelID]
	channelCfg.Enabled = true
	channelCfg.Credential = preview.CredentialID
	if opts.AllowedChats != nil {
		channelCfg.AllowedChats = CompactStrings(opts.AllowedChats)
	}
	if opts.AllowedUsers != nil {
		channelCfg.AllowedUsers = CompactStrings(opts.AllowedUsers)
	}
	if opts.RequireMention {
		channelCfg.RequireMention = true
	}
	if strings.TrimSpace(opts.ToolProgress) != "" {
		channelCfg.ToolProgress = strings.TrimSpace(opts.ToolProgress)
	}
	profile.Channels[channelID] = channelCfg
	cfg.Profiles[preview.ProfileID] = profile

	if cfg.Credentials == nil {
		cfg.Credentials = map[string]config.CredentialCfg{}
	}
	cfg.Credentials[preview.CredentialID] = ChannelCredential(preview.ProfileID, channelID)
	if err := config.WriteProfileConfigV2(preview.RegistryPath, cfg); err != nil {
		return ChannelBinding{}, err
	}
	return preview, nil
}

func WriteChannelCredential(home, baseHome, defaultProfileID, channelID string) (ChannelBinding, error) {
	channelID = strings.ToLower(strings.TrimSpace(channelID))
	preview := ChannelBindingPreview(home, baseHome, defaultProfileID, channelID)
	cfg, err := LoadRegistry(preview.RegistryPath)
	if err != nil {
		return ChannelBinding{}, err
	}
	if cfg.Credentials == nil {
		cfg.Credentials = map[string]config.CredentialCfg{}
	}
	cfg.Credentials[preview.CredentialID] = ChannelCredential(preview.ProfileID, channelID)
	if err := config.WriteProfileConfigV2(preview.RegistryPath, cfg); err != nil {
		return ChannelBinding{}, err
	}
	return preview, nil
}

func LoadRegistry(path string) (config.Config, error) {
	var cfg config.Config
	body, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return cfg, fmt.Errorf("setup gateway profile registry: read %s: %w", path, err)
	}
	if err := toml.Unmarshal(body, &cfg); err != nil {
		return cfg, fmt.Errorf("setup gateway profile registry: parse %s: %w", path, err)
	}
	return cfg, nil
}

func WriteTokenEnv(envPath string, binding ChannelBinding, legacyEnvName, token string) error {
	token = strings.TrimSpace(token)
	if token == "" {
		return nil
	}
	if binding.SecretEnvName != "" {
		if err := config.WriteEnvValue(envPath, binding.SecretEnvName, token); err != nil {
			return err
		}
		if err := os.Setenv(binding.SecretEnvName, token); err != nil {
			return err
		}
	}
	legacyEnvName = strings.TrimSpace(legacyEnvName)
	if legacyEnvName != "" && legacyEnvName != binding.SecretEnvName {
		if err := config.WriteEnvValue(envPath, legacyEnvName, token); err != nil {
			return err
		}
		if err := os.Setenv(legacyEnvName, token); err != nil {
			return err
		}
	}
	return nil
}

func WriteRuntimeSecretRef(configPath, key, envName string) error {
	envName = strings.TrimSpace(envName)
	if envName == "" {
		return nil
	}
	if err := config.WriteTOMLValue(configPath, key+".source", string(config.SecretRefSourceEnv)); err != nil {
		return err
	}
	if err := config.WriteTOMLValue(configPath, key+".id", envName); err != nil {
		return err
	}
	return nil
}

func ChannelCredential(profileID, channelID string) config.CredentialCfg {
	profileID = strings.ToLower(strings.TrimSpace(profileID))
	channelID = strings.ToLower(strings.TrimSpace(channelID))
	return config.CredentialCfg{
		Kind:         "channel",
		Channel:      channelID,
		OwnerProfile: profileID,
		SecretRef:    &config.SecretRef{Source: config.SecretRefSourceEnv, ID: ChannelEnv(profileID, channelID)},
	}
}

func ChannelEnv(profileID, channelID string) string {
	prefix := "GORMES_" + EnvPart(profileID) + "_"
	if strings.EqualFold(channelID, "telegram") {
		return prefix + "TELEGRAM_BOT_TOKEN"
	}
	if strings.EqualFold(channelID, "slack_app") {
		return prefix + "SLACK_APP_TOKEN"
	}
	return prefix + EnvPart(channelID) + "_TOKEN"
}

func EnvPart(value string) string {
	value = strings.ToUpper(strings.TrimSpace(value))
	return strings.NewReplacer("-", "_", ".", "_", "/", "_").Replace(value)
}
