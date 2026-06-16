package channels

import (
	"strconv"
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
)

type Getenv func(string) string

func ConfiguredCapabilityDetails(cfg config.Config, getenv Getenv) map[string]string {
	if getenv == nil {
		getenv = func(string) string { return "" }
	}
	details := map[string]string{}
	if cfg.Telegram.BotToken != "" {
		details["telegram"] = configuredTelegramGatewayStatusDetail(cfg.Telegram)
	}
	if cfg.Discord.Enabled() {
		detail := "first_run_discovery=" + strconv.FormatBool(cfg.Discord.FirstRunDiscovery)
		if cfg.Discord.AllowedChannelID != "" {
			detail = "allowed_channel_id=" + cfg.Discord.AllowedChannelID
		}
		details["discord"] = detail
	}
	if detail := ConfiguredWhatsAppGatewayStatusDetail(getenv); detail != "" {
		details["whatsapp"] = detail
	}
	if cfg.Slack.Enabled {
		details["slack"] = configuredSlackGatewayStatusDetail(cfg.Slack)
	}
	if cfg.Teams.Enabled {
		details["teams"] = cfg.Teams.RedactedStatus()
	}
	if cfg.Navivox.Enabled {
		details["navivox"] = configuredNavivoxGatewayStatusDetail(cfg.Navivox)
	}
	if cfg.Yuanbao.Enabled {
		details["yuanbao"] = cfg.Yuanbao.RedactedStatus()
	}
	return details
}

func ConfiguredWhatsAppGatewayStatusDetail(getenv Getenv) string {
	if getenv == nil {
		getenv = func(string) string { return "" }
	}
	if !strings.EqualFold(strings.TrimSpace(getenv("WHATSAPP_ENABLED")), "true") {
		return ""
	}
	mode := strings.TrimSpace(getenv("WHATSAPP_MODE"))
	if mode == "" {
		return "enabled=true"
	}
	return "mode=" + mode
}

func configuredTelegramGatewayStatusDetail(cfg config.TelegramCfg) string {
	detail := "first_run_discovery=" + strconv.FormatBool(cfg.FirstRunDiscovery)
	if cfg.AllowedChatID != 0 {
		detail = "allowed_chat_id=" + strconv.FormatInt(cfg.AllowedChatID, 10)
	}
	if len(cfg.AllowedUserIDs) > 0 {
		userDetail := "allowed_users=" + strconv.Itoa(len(cfg.AllowedUserIDs))
		if detail == "" {
			return userDetail
		}
		return detail + " " + userDetail
	}
	return detail
}

func configuredNavivoxGatewayStatusDetail(cfg config.NavivoxCfg) string {
	bindHost := strings.TrimSpace(cfg.BindHost)
	if bindHost == "" {
		bindHost = config.NavivoxDefaultBindHost
	}
	port := cfg.Port
	if port == 0 {
		port = config.NavivoxDefaultPort
	}
	exposure := strings.ToLower(strings.TrimSpace(cfg.ExposureMode))
	if exposure == "" {
		exposure = config.NavivoxExposureLocal
	}
	auth := strings.ToLower(strings.TrimSpace(cfg.AuthMode))
	if auth == "" {
		auth = config.NavivoxAuthPairingToken
	}
	return "bind=" + bindHost + ":" + strconv.Itoa(port) + " exposure=" + exposure + " auth=" + auth
}

func configuredSlackGatewayStatusDetail(cfg config.SlackCfg) string {
	if missing := missingSlackCredentials(cfg); len(missing) > 0 {
		return "missing_tokens=" + strings.Join(missing, ",")
	}
	detail := "first_run_discovery=" + strconv.FormatBool(cfg.FirstRunDiscovery)
	if cfg.AllowedChannelID != "" {
		detail = "allowed_channel_id=" + cfg.AllowedChannelID
	}
	return detail
}

func missingSlackCredentials(cfg config.SlackCfg) []string {
	missing := []string{}
	if strings.TrimSpace(cfg.BotToken) == "" {
		missing = append(missing, "bot_token")
	}
	if strings.TrimSpace(cfg.AppToken) == "" {
		missing = append(missing, "app_token")
	}
	return missing
}
