package gateway

import (
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
)

func TestChannelSetupTelegramStatusAndRedaction(t *testing.T) {
	const token = "123456:secret-token-that-must-not-leak"
	plan := BuildChannelSetupPlan(config.Config{
		Telegram: config.TelegramCfg{BotToken: token},
	})
	telegram := findChannelSetupEntry(t, plan, "telegram")
	if telegram.Status != ChannelSetupStatusPartial {
		t.Fatalf("telegram status = %q, want partial for token-only config", telegram.Status)
	}
	if !strings.Contains(strings.Join(telegram.Warnings, "\n"), "access policy") {
		t.Fatalf("telegram warnings = %v, want access-policy warning", telegram.Warnings)
	}
	rendered := strings.Join(append(telegram.CurrentValues, telegram.PlannedWrites...), "\n")
	if strings.Contains(rendered, token) {
		t.Fatalf("channel setup plan leaked token in current/planned values: %s", rendered)
	}
	if !strings.Contains(rendered, "telegram.bot_token=[REDACTED]") {
		t.Fatalf("channel setup plan missing redacted token evidence: %s", rendered)
	}

	plan = BuildChannelSetupPlan(config.Config{
		Telegram: config.TelegramCfg{
			BotToken:       token,
			AllowedUserIDs: []int64{6586915095},
			HomeChannel: config.TelegramHomeChannelCfg{
				ChatID:   "-1001234567890",
				ThreadID: "42",
			},
		},
	})
	telegram = findChannelSetupEntry(t, plan, "telegram")
	if telegram.Status != ChannelSetupStatusConfigured {
		t.Fatalf("telegram status = %q, want configured with token, allowlist, and home channel", telegram.Status)
	}
	for _, want := range []string{
		"telegram.allowed_user_ids=1",
		"telegram.home_channel.chat_id=-1001234567890",
		"telegram.home_channel.thread_id=42",
	} {
		if !strings.Contains(strings.Join(telegram.CurrentValues, "\n"), want) {
			t.Fatalf("telegram current values missing %q: %v", want, telegram.CurrentValues)
		}
	}
}

func TestChannelSetupPlanListsMessagingPlatforms(t *testing.T) {
	plan := BuildChannelSetupPlan(config.Config{})
	for _, want := range []string{"telegram", "discord", "slack", "whatsapp", "navivox"} {
		entry := findChannelSetupEntry(t, plan, want)
		if entry.Status != ChannelSetupStatusUnconfigured {
			t.Fatalf("%s status = %q, want unconfigured in empty config", want, entry.Status)
		}
		if len(entry.RequiredFields) == 0 {
			t.Fatalf("%s RequiredFields empty, want setup guidance", want)
		}
	}
}

func findChannelSetupEntry(t *testing.T, plan ChannelSetupPlan, id string) ChannelSetupEntry {
	t.Helper()
	for _, entry := range plan.Channels {
		if entry.ID == id {
			return entry
		}
	}
	t.Fatalf("missing channel setup entry %q in %+v", id, plan.Channels)
	return ChannelSetupEntry{}
}
