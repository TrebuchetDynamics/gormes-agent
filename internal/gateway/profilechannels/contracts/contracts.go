package contracts

import "github.com/TrebuchetDynamics/gormes-agent/internal/config"

const (
	ProfileChannelEvidenceCredentialMissing         = "channel_credential_missing"
	ProfileChannelEvidenceCredentialKindMismatch    = "channel_credential_kind_mismatch"
	ProfileChannelEvidenceCredentialChannelMismatch = "channel_credential_channel_mismatch"
	ProfileChannelEvidenceCredentialOwnerMismatch   = "channel_credential_owner_mismatch"
	ProfileChannelEvidenceCredentialSecretMissing   = "channel_credential_secret_missing"
	ProfileChannelEvidenceCredentialHashUnavailable = "channel_credential_hash_unavailable"
	ProfileChannelEvidenceAccessPolicyMissing       = "channel_access_policy_missing"
	ProfileChannelEvidenceTokenHashConflict         = "channel_token_hash_conflict"
)

type ProfileChannelReadinessReport struct {
	Bindings []ProfileChannelBindingReadiness  `json:"bindings"`
	Evidence []ProfileChannelReadinessEvidence `json:"evidence,omitempty"`
}

type ProfileChannelReadinessOptions struct {
	CredentialHashes map[string]string
}

type ProfileChannelBindingReadiness struct {
	ProfileID              string                            `json:"profile_id"`
	Channel                string                            `json:"channel"`
	Ready                  bool                              `json:"ready"`
	CredentialID           string                            `json:"credential_id,omitempty"`
	CredentialOwnerProfile string                            `json:"credential_owner_profile,omitempty"`
	CredentialHash         string                            `json:"credential_hash,omitempty"`
	SecretRefConfigured    bool                              `json:"secret_ref_configured"`
	SecretRefSource        string                            `json:"secret_ref_source,omitempty"`
	AllowedChatCount       int                               `json:"allowed_chat_count"`
	AllowedChatScopeHash   string                            `json:"allowed_chat_scope_hash,omitempty"`
	AllowedDirectChatCount int                               `json:"allowed_direct_chat_count"`
	AllowedGroupChatCount  int                               `json:"allowed_group_chat_count"`
	AllowedUserCount       int                               `json:"allowed_user_count"`
	AllowedUserScopeHash   string                            `json:"allowed_user_scope_hash,omitempty"`
	RequireMention         bool                              `json:"require_mention"`
	ToolProgress           string                            `json:"tool_progress,omitempty"`
	ServerCount            int                               `json:"server_count,omitempty"`
	VoiceProfileSet        bool                              `json:"voice_profile_set,omitempty"`
	Evidence               []ProfileChannelReadinessEvidence `json:"evidence,omitempty"`
}

type ProfileChannelReadinessEvidence struct {
	Code           string `json:"code"`
	ProfileID      string `json:"profile_id,omitempty"`
	Channel        string `json:"channel,omitempty"`
	CredentialID   string `json:"credential_id,omitempty"`
	CredentialHash string `json:"credential_hash,omitempty"`
	Field          string `json:"field,omitempty"`
	Message        string `json:"message,omitempty"`
	Redacted       bool   `json:"redacted"`
}

type ConfigBinding struct {
	Channel string
	Config  config.ProfileChannelCfg
}

func NewEvidence(code, profileID, channel, credentialID, field, message string) ProfileChannelReadinessEvidence {
	return ProfileChannelReadinessEvidence{
		Code:         code,
		ProfileID:    profileID,
		Channel:      channel,
		CredentialID: credentialID,
		Field:        field,
		Message:      message,
		Redacted:     true,
	}
}

func CollectReadinessEvidence(bindings []ProfileChannelBindingReadiness) []ProfileChannelReadinessEvidence {
	var evidence []ProfileChannelReadinessEvidence
	for _, binding := range bindings {
		evidence = append(evidence, binding.Evidence...)
	}
	return evidence
}

func HasEvidenceCode(items []ProfileChannelReadinessEvidence, code string) bool {
	for _, item := range items {
		if item.Code == code {
			return true
		}
	}
	return false
}
