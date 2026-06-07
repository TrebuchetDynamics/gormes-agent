package profilechannels

import (
	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway/profilechannels/contracts"
	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway/profilechannels/readiness"
)

const (
	ProfileChannelEvidenceCredentialMissing         = contracts.ProfileChannelEvidenceCredentialMissing
	ProfileChannelEvidenceCredentialKindMismatch    = contracts.ProfileChannelEvidenceCredentialKindMismatch
	ProfileChannelEvidenceCredentialChannelMismatch = contracts.ProfileChannelEvidenceCredentialChannelMismatch
	ProfileChannelEvidenceCredentialOwnerMismatch   = contracts.ProfileChannelEvidenceCredentialOwnerMismatch
	ProfileChannelEvidenceCredentialSecretMissing   = contracts.ProfileChannelEvidenceCredentialSecretMissing
	ProfileChannelEvidenceCredentialHashUnavailable = contracts.ProfileChannelEvidenceCredentialHashUnavailable
	ProfileChannelEvidenceAccessPolicyMissing       = contracts.ProfileChannelEvidenceAccessPolicyMissing
	ProfileChannelEvidenceTokenHashConflict         = contracts.ProfileChannelEvidenceTokenHashConflict
)

type ProfileChannelReadinessReport = contracts.ProfileChannelReadinessReport
type ProfileChannelReadinessOptions = contracts.ProfileChannelReadinessOptions
type ProfileChannelBindingReadiness = contracts.ProfileChannelBindingReadiness
type ProfileChannelReadinessEvidence = contracts.ProfileChannelReadinessEvidence
type ConfigBinding = contracts.ConfigBinding

func BuildProfileChannelReadiness(cfg config.Config) ProfileChannelReadinessReport {
	return readiness.BuildProfileChannelReadiness(cfg)
}

func BuildProfileChannelReadinessWithOptions(cfg config.Config, opts ProfileChannelReadinessOptions) ProfileChannelReadinessReport {
	return readiness.BuildProfileChannelReadinessWithOptions(cfg, opts)
}

func CollectReadinessEvidence(bindings []ProfileChannelBindingReadiness) []ProfileChannelReadinessEvidence {
	return contracts.CollectReadinessEvidence(bindings)
}

func NormalizedCredentialHashes(in map[string]string) map[string]string {
	return readiness.NormalizedCredentialHashes(in)
}

func SortedConfigBindings(channels map[string]config.ProfileChannelCfg) []ConfigBinding {
	return readiness.SortedConfigBindings(channels)
}

func NewEvidence(code, profileID, channel, credentialID, field, message string) ProfileChannelReadinessEvidence {
	return contracts.NewEvidence(code, profileID, channel, credentialID, field, message)
}
