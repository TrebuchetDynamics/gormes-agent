package gateway

import (
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/security"
)

func TestGatewayStartupAllowlistWarning(t *testing.T) {
	cfg := config.Config{
		Telegram: config.TelegramCfg{BotToken: "telegram-token"},
	}

	report := EvaluateStartupSecurity(cfg, emptyGatewaySecurityEnv)
	if !gatewayStartupSecurityHasCode(report, "gateway_allowlist_missing") {
		t.Fatalf("report = %#v, want gateway_allowlist_missing", report)
	}

	cases := []struct {
		name string
		env  map[string]string
	}{
		{name: "signal users", env: map[string]string{"SIGNAL_GROUP_ALLOWED_USERS": "123"}},
		{name: "telegram allow all", env: map[string]string{"TELEGRAM_ALLOW_ALL_USERS": "true"}},
		{name: "gateway allow all", env: map[string]string{"GATEWAY_ALLOW_ALL_USERS": "yes"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			report := EvaluateStartupSecurity(cfg, mapGatewaySecurityEnv(tc.env))
			if gatewayStartupSecurityHasCode(report, "gateway_allowlist_missing") {
				t.Fatalf("report = %#v, want no allowlist warning", report)
			}
		})
	}
}

func TestGatewayStartupAllowlistRecognizesSimpleXEnv(t *testing.T) {
	if StartupAllowlistConfigured(config.Config{}, func(string) string { return "" }) {
		t.Fatal("empty env startup allowlist = true, want false")
	}
	if !StartupAllowlistConfigured(config.Config{}, func(key string) string {
		switch key {
		case "SIMPLEX_WS_URL":
			return "ws://127.0.0.1:5225"
		case "SIMPLEX_ALLOWED_USERS":
			return "contact-42"
		default:
			return ""
		}
	}) {
		t.Fatal("SIMPLEX_ALLOWED_USERS did not satisfy startup allowlist evidence")
	}
	if !StartupAllowAllConfigured(func(key string) string {
		if key == "SIMPLEX_ALLOW_ALL_USERS" {
			return "true"
		}
		return ""
	}) {
		t.Fatal("SIMPLEX_ALLOW_ALL_USERS did not satisfy startup allow-all evidence")
	}
}

func TestGatewayWeakCredentialGuard_DisablesEnabledPlaceholderPlatforms(t *testing.T) {
	cfg := config.Config{
		Telegram: config.TelegramCfg{BotToken: " *** "},
		Discord:  config.DiscordCfg{Token: " changeme ", AllowedChannelID: "C123"},
		Slack: config.SlackCfg{
			Enabled:          true,
			BotToken:         "your_api_key",
			AppToken:         "xapp-real",
			AllowedChannelID: "C123",
		},
	}

	report := EvaluateStartupSecurity(cfg, emptyGatewaySecurityEnv)
	got := report.Config
	if got.Telegram.BotToken != "" {
		t.Fatalf("telegram token = %q, want disabled blank token", got.Telegram.BotToken)
	}
	if got.Discord.Token != "" {
		t.Fatalf("discord token = %q, want disabled blank token", got.Discord.Token)
	}
	if got.Slack.Enabled {
		t.Fatalf("slack enabled = true, want disabled")
	}
	if got.Slack.BotToken != "" {
		t.Fatalf("slack bot token = %q, want blank", got.Slack.BotToken)
	}

	if countGatewayStartupSecurityCode(report, "gateway_weak_credential_disabled") != 3 {
		t.Fatalf("report = %#v, want three weak-credential evidence items", report)
	}
	for _, evidence := range report.Evidence {
		if strings.Contains(evidence.Message, "***") ||
			strings.Contains(evidence.Message, "changeme") ||
			strings.Contains(evidence.Message, "your_api_key") {
			t.Fatalf("evidence leaked placeholder value: %#v", evidence)
		}
	}
}

func TestGatewayWeakCredentialGuard_IgnoresDisabledOrEmptyTokens(t *testing.T) {
	cfg := config.Config{
		Telegram: config.TelegramCfg{},
		Discord:  config.DiscordCfg{Token: " ", AllowedChannelID: "C123"},
		Slack: config.SlackCfg{
			Enabled:          false,
			BotToken:         "placeholder",
			AppToken:         "placeholder",
			AllowedChannelID: "C123",
		},
	}

	report := EvaluateStartupSecurity(cfg, emptyGatewaySecurityEnv)
	if countGatewayStartupSecurityCode(report, "gateway_weak_credential_disabled") != 0 {
		t.Fatalf("report = %#v, want no weak-credential evidence", report)
	}
}

// TestAdvisoryGatewayLogEntryEmitsOnCompromisedPackage proves that the gateway
// advisory log helper returns a non-empty message when the injected seam
// reports a compromised package version (parity intent:
// security_advisories.py gateway_log_message).
func TestAdvisoryGatewayLogEntryEmitsOnCompromisedPackage(t *testing.T) {
	home := t.TempDir()
	installed := func(pkg string) string {
		if pkg == "mistralai" {
			return "2.4.6" // the compromised PyPI release
		}
		return ""
	}
	msg := AdvisoryGatewayLogEntry(home, installed)
	if !strings.Contains(msg, "Security advisory") {
		t.Fatalf("msg = %q, want Security advisory", msg)
	}
	if !strings.Contains(msg, "shai-hulud-2026-05") {
		t.Fatalf("msg = %q, want advisory ID shai-hulud-2026-05", msg)
	}
}

// TestAdvisoryGatewayLogEntryEmptyOnCleanRuntime proves no message is returned
// when NoInstalledPackages (the production Go-binary seam) is used.
func TestAdvisoryGatewayLogEntryEmptyOnCleanRuntime(t *testing.T) {
	home := t.TempDir()
	msg := AdvisoryGatewayLogEntry(home, security.NoInstalledPackages)
	if msg != "" {
		t.Fatalf("msg = %q, want empty for clean runtime", msg)
	}
}

// TestAdvisoryGatewayLogEntrySilentWhenAcked proves that an acknowledged
// advisory produces an empty log entry.
func TestAdvisoryGatewayLogEntrySilentWhenAcked(t *testing.T) {
	home := t.TempDir()
	installed := func(pkg string) string {
		if pkg == "mistralai" {
			return "2.4.6"
		}
		return ""
	}
	store := security.NewAckStore(home)
	if err := store.Ack("shai-hulud-2026-05"); err != nil {
		t.Fatalf("ack: %v", err)
	}
	msg := AdvisoryGatewayLogEntry(home, installed)
	if msg != "" {
		t.Fatalf("msg = %q, want empty after ack", msg)
	}
}

func emptyGatewaySecurityEnv(string) string { return "" }

func mapGatewaySecurityEnv(values map[string]string) func(string) string {
	return func(key string) string {
		return values[key]
	}
}

func gatewayStartupSecurityHasCode(report StartupSecurityReport, code string) bool {
	return countGatewayStartupSecurityCode(report, code) > 0
}

func countGatewayStartupSecurityCode(report StartupSecurityReport, code string) int {
	count := 0
	for _, evidence := range report.Evidence {
		if evidence.Code == code {
			count++
		}
	}
	return count
}
