package readiness

import (
	"crypto/sha256"
	"encoding/hex"
	"sort"
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway/profilechannels/contracts"
)

func BuildProfileChannelReadiness(cfg config.Config) contracts.ProfileChannelReadinessReport {
	return BuildProfileChannelReadinessWithOptions(cfg, contracts.ProfileChannelReadinessOptions{})
}

func BuildProfileChannelReadinessWithOptions(cfg config.Config, opts contracts.ProfileChannelReadinessOptions) contracts.ProfileChannelReadinessReport {
	credentialHashes := NormalizedCredentialHashes(opts.CredentialHashes)
	var report contracts.ProfileChannelReadinessReport
	for _, service := range cfg.EnabledProfileServices() {
		channelConfigs := SortedConfigBindings(service.Profile.Channels)
		for _, channelConfig := range channelConfigs {
			if !channelConfig.Config.Enabled {
				continue
			}
			binding := buildProfileChannelBindingReadiness(cfg, service.ID, channelConfig.Channel, channelConfig.Config)
			if hash := credentialHashes[binding.CredentialID]; hash != "" {
				binding.CredentialHash = hash
			}
			report.Bindings = append(report.Bindings, binding)
		}
	}
	applyProfileChannelTokenHashConflicts(&report)
	report.Evidence = contracts.CollectReadinessEvidence(report.Bindings)
	return report
}

func buildProfileChannelBindingReadiness(cfg config.Config, profileID, channel string, channelCfg config.ProfileChannelCfg) contracts.ProfileChannelBindingReadiness {
	channel = strings.ToLower(strings.TrimSpace(channel))
	allowedChats := normalizedProfileChannelAllowList(channel, channelCfg.AllowedChats)
	allowedUsers := normalizedProfileChannelAllowedUsers(channel, channelCfg.AllowedUsers)
	allowedDirectChatCount, allowedGroupChatCount := profileChannelAllowedChatShapeCounts(channel, allowedChats)
	credentialID := strings.TrimSpace(channelCfg.Credential)
	binding := contracts.ProfileChannelBindingReadiness{
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

func validateProfileChannelCredential(cfg config.Config, profileID, channel, credentialID string) []contracts.ProfileChannelReadinessEvidence {
	if credentialID == "" {
		return []contracts.ProfileChannelReadinessEvidence{contracts.NewEvidence(contracts.ProfileChannelEvidenceCredentialMissing, profileID, channel, credentialID, "credential", "profile channel has no credential id")}
	}
	credential, ok := cfg.Credentials[credentialID]
	if !ok {
		return []contracts.ProfileChannelReadinessEvidence{contracts.NewEvidence(contracts.ProfileChannelEvidenceCredentialMissing, profileID, channel, credentialID, "credential", "profile channel credential id is not defined")}
	}

	var evidence []contracts.ProfileChannelReadinessEvidence
	if kind := strings.ToLower(strings.TrimSpace(credential.Kind)); kind != "channel" {
		evidence = append(evidence, contracts.NewEvidence(contracts.ProfileChannelEvidenceCredentialKindMismatch, profileID, channel, credentialID, "kind", "credential kind must be channel"))
	}
	if credentialChannel := strings.ToLower(strings.TrimSpace(credential.Channel)); credentialChannel != "" && credentialChannel != channel {
		evidence = append(evidence, contracts.NewEvidence(contracts.ProfileChannelEvidenceCredentialChannelMismatch, profileID, channel, credentialID, "channel", "credential channel does not match profile channel"))
	}
	if ownerProfile := strings.TrimSpace(credential.OwnerProfile); ownerProfile != "" && ownerProfile != profileID {
		evidence = append(evidence, contracts.NewEvidence(contracts.ProfileChannelEvidenceCredentialOwnerMismatch, profileID, channel, credentialID, "owner_profile", "credential owner profile does not match profile channel"))
	}
	if credential.SecretRef == nil || strings.TrimSpace(credential.SecretRef.ID) == "" {
		evidence = append(evidence, contracts.NewEvidence(contracts.ProfileChannelEvidenceCredentialSecretMissing, profileID, channel, credentialID, "secret_ref", "channel credential has no secret ref"))
	}
	return evidence
}

func validateProfileChannelAccessPolicy(profileID, channel, credentialID string, allowedChats, allowedUsers []string) []contracts.ProfileChannelReadinessEvidence {
	if credentialID == "" || channel != "whatsapp" || len(allowedChats) > 0 || len(allowedUsers) > 0 {
		return nil
	}
	return []contracts.ProfileChannelReadinessEvidence{contracts.NewEvidence(contracts.ProfileChannelEvidenceAccessPolicyMissing, profileID, channel, credentialID, "allowed_chats", "profile channel has no WhatsApp chat or user allow-list")}
}

func applyProfileChannelTokenHashConflicts(report *contracts.ProfileChannelReadinessReport) {
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
		profileIDs := map[string]struct{}{}
		for _, index := range indexes {
			binding := report.Bindings[index]
			profileIDs[binding.ProfileID] = struct{}{}
		}
		if len(profileIDs) < 2 {
			continue
		}
		for _, index := range indexes {
			binding := &report.Bindings[index]
			evidence := contracts.NewEvidence(contracts.ProfileChannelEvidenceTokenHashConflict, binding.ProfileID, binding.Channel, binding.CredentialID, "credential_hash", "credential token hash is already assigned to another profile-channel binding")
			evidence.CredentialHash = binding.CredentialHash
			binding.Evidence = append(binding.Evidence, evidence)
			binding.Ready = false
		}
	}
}

func NormalizedCredentialHashes(in map[string]string) map[string]string {
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

func SortedConfigBindings(channels map[string]config.ProfileChannelCfg) []contracts.ConfigBinding {
	if len(channels) == 0 {
		return nil
	}
	type candidate struct {
		channel string
		rawKey  string
		config  config.ProfileChannelCfg
	}
	candidates := make([]candidate, 0, len(channels))
	for rawKey, channelConfig := range channels {
		channel := strings.ToLower(strings.TrimSpace(rawKey))
		if channel == "" {
			continue
		}
		candidates = append(candidates, candidate{channel: channel, rawKey: rawKey, config: channelConfig})
	}
	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].channel != candidates[j].channel {
			return candidates[i].channel < candidates[j].channel
		}
		if candidates[i].config.Enabled != candidates[j].config.Enabled {
			return candidates[i].config.Enabled
		}
		iExact := candidates[i].rawKey == candidates[i].channel
		jExact := candidates[j].rawKey == candidates[j].channel
		if iExact != jExact {
			return iExact
		}
		return candidates[i].rawKey < candidates[j].rawKey
	})

	out := make([]contracts.ConfigBinding, 0, len(candidates))
	seen := map[string]struct{}{}
	for _, candidate := range candidates {
		if _, ok := seen[candidate.channel]; ok {
			continue
		}
		seen[candidate.channel] = struct{}{}
		out = append(out, contracts.ConfigBinding{Channel: candidate.channel, Config: candidate.config})
	}
	return out
}

func normalizedProfileChannelAllowList(channel string, values []string) []string {
	if strings.ToLower(strings.TrimSpace(channel)) != "whatsapp" {
		return normalizedProfileChannelList(values)
	}
	return normalizedProfileChannelListWithCanonicalizer(values, strings.ToLower)
}

func normalizedProfileChannelAllowedUsers(channel string, values []string) []string {
	if strings.ToLower(strings.TrimSpace(channel)) != "whatsapp" {
		return normalizedProfileChannelList(values)
	}
	return normalizedProfileChannelListWithCanonicalizer(values, canonicalProfileChannelWhatsAppUserID)
}

func canonicalProfileChannelWhatsAppUserID(value string) string {
	value = strings.TrimPrefix(strings.ToLower(strings.TrimSpace(value)), "+")
	if before, _, ok := strings.Cut(value, "@"); ok {
		value = before
	}
	if before, _, ok := strings.Cut(value, ":"); ok {
		value = before
	}
	return strings.TrimSpace(value)
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
