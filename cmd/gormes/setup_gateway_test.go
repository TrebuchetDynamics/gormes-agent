package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/cli"
	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/spf13/cobra"
)

func TestSetupGatewayChecklistShowsCorePlatforms(t *testing.T) {
	home := t.TempDir()
	t.Setenv("GORMES_HOME", home)
	clearSetupGatewayTelegramEnv(t)

	fake := &setupCommandFakeSeams{isTTY: false}
	stdout, stderr, err := runSetupTestCommand(t, fake.seams(), "gateway", "--non-interactive")
	if err != nil {
		t.Fatalf("Execute() error = %v stdout=%s stderr=%s", err, stdout, stderr)
	}

	for _, want := range []string{
		"Messaging Platforms",
		"Plan only: no files will be written and no live APIs will be called.",
		"Telegram (telegram): unconfigured",
		"Discord (discord): unconfigured",
		"Slack (slack): unconfigured",
		"WhatsApp (whatsapp): unconfigured",
		"Navivox (navivox): unconfigured",
		"Planned writes:",
	} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("stdout missing %q:\n%s", want, stdout)
		}
	}
	if _, err := os.Stat(config.ConfigPath()); !os.IsNotExist(err) {
		t.Fatalf("non-interactive setup gateway mutated config path %s: %v", config.ConfigPath(), err)
	}
}

func TestSetupGatewayPreselectsConfiguredPlatforms(t *testing.T) {
	home := t.TempDir()
	t.Setenv("GORMES_HOME", home)
	clearSetupGatewayTelegramEnv(t)
	t.Setenv("GORMES_TELEGRAM_TOKEN", "")
	t.Setenv("GORMES_TELEGRAM_BOT_TOKEN", "")
	t.Setenv("TELEGRAM_BOT_TOKEN", "")
	t.Setenv("TELEGRAM_TOKEN", "")
	t.Setenv("GORMES_TELEGRAM_CHAT_ID", "")
	t.Setenv("TELEGRAM_HOME_CHANNEL", "")
	t.Setenv("TELEGRAM_CHAT_ID", "")
	t.Setenv("GORMES_TELEGRAM_ALLOWED_USERS", "")
	t.Setenv("TELEGRAM_ALLOWED_USERS", "")
	t.Setenv("GORMES_DISCORD_TOKEN", "")
	t.Setenv("GORMES_DISCORD_CHANNEL_ID", "")
	t.Setenv("GORMES_SLACK_ENABLED", "")
	t.Setenv("GORMES_SLACK_BOT_TOKEN", "")
	t.Setenv("GORMES_SLACK_APP_TOKEN", "")
	t.Setenv("GORMES_SLACK_CHANNEL_ID", "")
	writeSetupGatewayFixtureConfig(t, `
[telegram]
bot_token = "123456:test-token"
allowed_chat_id = 4242

[discord]
token = "discord-token"
allowed_channel_id = "D42"

[slack]
enabled = true
bot_token = "xoxb-test"
app_token = "xapp-test"
allowed_channel_id = "C42"
`)

	fake := &setupCommandFakeSeams{isTTY: false}
	stdout, stderr, err := runSetupTestCommand(t, fake.seams(), "gateway", "--non-interactive")
	if err != nil {
		t.Fatalf("Execute() error = %v stdout=%s stderr=%s", err, stdout, stderr)
	}
	for _, want := range []string{
		"Telegram (telegram): configured",
		"telegram.home_channel.chat_id=4242",
		"Discord (discord): configured",
		"discord.allowed_channel_id=D42",
		"Slack (slack): configured",
		"slack.allowed_channel_id=C42",
	} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("stdout missing %q:\n%s", want, stdout)
		}
	}
}

func TestSetupGatewayNoSelectionDoesNotMutateConfig(t *testing.T) {
	home := t.TempDir()
	t.Setenv("GORMES_HOME", home)

	fake := &setupCommandFakeSeams{isTTY: true}
	stdout, stderr, err := runSetupTestCommandWithInput(t, fake.seams(), "\n", "gateway")
	if err != nil {
		t.Fatalf("Execute() error = %v stdout=%s stderr=%s", err, stdout, stderr)
	}
	for _, want := range []string{
		"Messaging platforms (comma-separated numbers or ids, blank to keep current):",
		"No platform setup changes selected.",
		"Keeping current gateway platform configuration.",
	} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("stdout missing %q:\n%s", want, stdout)
		}
	}
	if strings.Contains(stdout+stderr, "setup_gateway_platform_row_backed") {
		t.Fatalf("blank selection dispatched platform setup:\nstdout=%s\nstderr=%s", stdout, stderr)
	}
	if _, err := os.Stat(config.ConfigPath()); !os.IsNotExist(err) {
		t.Fatalf("blank setup gateway mutated config path %s: %v", config.ConfigPath(), err)
	}
}

func TestSetupGatewayTelegramWritesTokenAndAllowedChatWithoutLeakingSecret(t *testing.T) {
	home := t.TempDir()
	t.Setenv("GORMES_HOME", home)
	clearSetupGatewayTelegramEnv(t)
	t.Setenv("GORMES_TELEGRAM_TOKEN", "")
	t.Setenv("GORMES_TELEGRAM_BOT_TOKEN", "")
	t.Setenv("TELEGRAM_BOT_TOKEN", "")
	t.Setenv("TELEGRAM_TOKEN", "")
	t.Setenv("GORMES_TELEGRAM_CHAT_ID", "")
	t.Setenv("TELEGRAM_HOME_CHANNEL", "")
	t.Setenv("TELEGRAM_CHAT_ID", "")

	const token = "123456:setup-secret-token"
	fake := &setupCommandFakeSeams{isTTY: true}
	stdout, stderr, err := runSetupTestCommandWithInput(t, fake.seams(), "telegram\n"+token+"\n4242\n", "gateway")
	if err != nil {
		t.Fatalf("Execute() error = %v stdout=%s stderr=%s", err, stdout, stderr)
	}
	if strings.Contains(stdout+stderr, token) {
		t.Fatalf("setup gateway leaked Telegram token:\nstdout=%s\nstderr=%s", stdout, stderr)
	}
	envBody, err := os.ReadFile(config.EnvPath())
	if err != nil {
		t.Fatalf("read env file: %v", err)
	}
	if !strings.Contains(string(envBody), "GORMES_TELEGRAM_BOT_TOKEN="+token) {
		t.Fatalf(".env missing Telegram token env name:\n%s", envBody)
	}
	if data, err := os.ReadFile(config.ConfigPath()); err == nil && strings.Contains(string(data), token) {
		t.Fatalf("config.toml leaked Telegram token:\n%s", data)
	}
	cfg, err := config.Load(nil)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if cfg.Telegram.BotToken != token {
		t.Fatalf("Telegram.BotToken = %q, want configured token", cfg.Telegram.BotToken)
	}
	if cfg.Telegram.AllowedChatID != 4242 {
		t.Fatalf("Telegram.AllowedChatID = %d, want 4242", cfg.Telegram.AllowedChatID)
	}
	if cfg.Telegram.FirstRunDiscovery {
		t.Fatalf("Telegram.FirstRunDiscovery = true, want false when chat ID is configured")
	}
}

func TestSetupGatewayTelegramBlankFreshTokenFailsWithoutWritingChannelConfig(t *testing.T) {
	home := t.TempDir()
	t.Setenv("GORMES_HOME", home)
	clearSetupGatewayTelegramEnv(t)
	t.Setenv("GORMES_TELEGRAM_TOKEN", "")
	t.Setenv("GORMES_TELEGRAM_BOT_TOKEN", "")
	t.Setenv("TELEGRAM_BOT_TOKEN", "")
	t.Setenv("TELEGRAM_TOKEN", "")
	t.Setenv("GORMES_TELEGRAM_CHAT_ID", "")
	t.Setenv("TELEGRAM_HOME_CHANNEL", "")
	t.Setenv("TELEGRAM_CHAT_ID", "")

	fake := &setupCommandFakeSeams{isTTY: true}
	stdout, stderr, err := runSetupTestCommandWithInput(t, fake.seams(), "telegram\n\n4242\n", "gateway")
	if err == nil {
		t.Fatalf("Execute() error = nil, want missing Telegram token failure stdout=%s stderr=%s", stdout, stderr)
	}
	if strings.Contains(stdout, "Telegram gateway channel configured") {
		t.Fatalf("Telegram setup reported configured despite blank fresh token:\n%s", stdout)
	}
	if configBody, readErr := os.ReadFile(config.ConfigPath()); readErr == nil {
		for _, forbidden := range []string{"[telegram]", "allowed_chat_id", "first_run_discovery"} {
			if strings.Contains(string(configBody), forbidden) {
				t.Fatalf("Telegram setup wrote channel config despite blank fresh token:\n%s", configBody)
			}
		}
	} else if !os.IsNotExist(readErr) {
		t.Fatalf("read config: %v", readErr)
	}
}

func TestSetupGatewayDiscordWritesTokenAndAllowedChannelWithoutLeakingSecret(t *testing.T) {
	home := t.TempDir()
	t.Setenv("GORMES_HOME", home)
	t.Setenv("GORMES_DISCORD_TOKEN", "")
	t.Setenv("GORMES_DISCORD_CHANNEL_ID", "")

	const token = "discord-setup-secret-token"
	fake := &setupCommandFakeSeams{isTTY: true}
	stdout, stderr, err := runSetupTestCommandWithInput(t, fake.seams(), "discord\n"+token+"\nD4242\n", "gateway")
	if err != nil {
		t.Fatalf("Execute() error = %v stdout=%s stderr=%s", err, stdout, stderr)
	}
	if strings.Contains(stdout+stderr, token) {
		t.Fatalf("setup gateway leaked Discord token:\nstdout=%s\nstderr=%s\nerr=%v", stdout, stderr, err)
	}
	envBody, err := os.ReadFile(config.EnvPath())
	if err != nil {
		t.Fatalf("read env file: %v", err)
	}
	if !strings.Contains(string(envBody), "GORMES_DISCORD_TOKEN="+token) {
		t.Fatalf(".env missing Discord token env name:\n%s", envBody)
	}
	if data, err := os.ReadFile(config.ConfigPath()); err == nil && strings.Contains(string(data), token) {
		t.Fatalf("config.toml leaked Discord token:\n%s", data)
	}
	cfg, err := config.Load(nil)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if cfg.Discord.Token != token {
		t.Fatalf("Discord.Token = %q, want configured token", cfg.Discord.Token)
	}
	if cfg.Discord.AllowedChannelID != "D4242" {
		t.Fatalf("Discord.AllowedChannelID = %q, want D4242", cfg.Discord.AllowedChannelID)
	}
	if cfg.Discord.FirstRunDiscovery {
		t.Fatalf("Discord.FirstRunDiscovery = true, want false when channel ID is configured")
	}
}

func TestSetupGatewayDiscordBlankFreshTokenFailsWithoutWritingChannelConfig(t *testing.T) {
	home := t.TempDir()
	t.Setenv("GORMES_HOME", home)
	t.Setenv("GORMES_DISCORD_TOKEN", "")
	t.Setenv("GORMES_DISCORD_CHANNEL_ID", "")

	fake := &setupCommandFakeSeams{isTTY: true}
	stdout, stderr, err := runSetupTestCommandWithInput(t, fake.seams(), "discord\n\nD4242\n", "gateway")
	if err == nil {
		t.Fatalf("Execute() error = nil, want missing Discord token failure stdout=%s stderr=%s", stdout, stderr)
	}
	if strings.Contains(stdout, "Discord gateway channel configured") {
		t.Fatalf("Discord setup reported configured despite blank fresh token:\n%s", stdout)
	}
	if configBody, readErr := os.ReadFile(config.ConfigPath()); readErr == nil {
		for _, forbidden := range []string{"[discord]", "allowed_channel_id", "first_run_discovery"} {
			if strings.Contains(string(configBody), forbidden) {
				t.Fatalf("Discord setup wrote channel config despite blank fresh token:\n%s", configBody)
			}
		}
	} else if !os.IsNotExist(readErr) {
		t.Fatalf("read config: %v", readErr)
	}
}

func TestSetupGatewayDiscordWritesNumericSnowflakeChannelAsString(t *testing.T) {
	home := t.TempDir()
	t.Setenv("GORMES_HOME", home)
	t.Setenv("GORMES_DISCORD_TOKEN", "")
	t.Setenv("GORMES_DISCORD_CHANNEL_ID", "")

	const token = "discord-snowflake-secret"
	const channelID = "123456789012345678"
	fake := &setupCommandFakeSeams{isTTY: true}
	stdout, stderr, err := runSetupTestCommandWithInput(t, fake.seams(), "discord\n"+token+"\n"+channelID+"\n", "gateway")
	if err != nil {
		t.Fatalf("Execute() error = %v stdout=%s stderr=%s", err, stdout, stderr)
	}
	cfg, err := config.Load(nil)
	if err != nil {
		t.Fatalf("load config after numeric Discord channel setup: %v", err)
	}
	if cfg.Discord.AllowedChannelID != channelID {
		t.Fatalf("Discord.AllowedChannelID = %q, want %q", cfg.Discord.AllowedChannelID, channelID)
	}
	configBody, err := os.ReadFile(config.ConfigPath())
	if err != nil {
		t.Fatalf("read config: %v", err)
	}
	if !strings.Contains(string(configBody), `allowed_channel_id = "`+channelID+`"`) &&
		!strings.Contains(string(configBody), `allowed_channel_id = '`+channelID+`'`) {
		t.Fatalf("Discord allowed_channel_id was not written as a TOML string:\n%s", configBody)
	}
}

func TestSetupGatewaySlackWritesTokensAndAllowedChannelWithoutLeakingSecrets(t *testing.T) {
	home := t.TempDir()
	t.Setenv("GORMES_HOME", home)
	t.Setenv("GORMES_SLACK_ENABLED", "")
	t.Setenv("GORMES_SLACK_BOT_TOKEN", "")
	t.Setenv("GORMES_SLACK_APP_TOKEN", "")
	t.Setenv("GORMES_SLACK_CHANNEL_ID", "")

	const botToken = "xoxb-slack-setup-secret"
	const appToken = "xapp-slack-setup-secret"
	fake := &setupCommandFakeSeams{isTTY: true}
	stdout, stderr, err := runSetupTestCommandWithInput(t, fake.seams(), "slack\n"+botToken+"\n"+appToken+"\nC4242\n", "gateway")
	if err != nil {
		t.Fatalf("Execute() error = %v stdout=%s stderr=%s", err, stdout, stderr)
	}
	for _, secret := range []string{botToken, appToken} {
		if strings.Contains(stdout+stderr, secret) {
			t.Fatalf("setup gateway leaked Slack token %q:\nstdout=%s\nstderr=%s", secret, stdout, stderr)
		}
	}
	envBody, err := os.ReadFile(config.EnvPath())
	if err != nil {
		t.Fatalf("read env file: %v", err)
	}
	for _, want := range []string{
		"GORMES_SLACK_BOT_TOKEN=" + botToken,
		"GORMES_SLACK_APP_TOKEN=" + appToken,
	} {
		if !strings.Contains(string(envBody), want) {
			t.Fatalf(".env missing %q:\n%s", want, envBody)
		}
	}
	if data, err := os.ReadFile(config.ConfigPath()); err == nil && (strings.Contains(string(data), botToken) || strings.Contains(string(data), appToken)) {
		t.Fatalf("config.toml leaked Slack token:\n%s", data)
	}
	cfg, err := config.Load(nil)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if !cfg.Slack.Enabled {
		t.Fatal("Slack.Enabled = false, want true")
	}
	if cfg.Slack.BotToken != botToken {
		t.Fatalf("Slack.BotToken = %q, want configured bot token", cfg.Slack.BotToken)
	}
	if cfg.Slack.AppToken != appToken {
		t.Fatalf("Slack.AppToken = %q, want configured app token", cfg.Slack.AppToken)
	}
	if cfg.Slack.AllowedChannelID != "C4242" {
		t.Fatalf("Slack.AllowedChannelID = %q, want C4242", cfg.Slack.AllowedChannelID)
	}
	if cfg.Slack.FirstRunDiscovery {
		t.Fatalf("Slack.FirstRunDiscovery = true, want false when channel ID is configured")
	}
}

func TestSetupGatewaySlackBlankTokensDoesNotEnableOrReportConfigured(t *testing.T) {
	home := t.TempDir()
	t.Setenv("GORMES_HOME", home)
	t.Setenv("GORMES_SLACK_ENABLED", "")
	t.Setenv("GORMES_SLACK_BOT_TOKEN", "")
	t.Setenv("GORMES_SLACK_APP_TOKEN", "")
	t.Setenv("GORMES_SLACK_CHANNEL_ID", "")

	fake := &setupCommandFakeSeams{isTTY: true}
	stdout, stderr, err := runSetupTestCommandWithInput(t, fake.seams(), "slack\n\n\nC4242\n", "gateway")
	if err == nil {
		t.Fatalf("Execute() error = nil, want missing Slack token failure stdout=%s stderr=%s", stdout, stderr)
	}
	if strings.Contains(stdout, "Slack gateway channel configured") {
		t.Fatalf("Slack setup reported configured despite blank tokens:\n%s", stdout)
	}
	cfg, loadErr := config.Load(nil)
	if loadErr != nil {
		t.Fatalf("load config: %v", loadErr)
	}
	if cfg.Slack.Enabled || cfg.Slack.AllowedChannelID != "" || cfg.Slack.FirstRunDiscovery {
		t.Fatalf("Slack config = %+v, want not enabled and no channel/discovery state", cfg.Slack)
	}
}

func TestSetupGatewaySlackPartialTokensDoNotEnableOrReportConfigured(t *testing.T) {
	for _, tc := range []struct {
		name     string
		input    string
		envKey   string
		envValue string
	}{
		{name: "bot only", input: "slack\nxoxb-only\n\nC4242\n", envKey: "GORMES_SLACK_APP_TOKEN"},
		{name: "app only", input: "slack\n\nxapp-only\nC4242\n", envKey: "GORMES_SLACK_BOT_TOKEN"},
		{name: "existing bot only", input: "slack\n\n\nC4242\n", envKey: "GORMES_SLACK_BOT_TOKEN", envValue: "xoxb-existing"},
		{name: "existing app only", input: "slack\n\n\nC4242\n", envKey: "GORMES_SLACK_APP_TOKEN", envValue: "xapp-existing"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			home := t.TempDir()
			t.Setenv("GORMES_HOME", home)
			t.Setenv("GORMES_SLACK_ENABLED", "")
			t.Setenv("GORMES_SLACK_BOT_TOKEN", "")
			t.Setenv("GORMES_SLACK_APP_TOKEN", "")
			t.Setenv("GORMES_SLACK_CHANNEL_ID", "")
			if tc.envKey != "" {
				t.Setenv(tc.envKey, tc.envValue)
			}

			fake := &setupCommandFakeSeams{isTTY: true}
			stdout, stderr, err := runSetupTestCommandWithInput(t, fake.seams(), tc.input, "gateway")
			if err == nil {
				t.Fatalf("Execute() error = nil, want partial Slack token failure stdout=%s stderr=%s", stdout, stderr)
			}
			for _, leaked := range []string{"xoxb-only", "xapp-only", "xoxb-existing", "xapp-existing"} {
				if strings.Contains(stdout+stderr+err.Error(), leaked) {
					t.Fatalf("partial Slack token failure leaked %q:\nstdout=%s\nstderr=%s\nerr=%v", leaked, stdout, stderr, err)
				}
			}
			if strings.Contains(stdout, "Slack gateway channel configured") {
				t.Fatalf("Slack setup reported configured despite partial tokens:\n%s", stdout)
			}
			cfg, loadErr := config.Load(nil)
			if loadErr != nil {
				t.Fatalf("load config: %v", loadErr)
			}
			if cfg.Slack.Enabled || cfg.Slack.AllowedChannelID != "" || cfg.Slack.FirstRunDiscovery {
				t.Fatalf("Slack config = %+v, want not enabled and no channel/discovery state", cfg.Slack)
			}
		})
	}
}

func TestSetupGatewayBubbleTeaNavivoxSelectionRunsNativeSetup(t *testing.T) {
	home := t.TempDir()
	t.Setenv("GORMES_HOME", home)

	fake := &setupCommandFakeSeams{isTTY: true}
	seams := fake.seams()
	seams.RunGatewaySetupWizard = func(*cobra.Command, config.Config) (setupGatewayWizardResult, error) {
		return setupGatewayWizardResult{SelectedPlatforms: []string{"navivox"}, BubbleTea: true}, nil
	}
	stdout, stderr, err := runSetupTestCommandWithInput(t, seams, "y\n\n\n\n\n\n", "gateway")
	if err != nil {
		t.Fatalf("Execute() error = %v stdout=%s stderr=%s", err, stdout, stderr)
	}
	for _, want := range []string{"Navivox Gateway Channel", "Navivox gateway channel configured.", "Pairing QR image:"} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("stdout missing %q:\n%s", want, stdout)
		}
	}
	if strings.Contains(stdout+stderr, "Navivox Bubble Tea setup is not shipped") {
		t.Fatalf("Navivox Bubble Tea selection fell through to row-backed fallback:\nstdout=%s\nstderr=%s", stdout, stderr)
	}
	cfg, err := config.Load(nil)
	if err != nil {
		t.Fatal(err)
	}
	if !cfg.Navivox.Enabled || cfg.Navivox.Token == "" {
		t.Fatalf("Navivox config = %+v, want enabled with generated token", cfg.Navivox)
	}
}

func TestSetupNavivoxSectionRunsNativeGatewaySetup(t *testing.T) {
	home := t.TempDir()
	t.Setenv("GORMES_HOME", home)

	fake := &setupCommandFakeSeams{isTTY: true}
	stdout, stderr, err := runSetupTestCommandWithInput(t, fake.seams(), "n\n", "navivox")
	if err != nil {
		t.Fatalf("Execute() error = %v stdout=%s stderr=%s", err, stdout, stderr)
	}
	for _, want := range []string{
		"Gormes Setup — Navivox",
		"Navivox Gateway Channel",
		"Navivox gateway channel disabled.",
		"No firewall rules were changed.",
	} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("stdout missing %q:\n%s", want, stdout)
		}
	}
	if strings.Contains(stdout+stderr, "setup_section_unsupported") {
		t.Fatalf("direct navivox setup returned unsupported evidence:\nstdout=%s\nstderr=%s", stdout, stderr)
	}
	cfg, err := config.Load(nil)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Navivox.Enabled {
		t.Fatalf("Navivox enabled = true, want disabled")
	}
}

func TestSetupGatewayNavivoxCanRemainDisabled(t *testing.T) {
	home := t.TempDir()
	t.Setenv("GORMES_HOME", home)

	fake := &setupCommandFakeSeams{isTTY: true}
	stdout, stderr, err := runSetupTestCommandWithInput(t, fake.seams(), "navivox\nn\n", "gateway")
	if err != nil {
		t.Fatalf("Execute() error = %v stdout=%s stderr=%s", err, stdout, stderr)
	}
	for _, want := range []string{
		"Enable Navivox Gateway Channel?",
		"Navivox gateway channel disabled.",
		"No firewall rules were changed.",
	} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("stdout missing %q:\n%s", want, stdout)
		}
	}
	cfg, err := config.Load(nil)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Navivox.Enabled {
		t.Fatalf("Navivox enabled = true, want disabled")
	}
}

func TestSetupGatewayNavivoxLocalModeWritesSafeConfig(t *testing.T) {
	home := t.TempDir()
	t.Setenv("GORMES_HOME", home)

	fake := &setupCommandFakeSeams{isTTY: true}
	stdout, stderr, err := runSetupTestCommandWithInput(t, fake.seams(), "navivox\ny\n\n\n\n\ny\n", "gateway")
	if err != nil {
		t.Fatalf("Execute() error = %v stdout=%s stderr=%s", err, stdout, stderr)
	}
	qrPath := filepath.Join(home, "navivox", "pairing.png")
	for _, want := range []string{
		"Record manual firewall-open intent? [n]:",
		"Navivox gateway channel configured.",
		"Connection",
		"  HTTP: http://127.0.0.1:8765",
		"  WebSocket: ws://127.0.0.1:8765/v1/navivox/stream",
		"  Config:",
		"Pairing",
		"  Token: generated and stored as GORMES_NAVIVOX_TOKEN in:",
		"  Pairing QR image:\n  " + qrPath,
		"  Scan this QR from Navivox:",
		"  QR payload includes the token when required; the raw token is not printed.",
		"Auth rules",
		"  REST: Authorization: Bearer <Navivox token>",
		"Firewall",
		"  Status: no rules were changed by Gormes.",
		"  Operator request: recorded only; open 127.0.0.1:8765 manually if needed.",
		"Get Navivox",
		"  Android app source: https://github.com/TrebuchetDynamics/navivox-app",
		"  Build/run from source: git clone https://github.com/TrebuchetDynamics/navivox-app && cd navivox-app && flutter run",
		"Next steps",
		"  1. Install or open Navivox on Android.",
		"  2. Scan the QR above, or open the QR image from:",
		"  " + qrPath,
		"  3. Configure provider before starting gateway: gormes setup provider",
		"  4. Then start gateway: gormes gateway",
	} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("stdout missing %q:\n%s", want, stdout)
		}
	}
	if strings.Contains(stdout, "Pairing QR image: "+qrPath) {
		t.Fatalf("QR path should be on its own line for narrow terminals:\n%s", stdout)
	}
	cfg, err := config.Load(nil)
	if err != nil {
		t.Fatal(err)
	}
	if !cfg.Navivox.Enabled {
		t.Fatal("Navivox enabled = false, want true")
	}
	if cfg.Navivox.BindHost != "127.0.0.1" || cfg.Navivox.Port != 8765 || cfg.Navivox.ExposureMode != "local" {
		t.Fatalf("Navivox config = %+v, want local 127.0.0.1:8765", cfg.Navivox)
	}
	if cfg.Navivox.Token == "" {
		t.Fatal("Navivox token was not generated into the environment")
	}
	if strings.Contains(stdout, cfg.Navivox.Token) {
		t.Fatal("setup output leaked generated Navivox token")
	}
	info, err := os.Stat(qrPath)
	if err != nil {
		t.Fatalf("stat pairing QR image: %v", err)
	}
	if got := info.Mode().Perm(); got != 0o600 {
		t.Fatalf("pairing QR image mode = %v, want 0600 because it embeds the Navivox token", got)
	}
	body, err := os.ReadFile(qrPath)
	if err != nil {
		t.Fatalf("read pairing QR image: %v", err)
	}
	if !bytes.HasPrefix(body, []byte("\x89PNG\r\n\x1a\n")) {
		t.Fatalf("pairing QR image is not a PNG; first bytes=% x", body[:min(len(body), 8)])
	}
}

func TestSetupGatewayNavivoxVPNExposureModesWriteConfig(t *testing.T) {
	for _, tc := range []struct {
		name     string
		mode     string
		bindHost string
	}{
		{name: "wireguard", mode: config.NavivoxExposureWireGuard, bindHost: "10.0.0.1"},
		{name: "vpn", mode: config.NavivoxExposureVPN, bindHost: "10.8.0.5"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			home := t.TempDir()
			t.Setenv("GORMES_HOME", home)

			fake := &setupCommandFakeSeams{isTTY: true}
			input := strings.Join([]string{"navivox", "y", tc.mode, tc.bindHost, "", "", ""}, "\n") + "\n"
			stdout, stderr, err := runSetupTestCommandWithInput(t, fake.seams(), input, "gateway")
			if err != nil {
				t.Fatalf("Execute() error = %v stdout=%s stderr=%s", err, stdout, stderr)
			}
			if !strings.Contains(stdout, "Navivox gateway channel configured.") {
				t.Fatalf("stdout missing configured message:\n%s", stdout)
			}
			cfg, err := config.Load(nil)
			if err != nil {
				t.Fatal(err)
			}
			if cfg.Navivox.ExposureMode != tc.mode || cfg.Navivox.BindHost != tc.bindHost {
				t.Fatalf("Navivox config = %+v, want exposure %s bind %s", cfg.Navivox, tc.mode, tc.bindHost)
			}
			if cfg.Navivox.Token == "" {
				t.Fatal("Navivox token was not generated for token auth")
			}
			if strings.Contains(stdout, cfg.Navivox.Token) {
				t.Fatal("setup output leaked generated Navivox token")
			}
		})
	}
}

func TestSetupGatewaySelectedPlatformDelegatesOrReportsRowBacked(t *testing.T) {
	t.Run("future platform row backed by default", func(t *testing.T) {
		cmd := &cobra.Command{}
		var stdout strings.Builder
		cmd.SetOut(&stdout)
		err := runSetupGatewayPlatform(cmd, "futurechat", func(*cobra.Command) error {
			t.Fatal("unknown platform called WhatsApp setup")
			return nil
		})
		if err != nil {
			t.Fatalf("runSetupGatewayPlatform error = %v stdout=%s", err, stdout.String())
		}
		for _, want := range []string{
			"setup_gateway_platform_row_backed: platform=futurechat recommended_command=\"gormes setup gateway\"",
		} {
			if !strings.Contains(stdout.String(), want) {
				t.Fatalf("stdout missing %q:\n%s", want, stdout.String())
			}
		}
	})

	t.Run("navivox uses native setup", func(t *testing.T) {
		home := t.TempDir()
		t.Setenv("GORMES_HOME", home)

		cmd := &cobra.Command{}
		var stdout strings.Builder
		cmd.SetOut(&stdout)
		cmd.SetIn(strings.NewReader("n\n"))
		err := runSetupGatewayPlatform(cmd, "navivox", func(*cobra.Command) error {
			t.Fatal("navivox platform called WhatsApp setup")
			return nil
		})
		if err != nil {
			t.Fatalf("runSetupGatewayPlatform error = %v stdout=%s", err, stdout.String())
		}
		for _, want := range []string{"Navivox Gateway Channel", "Navivox gateway channel disabled."} {
			if !strings.Contains(stdout.String(), want) {
				t.Fatalf("stdout missing %q:\n%s", want, stdout.String())
			}
		}
		if strings.Contains(stdout.String(), "setup_gateway_platform_row_backed") {
			t.Fatalf("navivox fell through to row-backed fallback:\n%s", stdout.String())
		}
	})

	t.Run("injected platform handlers", func(t *testing.T) {
		home := t.TempDir()
		t.Setenv("GORMES_HOME", home)

		var called []string
		fake := &setupCommandFakeSeams{isTTY: true}
		fake.runGatewayPlatform = func(cmd *cobra.Command, platform string) error {
			called = append(called, platform)
			cmd.Printf("setup_gateway_platform_delegated: platform=%s\n", platform)
			return nil
		}
		stdout, stderr, err := runSetupTestCommandWithInput(t, fake.seams(), "1,slack\n", "gateway")
		if err != nil {
			t.Fatalf("Execute() error = %v stdout=%s stderr=%s", err, stdout, stderr)
		}
		if strings.Join(called, ",") != "telegram,slack" {
			t.Fatalf("called platforms = %v, want telegram,slack", called)
		}
		for _, want := range []string{
			"setup_gateway_platform_delegated: platform=telegram",
			"setup_gateway_platform_delegated: platform=slack",
		} {
			if !strings.Contains(stdout, want) {
				t.Fatalf("stdout missing %q:\n%s", want, stdout)
			}
		}
	})
}

func TestSetupGatewayDoesNotStartGateway(t *testing.T) {
	home := t.TempDir()
	t.Setenv("GORMES_HOME", home)
	clearSetupGatewayTelegramEnv(t)
	t.Setenv("GORMES_TELEGRAM_TOKEN", "")
	t.Setenv("GORMES_TELEGRAM_BOT_TOKEN", "")
	t.Setenv("TELEGRAM_BOT_TOKEN", "")
	t.Setenv("TELEGRAM_TOKEN", "")
	t.Setenv("GORMES_TELEGRAM_CHAT_ID", "")
	t.Setenv("TELEGRAM_HOME_CHANNEL", "")
	t.Setenv("TELEGRAM_CHAT_ID", "")
	t.Setenv("GORMES_TELEGRAM_ALLOWED_USERS", "")
	t.Setenv("TELEGRAM_ALLOWED_USERS", "")
	t.Setenv("GORMES_SLACK_ENABLED", "")
	t.Setenv("GORMES_SLACK_BOT_TOKEN", "")
	t.Setenv("GORMES_SLACK_APP_TOKEN", "")
	t.Setenv("GORMES_SLACK_CHANNEL_ID", "")

	fake := &setupCommandFakeSeams{isTTY: true}
	stdout, stderr, err := runSetupTestCommandWithInput(t, fake.seams(), "slack\nxoxb-no-start-test\nxapp-no-start-test\n\n", "gateway")
	if err != nil {
		t.Fatalf("Execute() error = %v stdout=%s stderr=%s", err, stdout, stderr)
	}
	for _, path := range []string{
		config.SessionDBPath(),
		config.MemoryDBPath(),
		config.GatewayRuntimeStatusPath(),
	} {
		if _, err := os.Stat(path); err == nil {
			t.Fatalf("setup gateway opened runtime/startup artifact %s\nstdout=%s", path, stdout)
		} else if !os.IsNotExist(err) {
			t.Fatalf("stat runtime/startup artifact %s: %v", path, err)
		}
	}
}

func TestSetupGatewayWhatsAppSelectionRoutesThroughWhatsAppSetupSeam(t *testing.T) {
	home := t.TempDir()
	t.Setenv("GORMES_HOME", home)

	whatsAppCalls := 0
	fake := &setupCommandFakeSeams{isTTY: true}
	seams := fake.seams()
	seams.RunWhatsAppSetup = func(cmd *cobra.Command) error {
		whatsAppCalls++
		cmd.Println("whatsapp seam reached")
		return nil
	}

	stdout, stderr, err := runSetupTestCommandWithInput(t, seams, "whatsapp\n", "gateway")
	if err != nil {
		t.Fatalf("Execute() error = %v stdout=%s stderr=%s", err, stdout, stderr)
	}
	if whatsAppCalls != 1 {
		t.Fatalf("RunWhatsAppSetup calls = %d, want 1", whatsAppCalls)
	}
	for _, forbidden := range []string{"setup_gateway_platform_row_backed", "Slack bot token", "Discord bot token", "Telegram bot token"} {
		if strings.Contains(stdout+stderr, forbidden) {
			t.Fatalf("setup gateway WhatsApp used wrong path %q:\nstdout=%s\nstderr=%s", forbidden, stdout, stderr)
		}
	}
	if !strings.Contains(stdout, "whatsapp seam reached") {
		t.Fatalf("stdout missing WhatsApp seam output:\n%s", stdout)
	}
}

func TestSetupGatewayWhatsAppDefaultRendersPlanWithoutLivePairing(t *testing.T) {
	home := t.TempDir()
	t.Setenv("GORMES_HOME", home)

	fake := &setupCommandFakeSeams{isTTY: true}
	stdout, stderr, err := runSetupTestCommandWithInput(t, fake.seams(), "whatsapp\n", "gateway")
	if err != nil {
		t.Fatalf("Execute() error = %v stdout=%s stderr=%s", err, stdout, stderr)
	}
	for _, want := range []string{
		"WhatsApp pairing setup",
		"Run without --plan to start the live QR pairing wizard.",
		"Start messaging with: gormes gateway",
	} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("stdout missing %q:\n%s", want, stdout)
		}
	}
	if strings.Contains(stdout+stderr, "Installing WhatsApp bridge dependencies") ||
		strings.Contains(stdout+stderr, "WhatsApp paired successfully") {
		t.Fatalf("default setup WhatsApp path launched live pairing work:\nstdout=%s\nstderr=%s", stdout, stderr)
	}
	if _, err := os.Stat(config.EnvPath()); err == nil {
		t.Fatalf("plan-only WhatsApp setup wrote dotenv file at %s", config.EnvPath())
	} else if !os.IsNotExist(err) {
		t.Fatalf("stat dotenv: %v", err)
	}
}

func TestSetupQuickWhatsAppTargetRoutesThroughWhatsAppSetupSeam(t *testing.T) {
	home := t.TempDir()
	t.Setenv("GORMES_HOME", home)
	t.Setenv("GORMES_API_KEY", "sk-whatsapp-quick-test")
	writeSetupGatewayFixtureConfig(t, `
[hermes]
provider = "openai"
endpoint = "https://api.openai.com/v1"
model = "gpt-4o-mini"
`)

	var events []string
	fake := &setupCommandFakeSeams{
		isTTY:   true,
		current: cli.ProviderModel{Provider: "openai", Model: "gpt-4o-mini"},
	}
	seams := fake.seams()
	seams.RunWhatsAppSetup = func(*cobra.Command) error {
		events = append(events, "whatsapp")
		return nil
	}
	seams.RunGatewayPlatform = func(*cobra.Command, string) error {
		t.Fatal("quick WhatsApp target used generic gateway platform setup")
		return nil
	}
	seams.RunSetupGateway = func(*cobra.Command, bool) error {
		t.Fatal("quick WhatsApp target started gateway setup section")
		return nil
	}
	seams.RunProviderLiveTest = func(*cobra.Command) error {
		events = append(events, "live-test")
		return nil
	}
	seams.LaunchChat = func(*cobra.Command) error {
		t.Fatal("quick WhatsApp target launched chat/TUI")
		return nil
	}

	stdout, stderr, err := runSetupTestCommand(t, seams, "--quick", "--target", "whatsapp")
	if err != nil {
		t.Fatalf("Execute() error = %v stdout=%s stderr=%s", err, stdout, stderr)
	}
	if got, want := strings.Join(events, ","), "whatsapp,live-test"; got != want {
		t.Fatalf("events = %s, want %s\nstdout=%s", got, want, stdout)
	}
	if !strings.Contains(stdout, "Channel setup checked. Start messaging with: gormes gateway") {
		t.Fatalf("stdout missing channel handoff:\n%s", stdout)
	}
	for _, path := range []string{
		config.SessionDBPath(),
		config.MemoryDBPath(),
		config.GatewayRuntimeStatusPath(),
	} {
		if _, err := os.Stat(path); err == nil {
			t.Fatalf("quick WhatsApp target opened runtime/startup artifact %s\nstdout=%s", path, stdout)
		} else if !os.IsNotExist(err) {
			t.Fatalf("stat runtime/startup artifact %s: %v", path, err)
		}
	}
}

func TestSetupQuickNonInteractiveWhatsAppTargetPrintsCommandWithoutPairing(t *testing.T) {
	home := t.TempDir()
	t.Setenv("GORMES_HOME", home)
	t.Setenv("GORMES_API_KEY", "sk-whatsapp-noninteractive-test")
	writeSetupGatewayFixtureConfig(t, `
[hermes]
provider = "openai"
endpoint = "https://api.openai.com/v1"
model = "gpt-4o-mini"
`)

	var events []string
	fake := &setupCommandFakeSeams{
		isTTY:   false,
		current: cli.ProviderModel{Provider: "openai", Model: "gpt-4o-mini"},
	}
	seams := fake.seams()
	seams.RunWhatsAppSetup = func(*cobra.Command) error {
		t.Fatal("non-interactive quick WhatsApp launched pairing setup")
		return nil
	}
	seams.RunGatewayPlatform = func(*cobra.Command, string) error {
		t.Fatal("non-interactive quick WhatsApp used generic gateway platform setup")
		return nil
	}
	seams.RunProviderLiveTest = func(*cobra.Command) error {
		events = append(events, "live-test")
		return nil
	}
	seams.LaunchChat = func(*cobra.Command) error {
		t.Fatal("non-interactive quick WhatsApp launched chat/TUI")
		return nil
	}

	stdout, stderr, err := runSetupTestCommand(t, seams, "--quick", "--non-interactive", "--target", "whatsapp")
	if err != nil {
		t.Fatalf("Execute() error = %v stdout=%s stderr=%s", err, stdout, stderr)
	}
	for _, want := range []string{
		"WhatsApp setup command: gormes whatsapp --plan",
		"Channel setup checked. Start messaging with: gormes gateway",
	} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("stdout missing %q:\n%s", want, stdout)
		}
	}
	if got, want := strings.Join(events, ","), "live-test"; got != want {
		t.Fatalf("events = %s, want %s", got, want)
	}
}

func TestSetupQuickNonInteractiveTelegramTargetPrintsCommandWithoutPrompting(t *testing.T) {
	home := t.TempDir()
	t.Setenv("GORMES_HOME", home)
	t.Setenv("GORMES_API_KEY", "sk-telegram-noninteractive-test")
	clearSetupGatewayTelegramEnv(t)
	t.Setenv("GORMES_TELEGRAM_TOKEN", "")
	t.Setenv("GORMES_TELEGRAM_BOT_TOKEN", "")
	t.Setenv("TELEGRAM_BOT_TOKEN", "")
	t.Setenv("TELEGRAM_TOKEN", "")
	t.Setenv("GORMES_TELEGRAM_CHAT_ID", "")
	t.Setenv("TELEGRAM_HOME_CHANNEL", "")
	t.Setenv("TELEGRAM_CHAT_ID", "")
	writeSetupGatewayFixtureConfig(t, `
[hermes]
provider = "openai"
endpoint = "https://api.openai.com/v1"
model = "gpt-4o-mini"
`)

	var events []string
	fake := &setupCommandFakeSeams{
		isTTY:   false,
		current: cli.ProviderModel{Provider: "openai", Model: "gpt-4o-mini"},
	}
	seams := fake.seams()
	seams.RunGatewayPlatform = func(*cobra.Command, string) error {
		t.Fatal("non-interactive quick Telegram prompted for platform setup")
		return nil
	}
	seams.RunProviderLiveTest = func(*cobra.Command) error {
		events = append(events, "live-test")
		return nil
	}
	seams.LaunchChat = func(*cobra.Command) error {
		t.Fatal("non-interactive quick Telegram launched chat/TUI")
		return nil
	}

	stdout, stderr, err := runSetupTestCommand(t, seams, "--quick", "--non-interactive", "--target", "telegram")
	if err != nil {
		t.Fatalf("Execute() error = %v stdout=%s stderr=%s", err, stdout, stderr)
	}
	for _, want := range []string{
		"Channel setup command: gormes setup gateway",
		"Channel setup checked. Start messaging with: gormes gateway",
	} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("stdout missing %q:\n%s", want, stdout)
		}
	}
	if got, want := strings.Join(events, ","), "live-test"; got != want {
		t.Fatalf("events = %s, want %s", got, want)
	}
}

func TestSetupQuickTelegramTargetSkipsConfiguredChannelSetup(t *testing.T) {
	home := t.TempDir()
	t.Setenv("GORMES_HOME", home)
	t.Setenv("GORMES_API_KEY", "sk-telegram-ready-test")
	clearSetupGatewayTelegramEnv(t)
	t.Setenv("GORMES_TELEGRAM_TOKEN", "")
	t.Setenv("GORMES_TELEGRAM_BOT_TOKEN", "")
	t.Setenv("TELEGRAM_BOT_TOKEN", "")
	t.Setenv("TELEGRAM_TOKEN", "")
	t.Setenv("GORMES_TELEGRAM_CHAT_ID", "")
	t.Setenv("TELEGRAM_HOME_CHANNEL", "")
	t.Setenv("TELEGRAM_CHAT_ID", "")
	writeSetupGatewayFixtureConfig(t, `
[hermes]
provider = "openai"
endpoint = "https://api.openai.com/v1"
model = "gpt-4o-mini"

[telegram]
bot_token = "123456:ready-token"
allowed_chat_id = 4242
`)

	var events []string
	fake := &setupCommandFakeSeams{
		isTTY:   true,
		current: cli.ProviderModel{Provider: "openai", Model: "gpt-4o-mini"},
	}
	seams := fake.seams()
	seams.RunGatewayPlatform = func(*cobra.Command, string) error {
		t.Fatal("quick Telegram target prompted for already configured platform setup")
		return nil
	}
	seams.RunProviderLiveTest = func(*cobra.Command) error {
		events = append(events, "live-test")
		return nil
	}
	seams.LaunchChat = func(*cobra.Command) error {
		t.Fatal("quick Telegram target launched chat/TUI")
		return nil
	}

	stdout, stderr, err := runSetupTestCommand(t, seams, "--quick", "--target", "telegram")
	if err != nil {
		t.Fatalf("Execute() error = %v stdout=%s stderr=%s", err, stdout, stderr)
	}
	if strings.Contains(stdout, "Channel setup command:") {
		t.Fatalf("stdout included channel setup guidance for configured Telegram:\n%s", stdout)
	}
	if !strings.Contains(stdout, "Channel setup checked. Start messaging with: gormes gateway") {
		t.Fatalf("stdout missing channel handoff:\n%s", stdout)
	}
	if got, want := strings.Join(events, ","), "live-test"; got != want {
		t.Fatalf("events = %s, want %s", got, want)
	}
}

func TestSetupGatewaySectionRoutesThroughGatewaySeam(t *testing.T) {
	home := t.TempDir()
	t.Setenv("GORMES_HOME", home)

	gatewayCalls := 0
	fake := &setupCommandFakeSeams{isTTY: true}
	fake.runSetupGateway = func(cmd *cobra.Command, nonInteractive bool) error {
		gatewayCalls++
		if nonInteractive {
			t.Fatal("interactive gateway setup was marked non-interactive")
		}
		cmd.Println("gateway seam reached")
		return nil
	}

	stdout, stderr, err := runSetupTestCommand(t, fake.seams(), "gateway")
	if err != nil {
		t.Fatalf("Execute() error = %v stdout=%s stderr=%s", err, stdout, stderr)
	}
	if gatewayCalls != 1 {
		t.Fatalf("RunSetupGateway calls = %d, want 1", gatewayCalls)
	}
	if !strings.Contains(stdout, "gateway seam reached") {
		t.Fatalf("stdout missing gateway seam output:\n%s", stdout)
	}
}

func writeSetupGatewayFixtureConfig(t *testing.T, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(config.ConfigPath()), 0o700); err != nil {
		t.Fatalf("mkdir config home: %v", err)
	}
	if err := os.WriteFile(config.ConfigPath(), []byte(body), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
}
