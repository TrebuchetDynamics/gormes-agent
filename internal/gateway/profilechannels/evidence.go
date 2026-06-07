package profilechannels

import "github.com/TrebuchetDynamics/gormes-agent/internal/gateway/profilechannels/contracts"

func HasEvidenceCode(items []ProfileChannelReadinessEvidence, code string) bool {
	return contracts.HasEvidenceCode(items, code)
}
