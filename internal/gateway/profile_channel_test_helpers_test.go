package gateway

import (
	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway/profilechannels"
	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway/profilechanneltest"
)

func channelCredential(channel, ownerProfile, envID string) config.CredentialCfg {
	return profilechanneltest.ChannelCredential(channel, ownerProfile, envID)
}

func hasProfileChannelEvidence(items []ProfileChannelReadinessEvidence, code string) bool {
	return profilechannels.HasEvidenceCode(items, code)
}
