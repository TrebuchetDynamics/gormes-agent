package gateway

import (
	"log/slog"
	"os"
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	runtimegateway "github.com/TrebuchetDynamics/gormes-agent/internal/gateway"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/gormescli/channelruntime"
)

// StartupSecurityReport is the sanitized startup admission result used before
// opening live gateway channels. Config may have weak placeholder credentials
// blanked so the foreground gateway cannot accidentally enable those channels.
type StartupSecurityReport struct {
	Config   config.Config
	Evidence []runtimegateway.AdmissionEvidence
}

// EvaluateStartupSecurity preserves the gateway startup guards for missing
// allowlists and placeholder credentials. It returns a copy of cfg with weak
// credential platforms disabled plus redacted admission evidence for logs.
func EvaluateStartupSecurity(cfg config.Config, lookupEnv func(string) string) StartupSecurityReport {
	if lookupEnv == nil {
		lookupEnv = os.Getenv
	}
	report := StartupSecurityReport{Config: cfg}
	report.Evidence = append(report.Evidence, runtimegateway.CheckStartupAllowlist(runtimegateway.StartupAdmissionInput{
		AllowlistConfigured: StartupAllowlistConfigured(cfg, lookupEnv),
		AllowAll:            StartupAllowAllConfigured(lookupEnv),
	})...)
	credentialReport := runtimegateway.CheckWeakCredentialPlatforms([]runtimegateway.CredentialGuardPlatform{
		{
			Name:    "telegram",
			Enabled: strings.TrimSpace(cfg.Telegram.BotToken) != "",
			Credentials: []runtimegateway.CredentialGuardValue{{
				Field: "bot_token",
				Value: cfg.Telegram.BotToken,
			}},
		},
		{
			Name:    "discord",
			Enabled: cfg.Discord.Enabled(),
			Credentials: []runtimegateway.CredentialGuardValue{{
				Field: "token",
				Value: cfg.Discord.Token,
			}},
		},
		{
			Name:    "slack",
			Enabled: cfg.Slack.Enabled,
			Credentials: []runtimegateway.CredentialGuardValue{
				{Field: "bot_token", Value: cfg.Slack.BotToken},
				{Field: "app_token", Value: cfg.Slack.AppToken},
			},
		},
	})
	report.Evidence = append(report.Evidence, credentialReport.Evidence...)
	for _, platform := range credentialReport.DisabledPlatforms {
		switch platform {
		case "telegram":
			report.Config.Telegram.BotToken = ""
		case "discord":
			report.Config.Discord.Token = ""
		case "slack":
			report.Config.Slack.Enabled = false
			report.Config.Slack.BotToken = ""
			report.Config.Slack.AppToken = ""
		}
	}
	return report
}

// StartupAllowlistConfigured reports whether startup has a scoped gateway
// allowlist configured through config or supported environment variables.
func StartupAllowlistConfigured(cfg config.Config, lookupEnv func(string) string) bool {
	if cfg.Telegram.AllowedChatID != 0 || len(cfg.Telegram.AllowedUserIDs) > 0 {
		return true
	}
	if strings.TrimSpace(cfg.Discord.AllowedChannelID) != "" {
		return true
	}
	if strings.TrimSpace(cfg.Slack.AllowedChannelID) != "" {
		return true
	}
	if len(cfg.Teams.AllowedUserIDs()) > 0 || cfg.Teams.AllowAllUsers {
		return true
	}
	if strings.TrimSpace(cfg.Yuanbao.AllowedConversationID) != "" {
		return true
	}
	if cfg.Navivox.Enabled {
		return true
	}
	if channelruntime.SimpleXStartupAllowlistConfigured(lookupEnv) {
		return true
	}
	for _, key := range []string{
		"SIGNAL_GROUP_ALLOWED_USERS",
		"GORMES_TELEGRAM_ALLOWED_USERS",
		"TELEGRAM_ALLOWED_USERS",
		"GORMES_DISCORD_CHANNEL_ID",
		"GORMES_SLACK_CHANNEL_ID",
		"TEAMS_ALLOWED_USERS",
	} {
		if strings.TrimSpace(lookupEnv(key)) != "" {
			return true
		}
	}
	return false
}

// StartupAllowAllConfigured reports whether startup explicitly opted into
// allowing all gateway users for a supported channel.
func StartupAllowAllConfigured(lookupEnv func(string) string) bool {
	for _, key := range []string{"GATEWAY_ALLOW_ALL_USERS", "TELEGRAM_ALLOW_ALL_USERS", "TEAMS_ALLOW_ALL_USERS", "SIMPLEX_ALLOW_ALL_USERS"} {
		if parseStartupBool(lookupEnv(key)) {
			return true
		}
	}
	return false
}

func parseStartupBool(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "1", "t", "true", "y", "yes", "on":
		return true
	default:
		return false
	}
}

// LogStartupSecurityEvidence writes redacted startup admission findings to the
// gateway logger while skipping empty evidence entries.
func LogStartupSecurityEvidence(evidence []runtimegateway.AdmissionEvidence, log *slog.Logger) {
	if log == nil {
		log = slog.Default()
	}
	for _, item := range evidence {
		if item.Code == "" {
			continue
		}
		log.Warn("gateway startup admission", "code", item.Code, "platform", item.Platform, "field", item.Field, "message", item.Message)
	}
}
