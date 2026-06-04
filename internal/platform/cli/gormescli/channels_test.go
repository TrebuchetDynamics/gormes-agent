package gormescli

import (
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
)

func TestConfiguredChannelCapabilityDetailsRedactsTelegramToken(t *testing.T) {
	cfg := config.Config{}
	cfg.Telegram.BotToken = "12345:secret-token"
	cfg.Telegram.AllowedChatID = 42

	details := ConfiguredChannelCapabilityDetails(cfg)
	got := details["telegram"]
	if !strings.Contains(got, "allowed_chat_id=42") {
		t.Fatalf("telegram detail = %q, want allowed chat id", got)
	}
	if strings.Contains(got, "secret-token") {
		t.Fatalf("telegram detail leaked token: %q", got)
	}
}
