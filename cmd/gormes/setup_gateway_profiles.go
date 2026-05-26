package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/pelletier/go-toml/v2"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
)

type setupGatewayProfileChannelBinding struct {
	ProfileID     string
	ChannelID     string
	CredentialID  string
	SecretEnvName string
	RegistryPath  string
}

type setupGatewayProfileChannelOptions struct {
	ChannelID      string
	AllowedChats   []string
	AllowedUsers   []string
	RequireMention bool
	ToolProgress   string
}

func setupGatewayProfileID() string {
	home := filepath.Clean(strings.TrimSpace(config.GormesHome()))
	if home != "" && filepath.Base(filepath.Dir(home)) == "profiles" {
		if name := strings.TrimSpace(filepath.Base(home)); name != "" {
			return strings.ToLower(name)
		}
	}
	return config.DefaultProfileID
}

func setupGatewayProfileRegistryPath() string {
	return filepath.Join(config.GormesBaseHome(), "config.toml")
}

func setupGatewayProfileCredentialID(profileID, channelID string) string {
	profileID = strings.ToLower(strings.TrimSpace(profileID))
	channelID = strings.ToLower(strings.TrimSpace(channelID))
	channelID = strings.NewReplacer(".", "_", "/", "_", " ", "_").Replace(channelID)
	return profileID + "-" + channelID
}

func setupGatewayProfileChannelPreview(channelID string) setupGatewayProfileChannelBinding {
	channelID = strings.ToLower(strings.TrimSpace(channelID))
	profileID := setupGatewayProfileID()
	return setupGatewayProfileChannelBinding{
		ProfileID:     profileID,
		ChannelID:     channelID,
		CredentialID:  setupGatewayProfileCredentialID(profileID, channelID),
		SecretEnvName: setupProfilesChannelEnv(profileID, channelID),
		RegistryPath:  setupGatewayProfileRegistryPath(),
	}
}

func writeSetupGatewayProfileChannelBinding(opts setupGatewayProfileChannelOptions) (setupGatewayProfileChannelBinding, error) {
	channelID := strings.ToLower(strings.TrimSpace(opts.ChannelID))
	if channelID == "" {
		return setupGatewayProfileChannelBinding{}, fmt.Errorf("setup gateway profile channel: channel id is required")
	}
	preview := setupGatewayProfileChannelPreview(channelID)
	profileID := preview.ProfileID
	credentialID := preview.CredentialID
	secretEnv := preview.SecretEnvName
	registryPath := preview.RegistryPath

	cfg, err := loadSetupGatewayProfileRegistry(registryPath)
	if err != nil {
		return setupGatewayProfileChannelBinding{}, err
	}
	if cfg.Profiles == nil {
		cfg.Profiles = map[string]config.ProfileCfg{}
	}
	profile := cfg.Profiles[profileID]
	profile.Enabled = true
	if profile.Channels == nil {
		profile.Channels = map[string]config.ProfileChannelCfg{}
	}
	channelCfg := profile.Channels[channelID]
	channelCfg.Enabled = true
	channelCfg.Credential = credentialID
	if opts.AllowedChats != nil {
		channelCfg.AllowedChats = compactSetupProfileStrings(opts.AllowedChats)
	}
	if opts.AllowedUsers != nil {
		channelCfg.AllowedUsers = compactSetupProfileStrings(opts.AllowedUsers)
	}
	if opts.RequireMention {
		channelCfg.RequireMention = true
	}
	if strings.TrimSpace(opts.ToolProgress) != "" {
		channelCfg.ToolProgress = strings.TrimSpace(opts.ToolProgress)
	}
	profile.Channels[channelID] = channelCfg
	cfg.Profiles[profileID] = profile

	if cfg.Credentials == nil {
		cfg.Credentials = map[string]config.CredentialCfg{}
	}
	cfg.Credentials[credentialID] = setupProfilesChannelCredential(profileID, channelID)
	if err := config.WriteProfileConfigV2(registryPath, cfg); err != nil {
		return setupGatewayProfileChannelBinding{}, err
	}
	return setupGatewayProfileChannelBinding{ProfileID: profileID, ChannelID: channelID, CredentialID: credentialID, SecretEnvName: secretEnv, RegistryPath: registryPath}, nil
}

func writeSetupGatewayProfileChannelCredential(channelID string) (setupGatewayProfileChannelBinding, error) {
	channelID = strings.ToLower(strings.TrimSpace(channelID))
	profileID := setupGatewayProfileID()
	credentialID := setupGatewayProfileCredentialID(profileID, channelID)
	secretEnv := setupProfilesChannelEnv(profileID, channelID)
	registryPath := setupGatewayProfileRegistryPath()
	cfg, err := loadSetupGatewayProfileRegistry(registryPath)
	if err != nil {
		return setupGatewayProfileChannelBinding{}, err
	}
	if cfg.Credentials == nil {
		cfg.Credentials = map[string]config.CredentialCfg{}
	}
	cfg.Credentials[credentialID] = setupProfilesChannelCredential(profileID, channelID)
	if err := config.WriteProfileConfigV2(registryPath, cfg); err != nil {
		return setupGatewayProfileChannelBinding{}, err
	}
	return setupGatewayProfileChannelBinding{ProfileID: profileID, ChannelID: channelID, CredentialID: credentialID, SecretEnvName: secretEnv, RegistryPath: registryPath}, nil
}

func loadSetupGatewayProfileRegistry(path string) (config.Config, error) {
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

func writeSetupGatewayTokenEnv(binding setupGatewayProfileChannelBinding, legacyEnvName, token string) error {
	token = strings.TrimSpace(token)
	if token == "" {
		return nil
	}
	if binding.SecretEnvName != "" {
		if err := config.WriteEnvValue(config.EnvPath(), binding.SecretEnvName, token); err != nil {
			return err
		}
		if err := os.Setenv(binding.SecretEnvName, token); err != nil {
			return err
		}
	}
	legacyEnvName = strings.TrimSpace(legacyEnvName)
	if legacyEnvName != "" && legacyEnvName != binding.SecretEnvName {
		if err := config.WriteEnvValue(config.EnvPath(), legacyEnvName, token); err != nil {
			return err
		}
		if err := os.Setenv(legacyEnvName, token); err != nil {
			return err
		}
	}
	return nil
}

func writeSetupGatewayRuntimeSecretRef(key, envName string) error {
	envName = strings.TrimSpace(envName)
	if envName == "" {
		return nil
	}
	if err := config.WriteTOMLValue(config.ConfigPath(), key+".source", string(config.SecretRefSourceEnv)); err != nil {
		return err
	}
	if err := config.WriteTOMLValue(config.ConfigPath(), key+".id", envName); err != nil {
		return err
	}
	return nil
}

func compactSetupProfileStrings(values []string) []string {
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

func setupInt64Strings(values []int64) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		out = append(out, fmt.Sprintf("%d", value))
	}
	return out
}
