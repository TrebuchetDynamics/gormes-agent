package gateway

import "github.com/TrebuchetDynamics/gormes-agent/internal/config"

func channelCredential(channel, ownerProfile, envID string) config.CredentialCfg {
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

func hasProfileChannelEvidence(items []ProfileChannelReadinessEvidence, code string) bool {
	for _, item := range items {
		if item.Code == code {
			return true
		}
	}
	return false
}
