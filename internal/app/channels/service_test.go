package channels

import (
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
)

func TestConfiguredWhatsAppGatewayStatusDetail(t *testing.T) {
	cases := []struct {
		name string
		env  map[string]string
		want string
	}{
		{name: "disabled", env: map[string]string{"WHATSAPP_ENABLED": "false"}, want: ""},
		{name: "enabled no mode", env: map[string]string{"WHATSAPP_ENABLED": "true"}, want: "enabled=true"},
		{name: "enabled mode", env: map[string]string{"WHATSAPP_ENABLED": " true ", "WHATSAPP_MODE": " bot "}, want: "mode=bot"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := ConfiguredWhatsAppGatewayStatusDetail(func(key string) string { return tc.env[key] })
			if got != tc.want {
				t.Fatalf("ConfiguredWhatsAppGatewayStatusDetail = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestConfiguredCapabilityDetailsRedactsAndSummarizes(t *testing.T) {
	cfg := config.Config{}
	cfg.Telegram.BotToken = "12345:secret-token"
	cfg.Telegram.AllowedChatID = 42
	cfg.Slack.Enabled = true
	cfg.Slack.BotToken = "xoxb-secret"
	cfg.Slack.AppToken = "xapp-secret"
	cfg.Slack.AllowedChannelID = "C123"
	cfg.Teams.Enabled = true
	cfg.Teams.ClientID = "teams-client"
	cfg.Teams.ClientSecret = "teams-secret"
	cfg.Teams.TenantID = "teams-tenant"
	cfg.Teams.Port = 5001
	cfg.Teams.AllowedUsers = []string{"aad-1", "aad-2"}
	cfg.Teams.AllowAllUsers = true

	details := ConfiguredCapabilityDetails(cfg, func(key string) string {
		if key == "WHATSAPP_ENABLED" {
			return "true"
		}
		if key == "WHATSAPP_MODE" {
			return "bot"
		}
		return ""
	})

	wants := map[string]string{
		"telegram": "allowed_chat_id=42",
		"slack":    "allowed_channel_id=C123",
		"whatsapp": "mode=bot",
		"teams":    "configured port=5001 allowed_users=2 allow_all_users=true",
	}
	for key, want := range wants {
		if got := details[key]; got != want {
			t.Fatalf("details[%q] = %q, want %q (all %#v)", key, got, want, details)
		}
	}
}

func TestConfiguredCapabilityDetailsReportsMissingSlackTokens(t *testing.T) {
	cfg := config.Config{}
	cfg.Slack.Enabled = true
	if got := ConfiguredCapabilityDetails(cfg, nil)["slack"]; got != "missing_tokens=bot_token,app_token" {
		t.Fatalf("slack detail = %q", got)
	}
}
