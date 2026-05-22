package gateway

import (
	"crypto/sha256"
	"encoding/hex"
	"sort"
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
)

const (
	ProfileChannelEvidenceCredentialMissing         = "channel_credential_missing"
	ProfileChannelEvidenceCredentialKindMismatch    = "channel_credential_kind_mismatch"
	ProfileChannelEvidenceCredentialChannelMismatch = "channel_credential_channel_mismatch"
	ProfileChannelEvidenceCredentialOwnerMismatch   = "channel_credential_owner_mismatch"
	ProfileChannelEvidenceCredentialSecretMissing   = "channel_credential_secret_missing"
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

func BuildProfileChannelReadiness(cfg config.Config) ProfileChannelReadinessReport {
	return BuildProfileChannelReadinessWithOptions(cfg, ProfileChannelReadinessOptions{})
}

func BuildProfileChannelReadinessWithOptions(cfg config.Config, opts ProfileChannelReadinessOptions) ProfileChannelReadinessReport {
	credentialHashes := normalizedProfileChannelCredentialHashes(opts.CredentialHashes)
	var report ProfileChannelReadinessReport
	for _, service := range cfg.EnabledProfileServices() {
		channelNames := sortedProfileChannelNames(service.Profile.Channels)
		for _, channel := range channelNames {
			channelCfg := service.Profile.Channels[channel]
			if !channelCfg.Enabled {
				continue
			}
			binding := buildProfileChannelBindingReadiness(cfg, service.ID, channel, channelCfg)
			if hash := credentialHashes[binding.CredentialID]; hash != "" {
				binding.CredentialHash = hash
			}
			report.Bindings = append(report.Bindings, binding)
		}
	}
	applyProfileChannelTokenHashConflicts(&report)
	report.Evidence = collectProfileChannelReadinessEvidence(report.Bindings)
	return report
}

func buildProfileChannelBindingReadiness(cfg config.Config, profileID, channel string, channelCfg config.ProfileChannelCfg) ProfileChannelBindingReadiness {
	channel = strings.ToLower(strings.TrimSpace(channel))
	allowedChats := normalizedProfileChannelAllowList(channel, channelCfg.AllowedChats)
	allowedUsers := normalizedProfileChannelAllowList(channel, channelCfg.AllowedUsers)
	allowedDirectChatCount, allowedGroupChatCount := profileChannelAllowedChatShapeCounts(channel, allowedChats)
	credentialID := strings.TrimSpace(channelCfg.Credential)
	binding := ProfileChannelBindingReadiness{
		ProfileID:              strings.TrimSpace(profileID),
		Channel:                channel,
		CredentialID:           credentialID,
		AllowedChatCount:       len(allowedChats),
		AllowedChatScopeHash:   profileChannelScopeHash("allowed_chats", allowedChats),
		AllowedDirectChatCount: allowedDirectChatCount,
		AllowedGroupChatCount:  allowedGroupChatCount,
		AllowedUserCount:       len(allowedUsers),
		AllowedUserScopeHash:   profileChannelScopeHash("allowed_users", allowedUsers),
		RequireMention:         channelCfg.RequireMention,
		ToolProgress:           strings.ToLower(strings.TrimSpace(channelCfg.ToolProgress)),
		ServerCount:            len(normalizedProfileChannelList(channelCfg.Servers)),
		VoiceProfileSet:        strings.TrimSpace(channelCfg.VoiceProfile) != "",
	}
	binding.Evidence = validateProfileChannelCredential(cfg, binding.ProfileID, binding.Channel, credentialID)
	binding.Evidence = append(binding.Evidence, validateProfileChannelAccessPolicy(binding.ProfileID, binding.Channel, credentialID, allowedChats, allowedUsers)...)
	if credentialID != "" {
		if credential, ok := cfg.Credentials[credentialID]; ok {
			binding.CredentialOwnerProfile = strings.TrimSpace(credential.OwnerProfile)
			if credential.SecretRef != nil {
				binding.SecretRefConfigured = true
				binding.SecretRefSource = strings.ToLower(strings.TrimSpace(string(credential.SecretRef.Source)))
			}
		}
	}
	binding.Ready = len(binding.Evidence) == 0
	return binding
}

func validateProfileChannelCredential(cfg config.Config, profileID, channel, credentialID string) []ProfileChannelReadinessEvidence {
	if credentialID == "" {
		return []ProfileChannelReadinessEvidence{newProfileChannelEvidence(ProfileChannelEvidenceCredentialMissing, profileID, channel, credentialID, "credential", "profile channel has no credential id")}
	}
	credential, ok := cfg.Credentials[credentialID]
	if !ok {
		return []ProfileChannelReadinessEvidence{newProfileChannelEvidence(ProfileChannelEvidenceCredentialMissing, profileID, channel, credentialID, "credential", "profile channel credential id is not defined")}
	}

	var evidence []ProfileChannelReadinessEvidence
	if kind := strings.ToLower(strings.TrimSpace(credential.Kind)); kind != "channel" {
		evidence = append(evidence, newProfileChannelEvidence(ProfileChannelEvidenceCredentialKindMismatch, profileID, channel, credentialID, "kind", "credential kind must be channel"))
	}
	if credentialChannel := strings.ToLower(strings.TrimSpace(credential.Channel)); credentialChannel != "" && credentialChannel != channel {
		evidence = append(evidence, newProfileChannelEvidence(ProfileChannelEvidenceCredentialChannelMismatch, profileID, channel, credentialID, "channel", "credential channel does not match profile channel"))
	}
	if ownerProfile := strings.TrimSpace(credential.OwnerProfile); ownerProfile != "" && ownerProfile != profileID {
		evidence = append(evidence, newProfileChannelEvidence(ProfileChannelEvidenceCredentialOwnerMismatch, profileID, channel, credentialID, "owner_profile", "credential owner profile does not match profile channel"))
	}
	if credential.SecretRef == nil || strings.TrimSpace(credential.SecretRef.ID) == "" {
		evidence = append(evidence, newProfileChannelEvidence(ProfileChannelEvidenceCredentialSecretMissing, profileID, channel, credentialID, "secret_ref", "channel credential has no secret ref"))
	}
	return evidence
}

func validateProfileChannelAccessPolicy(profileID, channel, credentialID string, allowedChats, allowedUsers []string) []ProfileChannelReadinessEvidence {
	if credentialID == "" || channel != "whatsapp" || len(allowedChats) > 0 || len(allowedUsers) > 0 {
		return nil
	}
	return []ProfileChannelReadinessEvidence{newProfileChannelEvidence(ProfileChannelEvidenceAccessPolicyMissing, profileID, channel, credentialID, "allowed_chats", "profile channel has no WhatsApp chat or user allow-list")}
}

func applyProfileChannelTokenHashConflicts(report *ProfileChannelReadinessReport) {
	if report == nil || len(report.Bindings) < 2 {
		return
	}
	type conflictKey struct {
		channel string
		hash    string
	}
	buckets := map[conflictKey][]int{}
	for i := range report.Bindings {
		binding := &report.Bindings[i]
		binding.CredentialHash = normalizeProfileChannelCredentialHash(binding.CredentialHash)
		if binding.Channel == "" || binding.CredentialID == "" || binding.CredentialHash == "" {
			continue
		}
		key := conflictKey{channel: binding.Channel, hash: binding.CredentialHash}
		buckets[key] = append(buckets[key], i)
	}

	for _, indexes := range buckets {
		if len(indexes) < 2 {
			continue
		}
		credentialIDs := map[string]struct{}{}
		profileIDs := map[string]struct{}{}
		for _, index := range indexes {
			binding := report.Bindings[index]
			credentialIDs[binding.CredentialID] = struct{}{}
			profileIDs[binding.ProfileID] = struct{}{}
		}
		if len(credentialIDs) < 2 || len(profileIDs) < 2 {
			continue
		}
		for _, index := range indexes {
			binding := &report.Bindings[index]
			evidence := newProfileChannelEvidence(ProfileChannelEvidenceTokenHashConflict, binding.ProfileID, binding.Channel, binding.CredentialID, "credential_hash", "credential token hash is already assigned to another profile-channel binding")
			evidence.CredentialHash = binding.CredentialHash
			binding.Evidence = append(binding.Evidence, evidence)
			binding.Ready = false
		}
	}
}

func collectProfileChannelReadinessEvidence(bindings []ProfileChannelBindingReadiness) []ProfileChannelReadinessEvidence {
	var evidence []ProfileChannelReadinessEvidence
	for _, binding := range bindings {
		evidence = append(evidence, binding.Evidence...)
	}
	return evidence
}

func normalizedProfileChannelCredentialHashes(in map[string]string) map[string]string {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]string, len(in))
	for credentialID, hash := range in {
		credentialID = strings.TrimSpace(credentialID)
		hash = normalizeProfileChannelCredentialHash(hash)
		if credentialID == "" || hash == "" {
			continue
		}
		out[credentialID] = hash
	}
	return out
}

func normalizeProfileChannelCredentialHash(hash string) string {
	return strings.ToLower(strings.TrimSpace(hash))
}

func sortedProfileChannelNames(channels map[string]config.ProfileChannelCfg) []string {
	if len(channels) == 0 {
		return nil
	}
	names := make([]string, 0, len(channels))
	for channel := range channels {
		channel = strings.ToLower(strings.TrimSpace(channel))
		if channel != "" {
			names = append(names, channel)
		}
	}
	sort.Strings(names)
	return names
}

func normalizedProfileChannelAllowList(channel string, values []string) []string {
	if strings.ToLower(strings.TrimSpace(channel)) != "whatsapp" {
		return normalizedProfileChannelList(values)
	}
	return normalizedProfileChannelListWithCanonicalizer(values, strings.ToLower)
}

func normalizedProfileChannelList(values []string) []string {
	return normalizedProfileChannelListWithCanonicalizer(values, func(value string) string { return value })
}

func normalizedProfileChannelListWithCanonicalizer(values []string, canonicalize func(string) string) []string {
	if len(values) == 0 {
		return nil
	}
	out := make([]string, 0, len(values))
	seen := map[string]struct{}{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if canonicalize != nil {
			value = canonicalize(value)
		}
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}

func profileChannelScopeHash(kind string, values []string) string {
	if len(values) == 0 {
		return ""
	}
	sum := sha256.Sum256([]byte(kind + "\x00" + strings.Join(values, "\x00")))
	return hex.EncodeToString(sum[:])
}

func profileChannelAllowedChatShapeCounts(channel string, allowedChats []string) (direct int, group int) {
	if strings.ToLower(strings.TrimSpace(channel)) != "whatsapp" {
		return 0, 0
	}
	for _, chat := range allowedChats {
		chat = strings.ToLower(strings.TrimSpace(chat))
		if chat == "" {
			continue
		}
		if strings.HasSuffix(chat, "@g.us") {
			group++
			continue
		}
		direct++
	}
	return direct, group
}

func newProfileChannelEvidence(code, profileID, channel, credentialID, field, message string) ProfileChannelReadinessEvidence {
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
