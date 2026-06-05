package gateway

import (
	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway/profilechannels"
)

const (
	ProfileChannelEvidenceCredentialMissing         = profilechannels.ProfileChannelEvidenceCredentialMissing
	ProfileChannelEvidenceCredentialKindMismatch    = profilechannels.ProfileChannelEvidenceCredentialKindMismatch
	ProfileChannelEvidenceCredentialChannelMismatch = profilechannels.ProfileChannelEvidenceCredentialChannelMismatch
	ProfileChannelEvidenceCredentialOwnerMismatch   = profilechannels.ProfileChannelEvidenceCredentialOwnerMismatch
	ProfileChannelEvidenceCredentialSecretMissing   = profilechannels.ProfileChannelEvidenceCredentialSecretMissing
	ProfileChannelEvidenceCredentialHashUnavailable = profilechannels.ProfileChannelEvidenceCredentialHashUnavailable
	ProfileChannelEvidenceAccessPolicyMissing       = profilechannels.ProfileChannelEvidenceAccessPolicyMissing
	ProfileChannelEvidenceTokenHashConflict         = profilechannels.ProfileChannelEvidenceTokenHashConflict
)

type ProfileChannelReadinessReport = profilechannels.ProfileChannelReadinessReport
type ProfileChannelReadinessOptions = profilechannels.ProfileChannelReadinessOptions
type ProfileChannelBindingReadiness = profilechannels.ProfileChannelBindingReadiness
type ProfileChannelReadinessEvidence = profilechannels.ProfileChannelReadinessEvidence

func BuildProfileChannelReadiness(cfg config.Config) ProfileChannelReadinessReport {
	return profilechannels.BuildProfileChannelReadiness(cfg)
}

func BuildProfileChannelReadinessWithOptions(cfg config.Config, opts ProfileChannelReadinessOptions) ProfileChannelReadinessReport {
	return profilechannels.BuildProfileChannelReadinessWithOptions(cfg, opts)
}

func normalizedProfileChannelCredentialHashes(in map[string]string) map[string]string {
	return profilechannels.NormalizedCredentialHashes(in)
}

func sortedProfileChannelConfigBindings(channels map[string]config.ProfileChannelCfg) []profilechannels.ConfigBinding {
	return profilechannels.SortedConfigBindings(channels)
}

func collectProfileChannelReadinessEvidence(bindings []ProfileChannelBindingReadiness) []ProfileChannelReadinessEvidence {
	return profilechannels.CollectReadinessEvidence(bindings)
}

func newProfileChannelEvidence(code, profileID, channel, credentialID, field, message string) ProfileChannelReadinessEvidence {
	return profilechannels.NewEvidence(code, profileID, channel, credentialID, field, message)
}
