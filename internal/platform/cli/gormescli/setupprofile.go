package gormescli

import (
	"github.com/TrebuchetDynamics/gormes-agent/internal/app/setupprofile"
	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
)

type SetupGatewayProfileChannelBinding = setupprofile.ChannelBinding

type SetupGatewayProfileChannelOptions = setupprofile.ChannelOptions

func SetupProfileID(home, defaultProfileID string) string {
	return setupprofile.ProfileID(home, defaultProfileID)
}

func SetupProfileRegistryPath(baseHome string) string {
	return setupprofile.RegistryPath(baseHome)
}

func SetupProfileCredentialID(profileID, channelID string) string {
	return setupprofile.CredentialID(profileID, channelID)
}

func SetupGatewayProfileChannelPreview(channelID string) SetupGatewayProfileChannelBinding {
	return setupprofile.ChannelBindingPreview(config.GormesHome(), config.GormesBaseHome(), config.DefaultProfileID, channelID)
}

func WriteSetupGatewayProfileChannelBinding(opts SetupGatewayProfileChannelOptions) (SetupGatewayProfileChannelBinding, error) {
	return setupprofile.WriteChannelBinding(config.GormesHome(), config.GormesBaseHome(), config.DefaultProfileID, opts)
}

func WriteSetupGatewayProfileChannelCredential(channelID string) (SetupGatewayProfileChannelBinding, error) {
	return setupprofile.WriteChannelCredential(config.GormesHome(), config.GormesBaseHome(), config.DefaultProfileID, channelID)
}

func LoadSetupGatewayProfileRegistry(path string) (config.Config, error) {
	return setupprofile.LoadRegistry(path)
}

func WriteSetupGatewayTokenEnv(binding SetupGatewayProfileChannelBinding, legacyEnvName, token string) error {
	return setupprofile.WriteTokenEnv(config.EnvPath(), binding, legacyEnvName, token)
}

func WriteSetupGatewayRuntimeSecretRef(key, envName string) error {
	return setupprofile.WriteRuntimeSecretRef(config.ConfigPath(), key, envName)
}

func CompactSetupProfileStrings(values []string) []string {
	return setupprofile.CompactStrings(values)
}

func SetupInt64Strings(values []int64) []string {
	return setupprofile.Int64Strings(values)
}
