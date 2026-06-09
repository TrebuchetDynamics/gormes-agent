package channelsetup

import (
	"strconv"
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway/profilechannels"
)

// PairingPlatformState is the operator-facing per-platform pairing state used
// by setup planning. It is duplicated here as a small read model so this pure
// package does not import the root gateway manager package.
type PairingPlatformState string

const (
	PairingPlatformStatePaired   PairingPlatformState = "paired"
	PairingPlatformStateUnpaired PairingPlatformState = "unpaired"
)

// PairingDegradedReason classifies read-only pairing-state degradation for
// setup guidance.
type PairingDegradedReason string

const (
	PairingDegradedMissing          PairingDegradedReason = "missing"
	PairingDegradedCorrupt          PairingDegradedReason = "corrupt"
	PairingDegradedPermissionDenied PairingDegradedReason = "permission_denied"
	PairingDegradedReadFailed       PairingDegradedReason = "read_failed"
)

type PairingPlatformStatus struct {
	Platform      string
	State         PairingPlatformState
	PendingCount  int
	ApprovedCount int
}

type PairingDegradedEvidence struct {
	Reason   PairingDegradedReason
	Path     string
	Message  string
	Platform string
	UserID   string
	Code     string
}

type PairingStatus struct {
	Platforms []PairingPlatformStatus
	Degraded  []PairingDegradedEvidence
}

type ChannelSetupStatus string

const (
	ChannelSetupStatusUnconfigured ChannelSetupStatus = "unconfigured"
	ChannelSetupStatusPartial      ChannelSetupStatus = "partial"
	ChannelSetupStatusConfigured   ChannelSetupStatus = "configured"
	ChannelSetupStatusPaired       ChannelSetupStatus = "paired"
	ChannelSetupStatusRunning      ChannelSetupStatus = "running"
	ChannelSetupStatusFailed       ChannelSetupStatus = "failed"
)

type ChannelSetupPlan struct {
	Channels      []ChannelSetupEntry
	GatewayAction string
}

// ChannelSetupPlanOptions carries optional read-only runtime evidence used to
// enrich setup guidance without reading live state from disk.
type ChannelSetupPlanOptions struct {
	Pairing PairingStatus
	// CredentialHashes carries caller-supplied redacted token hashes keyed by
	// credential id. Setup planning uses them only for duplicate ownership
	// evidence and never resolves live secret values.
	CredentialHashes map[string]string
}

type ChannelSetupEntry struct {
	ID             string
	DisplayName    string
	Status         ChannelSetupStatus
	RequiredFields []string
	CurrentValues  []string
	PlannedWrites  []string
	Warnings       []string
	NextCommand    string
}

func BuildChannelSetupPlan(cfg config.Config) ChannelSetupPlan {
	return BuildChannelSetupPlanWithOptions(cfg, ChannelSetupPlanOptions{})
}

// BuildChannelSetupPlanWithOptions builds setup guidance from config plus
// caller-supplied read-model evidence such as gateway pairing status.
func BuildChannelSetupPlanWithOptions(cfg config.Config, opts ChannelSetupPlanOptions) ChannelSetupPlan {
	return ChannelSetupPlan{
		Channels: []ChannelSetupEntry{
			buildTelegramSetupEntry(cfg.Telegram),
			buildDiscordSetupEntry(cfg.Discord),
			buildSlackSetupEntry(cfg.Slack),
			buildWhatsAppSetupEntry(cfg, opts),
			buildNavivoxSetupEntry(cfg.Navivox),
		},
		GatewayAction: "Start or restart messaging with: gormes gateway",
	}
}

func buildWhatsAppSetupEntry(cfg config.Config, opts ChannelSetupPlanOptions) ChannelSetupEntry {
	entry := ChannelSetupEntry{
		ID:          "whatsapp",
		DisplayName: "WhatsApp",
		RequiredFields: []string{
			"profiles.<id>.channels.whatsapp.credential",
			"profiles.<id>.channels.whatsapp.allowed_chats or explicit open access",
			"credentials.<id>.secret_ref",
		},
		NextCommand: "gormes whatsapp --plan",
	}

	bindings := profilechannels.BuildProfileChannelReadinessWithOptions(cfg, profilechannels.ProfileChannelReadinessOptions{CredentialHashes: opts.CredentialHashes}).Bindings
	readyCount := 0
	whatsAppCount := 0
	for _, binding := range bindings {
		if binding.Channel != "whatsapp" {
			continue
		}
		whatsAppCount++
		if binding.Ready {
			readyCount++
		}
		profilePrefix := "profiles." + binding.ProfileID + ".channels.whatsapp"
		if binding.CredentialID != "" {
			entry.CurrentValues = append(entry.CurrentValues, profilePrefix+".credential="+binding.CredentialID)
		} else {
			entry.PlannedWrites = append(entry.PlannedWrites, profilePrefix+".credential -> config.toml")
		}
		if binding.SecretRefConfigured {
			entry.CurrentValues = append(entry.CurrentValues, "credentials."+binding.CredentialID+".secret_ref=[REDACTED:"+binding.SecretRefSource+"]")
		} else if binding.CredentialID != "" {
			entry.PlannedWrites = append(entry.PlannedWrites, "credentials."+binding.CredentialID+".secret_ref -> secret store")
		}
		entry.CurrentValues = append(entry.CurrentValues,
			profilePrefix+".allowed_chats="+strconv.Itoa(binding.AllowedChatCount),
			profilePrefix+".allowed_direct_chats="+strconv.Itoa(binding.AllowedDirectChatCount),
			profilePrefix+".allowed_group_chats="+strconv.Itoa(binding.AllowedGroupChatCount),
			profilePrefix+".allowed_users="+strconv.Itoa(binding.AllowedUserCount),
		)
		for _, evidence := range binding.Evidence {
			entry.Warnings = append(entry.Warnings, profilePrefix+": "+evidence.Code+" ("+evidence.Field+")")
			if evidence.Code == profilechannels.ProfileChannelEvidenceAccessPolicyMissing {
				entry.PlannedWrites = append(entry.PlannedWrites, profilePrefix+".allowed_chats or allowed_users -> config.toml")
			}
		}
	}

	switch {
	case whatsAppCount == 0:
		entry.Status = ChannelSetupStatusUnconfigured
		entry.PlannedWrites = []string{"profiles.<id>.channels.whatsapp -> config.toml", "WhatsApp session credentials -> gormes whatsapp --plan"}
	case readyCount == whatsAppCount:
		entry.Status = ChannelSetupStatusConfigured
	default:
		entry.Status = ChannelSetupStatusPartial
	}
	applyWhatsAppPairingSetupStatus(&entry, opts.Pairing)
	return entry
}

func applyWhatsAppPairingSetupStatus(entry *ChannelSetupEntry, pairing PairingStatus) {
	if entry == nil {
		return
	}
	for _, platform := range pairing.Platforms {
		if !strings.EqualFold(strings.TrimSpace(platform.Platform), "whatsapp") {
			continue
		}
		state := PairingPlatformState(strings.ToLower(strings.TrimSpace(string(platform.State))))
		if state == "" {
			break
		}
		entry.CurrentValues = append(entry.CurrentValues,
			"whatsapp.pairing="+string(state),
			"whatsapp.pairing_approved_users="+strconv.Itoa(nonNegativeCount(platform.ApprovedCount)),
			"whatsapp.pairing_pending_codes="+strconv.Itoa(nonNegativeCount(platform.PendingCount)),
		)
		if state == PairingPlatformStatePaired && entry.Status == ChannelSetupStatusConfigured {
			entry.Status = ChannelSetupStatusPaired
			entry.NextCommand = "gormes gateway"
		}
		if state == PairingPlatformStateUnpaired && entry.Status == ChannelSetupStatusConfigured {
			entry.Status = ChannelSetupStatusPartial
			entry.NextCommand = "gormes whatsapp"
			entry.Warnings = append(entry.Warnings, "WhatsApp pairing is not complete; run gormes whatsapp to link a device.")
			entry.PlannedWrites = append(entry.PlannedWrites, "WhatsApp pairing session -> gormes whatsapp")
		}
		break
	}
	applyWhatsAppPairingDegradedSetupStatus(entry, pairing.Degraded)
}

func nonNegativeCount(value int) int {
	if value < 0 {
		return 0
	}
	return value
}

func applyWhatsAppPairingDegradedSetupStatus(entry *ChannelSetupEntry, degraded []PairingDegradedEvidence) {
	seen := map[string]struct{}{}
	for _, evidence := range degraded {
		if !isWhatsAppPairingDegradedEvidence(evidence) {
			continue
		}
		reason := PairingDegradedReason(strings.ToLower(strings.TrimSpace(string(evidence.Reason))))
		if reason == "" {
			continue
		}
		reasonText := string(reason)
		if _, ok := seen[reasonText]; ok {
			continue
		}
		seen[reasonText] = struct{}{}
		entry.CurrentValues = append(entry.CurrentValues, "whatsapp.pairing_degraded="+reasonText)
		entry.Warnings = append(entry.Warnings, "WhatsApp pairing status is degraded: "+reasonText)
		if whatsappPairingDegradationBlocksSetup(reason) && (entry.Status == ChannelSetupStatusConfigured || entry.Status == ChannelSetupStatusPaired) {
			entry.Status = ChannelSetupStatusPartial
			entry.NextCommand = "gormes whatsapp"
			entry.PlannedWrites = append(entry.PlannedWrites, "WhatsApp pairing readout repair -> gormes whatsapp")
		}
	}
}

func isWhatsAppPairingDegradedEvidence(evidence PairingDegradedEvidence) bool {
	platform := strings.ToLower(strings.TrimSpace(evidence.Platform))
	return platform == "" || platform == "whatsapp"
}

func whatsappPairingDegradationBlocksSetup(reason PairingDegradedReason) bool {
	switch reason {
	case PairingDegradedMissing, PairingDegradedCorrupt, PairingDegradedPermissionDenied, PairingDegradedReadFailed:
		return true
	default:
		return false
	}
}

func buildTelegramSetupEntry(cfg config.TelegramCfg) ChannelSetupEntry {
	entry := ChannelSetupEntry{
		ID:          "telegram",
		DisplayName: "Telegram",
		RequiredFields: []string{
			"profiles.<id>.channels.telegram.credential",
			"credentials.<id>.secret_ref",
			"profiles.<id>.channels.telegram.allowed_users/allowed_chats or explicit open access",
			"telegram.home_channel.chat_id",
		},
		NextCommand: "gormes setup gateway",
	}
	tokenConfigured := strings.TrimSpace(cfg.BotToken) != "" || cfg.BotTokenRef != nil
	policyConfigured := len(cfg.AllowedUserIDs) > 0 || len(cfg.AllowedChatIDs()) > 0 || cfg.AllowedChatID != 0 || cfg.GuestMode
	homeConfigured := strings.TrimSpace(cfg.HomeChannel.ChatID) != "" || cfg.AllowedChatID != 0

	switch {
	case !tokenConfigured:
		entry.Status = ChannelSetupStatusUnconfigured
	case !policyConfigured:
		entry.Status = ChannelSetupStatusPartial
		entry.Warnings = append(entry.Warnings, "Telegram token is configured but access policy is missing.")
	default:
		entry.Status = ChannelSetupStatusConfigured
	}
	if tokenConfigured {
		entry.CurrentValues = append(entry.CurrentValues, "telegram.bot_token=[REDACTED]")
	} else {
		entry.PlannedWrites = append(entry.PlannedWrites, "profiles.<id>.channels.telegram.credential -> config.toml", "credentials.<id>.secret_ref -> profile-scoped .env", "telegram.bot_token_ref -> config.toml")
	}
	if len(cfg.AllowedUserIDs) > 0 {
		entry.CurrentValues = append(entry.CurrentValues, "telegram.allowed_user_ids="+strconv.Itoa(len(cfg.AllowedUserIDs)))
	} else if !cfg.GuestMode {
		entry.PlannedWrites = append(entry.PlannedWrites, "profiles.<id>.channels.telegram.allowed_users/allowed_chats or explicit open access -> config.toml")
	}
	if cfg.GuestMode {
		entry.CurrentValues = append(entry.CurrentValues, "telegram.access_policy=open")
	}
	homeChatID := strings.TrimSpace(cfg.HomeChannel.ChatID)
	if homeChatID == "" && cfg.AllowedChatID != 0 {
		homeChatID = strconv.FormatInt(cfg.AllowedChatID, 10)
	}
	if homeChatID != "" {
		entry.CurrentValues = append(entry.CurrentValues, "telegram.home_channel.chat_id="+homeChatID)
	} else {
		entry.PlannedWrites = append(entry.PlannedWrites, "telegram.home_channel.chat_id -> config.toml")
		if tokenConfigured {
			entry.Warnings = append(entry.Warnings, "Telegram home channel is missing; /set-home or setup can fill it after pairing.")
		}
	}
	if threadID := strings.TrimSpace(cfg.HomeChannel.ThreadID); threadID != "" {
		entry.CurrentValues = append(entry.CurrentValues, "telegram.home_channel.thread_id="+threadID)
	}
	if !homeConfigured && tokenConfigured && policyConfigured {
		entry.Status = ChannelSetupStatusConfigured
	}
	return entry
}

func buildDiscordSetupEntry(cfg config.DiscordCfg) ChannelSetupEntry {
	entry := ChannelSetupEntry{
		ID:             "discord",
		DisplayName:    "Discord",
		RequiredFields: []string{"profiles.<id>.channels.discord.credential", "credentials.<id>.secret_ref", "profiles.<id>.channels.discord.allowed_chats or first_run_discovery"},
		NextCommand:    "gormes setup gateway",
	}
	if cfg.Enabled() {
		entry.Status = ChannelSetupStatusConfigured
		entry.CurrentValues = append(entry.CurrentValues, "discord.token=[REDACTED]")
		if strings.TrimSpace(cfg.AllowedChannelID) != "" {
			entry.CurrentValues = append(entry.CurrentValues, "discord.allowed_channel_id="+strings.TrimSpace(cfg.AllowedChannelID))
		}
		return entry
	}
	entry.Status = ChannelSetupStatusUnconfigured
	entry.PlannedWrites = []string{"profiles.<id>.channels.discord.credential -> config.toml", "credentials.<id>.secret_ref -> profile-scoped .env", "discord.token_ref -> config.toml", "discord.allowed_channel_id -> config.toml"}
	return entry
}

func buildSlackSetupEntry(cfg config.SlackCfg) ChannelSetupEntry {
	entry := ChannelSetupEntry{
		ID:             "slack",
		DisplayName:    "Slack",
		RequiredFields: []string{"profiles.<id>.channels.slack.credential", "credentials.<id>.secret_ref", "credentials.<id>-slack_app.secret_ref", "profiles.<id>.channels.slack.allowed_chats or first_run_discovery"},
		NextCommand:    "gormes setup gateway",
	}
	if cfg.Enabled {
		entry.Status = ChannelSetupStatusConfigured
		entry.CurrentValues = append(entry.CurrentValues, "slack.bot_token=[REDACTED]", "slack.app_token=[REDACTED]")
		if strings.TrimSpace(cfg.AllowedChannelID) != "" {
			entry.CurrentValues = append(entry.CurrentValues, "slack.allowed_channel_id="+strings.TrimSpace(cfg.AllowedChannelID))
		}
		return entry
	}
	entry.Status = ChannelSetupStatusUnconfigured
	entry.PlannedWrites = []string{"profiles.<id>.channels.slack.credential -> config.toml", "credentials.<id>.secret_ref -> profile-scoped .env", "credentials.<id>-slack_app.secret_ref -> profile-scoped .env", "slack.bot_token_ref -> config.toml", "slack.app_token_ref -> config.toml", "slack.allowed_channel_id -> config.toml"}
	return entry
}

func buildNavivoxSetupEntry(cfg config.NavivoxCfg) ChannelSetupEntry {
	entry := ChannelSetupEntry{
		ID:             "navivox",
		DisplayName:    "Navivox",
		RequiredFields: []string{"profiles.<id>.channels.navivox.credential", "credentials.<id>.secret_ref when token auth is enabled", "navivox.enabled", "navivox.bind_host", "navivox.auth_mode"},
		NextCommand:    "gormes setup gateway",
	}
	if cfg.Enabled {
		entry.Status = ChannelSetupStatusConfigured
		entry.CurrentValues = append(entry.CurrentValues, "navivox.enabled=true", "navivox.token=[REDACTED]")
		return entry
	}
	entry.Status = ChannelSetupStatusUnconfigured
	entry.PlannedWrites = []string{"profiles.<id>.channels.navivox.credential -> config.toml", "credentials.<id>.secret_ref -> profile-scoped .env when token auth is enabled", "navivox.enabled -> config.toml"}
	return entry
}
