package gateway

import (
	"strconv"
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
)

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
	return ChannelSetupPlan{
		Channels: []ChannelSetupEntry{
			buildTelegramSetupEntry(cfg.Telegram),
			buildDiscordSetupEntry(cfg.Discord),
			buildSlackSetupEntry(cfg.Slack),
			buildWhatsAppSetupEntry(cfg),
			buildNavivoxSetupEntry(cfg.Navivox),
		},
		GatewayAction: "Start or restart messaging with: gormes gateway",
	}
}

func buildWhatsAppSetupEntry(cfg config.Config) ChannelSetupEntry {
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

	bindings := BuildProfileChannelReadiness(cfg).Bindings
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
			if evidence.Code == ProfileChannelEvidenceAccessPolicyMissing {
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
	return entry
}

func buildTelegramSetupEntry(cfg config.TelegramCfg) ChannelSetupEntry {
	entry := ChannelSetupEntry{
		ID:          "telegram",
		DisplayName: "Telegram",
		RequiredFields: []string{
			"telegram.bot_token",
			"telegram.allowed_user_ids or explicit open access",
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
		entry.PlannedWrites = append(entry.PlannedWrites, "telegram.bot_token -> .env")
	}
	if len(cfg.AllowedUserIDs) > 0 {
		entry.CurrentValues = append(entry.CurrentValues, "telegram.allowed_user_ids="+strconv.Itoa(len(cfg.AllowedUserIDs)))
	} else if !cfg.GuestMode {
		entry.PlannedWrites = append(entry.PlannedWrites, "telegram.allowed_user_ids or explicit open access -> config.toml")
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
		RequiredFields: []string{"discord.token", "discord.allowed_channel_id or first_run_discovery"},
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
	entry.PlannedWrites = []string{"discord.token -> .env", "discord.allowed_channel_id -> config.toml"}
	return entry
}

func buildSlackSetupEntry(cfg config.SlackCfg) ChannelSetupEntry {
	entry := ChannelSetupEntry{
		ID:             "slack",
		DisplayName:    "Slack",
		RequiredFields: []string{"slack.bot_token", "slack.app_token", "slack.allowed_channel_id or first_run_discovery"},
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
	entry.PlannedWrites = []string{"slack.bot_token -> .env", "slack.app_token -> .env", "slack.allowed_channel_id -> config.toml"}
	return entry
}

func buildNavivoxSetupEntry(cfg config.NavivoxCfg) ChannelSetupEntry {
	entry := ChannelSetupEntry{
		ID:             "navivox",
		DisplayName:    "Navivox",
		RequiredFields: []string{"navivox.enabled", "navivox.bind_host", "navivox.auth_mode"},
		NextCommand:    "gormes setup gateway",
	}
	if cfg.Enabled {
		entry.Status = ChannelSetupStatusConfigured
		entry.CurrentValues = append(entry.CurrentValues, "navivox.enabled=true", "navivox.token=[REDACTED]")
		return entry
	}
	entry.Status = ChannelSetupStatusUnconfigured
	entry.PlannedWrites = []string{"navivox.enabled -> config.toml"}
	return entry
}

func buildStaticSetupEntry(id, displayName string, required []string, nextCommand string) ChannelSetupEntry {
	return ChannelSetupEntry{
		ID:             id,
		DisplayName:    displayName,
		Status:         ChannelSetupStatusUnconfigured,
		RequiredFields: append([]string(nil), required...),
		PlannedWrites:  []string{nextCommand},
		NextCommand:    nextCommand,
	}
}
