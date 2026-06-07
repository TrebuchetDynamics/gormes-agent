package profilechanneltest

import (
	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway/tokenlock"
)

func ChannelCredential(channel, ownerProfile, envID string) config.CredentialCfg {
	return config.CredentialCfg{
		Kind:         "channel",
		Channel:      channel,
		OwnerProfile: ownerProfile,
		SecretRef: &config.SecretRef{
			Source: config.SecretRefSourceEnv,
			ID:     envID,
		},
	}
}

func TokenCredentialHash(credential string) string {
	return tokenlock.TokenCredentialHash(credential)
}
