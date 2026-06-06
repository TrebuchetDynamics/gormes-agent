package gormescli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	setupwizard "github.com/TrebuchetDynamics/gormes-agent/internal/tui/wizard"
	"github.com/spf13/cobra"
)

func TestSetupGatewayPlanPrintsRedactedChannelStatusesWithoutWrites(t *testing.T) {
	home := t.TempDir()
	t.Setenv("GORMES_HOME", home)
	clearSetupGatewayTelegramEnv(t)
	const token = "123456:abcdefghijklmnopqrstuvwxyzABCDE"
	t.Setenv("GORMES_TELEGRAM_BOT_TOKEN", token)
	writeSetupGatewayFixtureConfig(t, `
[telegram]
allowed_user_ids = [6586915095]

[telegram.home_channel]
chat_id = "-1001234567890"
`)

	fake := &setupCommandFakeSeams{isTTY: false}
	stdout, stderr, err := runSetupTestCommand(t, fake.seams(), "gateway", "--plan")
	if err != nil {
		t.Fatalf("Execute() error = %v stdout=%s stderr=%s", err, stdout, stderr)
	}
	for _, want := range []string{
		"Messaging Platforms",
		"Plan only: no files will be written and no live APIs will be called.",
		"Telegram",
		"configured",
		"telegram.bot_token=[REDACTED]",
		"telegram.home_channel.chat_id=-1001234567890",
		"Discord",
		"Slack",
		"WhatsApp",
		"Navivox",
		"Gateway action:",
	} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("stdout missing %q:\n%s", want, stdout)
		}
	}
	if strings.Contains(stdout+stderr, token) {
		t.Fatalf("setup gateway --plan leaked Telegram token:\nstdout=%s\nstderr=%s", stdout, stderr)
	}
	if _, err := os.Stat(config.EnvPath()); err == nil {
		t.Fatalf("--plan wrote dotenv file at %s", config.EnvPath())
	} else if !os.IsNotExist(err) {
		t.Fatalf("stat dotenv: %v", err)
	}
}

func TestSetupGatewayInteractiveUsesBubbleTeaWizard(t *testing.T) {
	home := t.TempDir()
	t.Setenv("GORMES_HOME", home)
	clearSetupGatewayTelegramEnv(t)

	wizardCalls := 0
	fake := &setupCommandFakeSeams{isTTY: true}
	fake.runGatewaySetupWizard = func(cmd *cobra.Command, cfg config.Config) (setupGatewayWizardResult, error) {
		wizardCalls++
		cmd.Println("bubble tea gateway wizard reached")
		return setupGatewayWizardResult{}, nil
	}

	stdout, stderr, err := runSetupTestCommand(t, fake.seams(), "gateway")
	if err != nil {
		t.Fatalf("Execute() error = %v stdout=%s stderr=%s", err, stdout, stderr)
	}
	if wizardCalls != 1 {
		t.Fatalf("wizard calls = %d, want 1", wizardCalls)
	}
	for _, forbidden := range []string{
		"Which platforms would you like to set up?",
		"Messaging platforms (comma-separated numbers or ids",
		"[ ] Telegram",
	} {
		if strings.Contains(stdout, forbidden) {
			t.Fatalf("interactive setup gateway printed legacy raw marker %q:\n%s", forbidden, stdout)
		}
	}
	if !strings.Contains(stdout, "bubble tea gateway wizard reached") {
		t.Fatalf("stdout missing wizard seam output:\n%s", stdout)
	}
	if _, err := os.Stat(config.ConfigPath()); !os.IsNotExist(err) {
		t.Fatalf("wizard cancellation mutated config path %s: %v", config.ConfigPath(), err)
	}
}

func TestSetupTelegramBubbleTeaWizardAvoidsRepeatedIDPrompts(t *testing.T) {
	steps := setupTelegramGatewayWizardSteps(config.TelegramCfg{})
	if got := len(steps); got != 2 {
		t.Fatalf("Telegram setup first screen has %d steps, want token + access policy only", got)
	}
	for _, step := range steps {
		if step.ID == "apply" || step.Kind == setupwizard.KindConfirm || strings.Contains(step.Prompt, "Write these Telegram settings now") {
			t.Fatalf("Telegram setup wizard step %+v should not ask for a second write confirmation", step)
		}
		for _, forbidden := range []string{"Allowed Telegram user IDs", "Home channel chat ID", "Home channel thread ID"} {
			if strings.Contains(step.Prompt, forbidden) {
				t.Fatalf("Telegram setup first screen prompt %q should not ask optional IDs", step.Prompt)
			}
		}
	}

	followup := setupTelegramGatewayAllowedUsersSteps()
	if len(followup) != 1 || followup[0].ID != "allowed_users" {
		t.Fatalf("allowlist follow-up steps = %+v, want exactly allowed_users", followup)
	}
}

func TestSetupGatewayTelegramBubbleTeaReviewWritesStructuredConfig(t *testing.T) {
	home := t.TempDir()
	t.Setenv("GORMES_HOME", home)
	clearSetupGatewayTelegramEnv(t)
	token := "123456:abcdefghijklmnopqrstuvwxyzABCDE"

	fake := &setupCommandFakeSeams{isTTY: true}
	fake.runGatewaySetupWizard = func(*cobra.Command, config.Config) (setupGatewayWizardResult, error) {
		return setupGatewayWizardResult{
			SelectedPlatforms: []string{"telegram"},
			Telegram: &setupTelegramGatewayAnswers{
				Token:        token,
				AccessPolicy: "allowlist",
				AllowedUsers: "6586915095,12345",
				HomeChatID:   "-1001234567890",
				HomeThreadID: "42",
				Apply:        true,
			},
		}, nil
	}

	stdout, stderr, err := runSetupTestCommand(t, fake.seams(), "gateway")
	if err != nil {
		t.Fatalf("Execute() error = %v stdout=%s stderr=%s", err, stdout, stderr)
	}
	if strings.Contains(stdout+stderr, token) {
		t.Fatalf("Telegram setup leaked token:\nstdout=%s\nstderr=%s", stdout, stderr)
	}
	for _, want := range []string{
		"Review Telegram gateway changes",
		"GORMES_TELEGRAM_BOT_TOKEN=[REDACTED]",
		"telegram.allowed_user_ids=2",
		"telegram.home_channel.chat_id=-1001234567890",
		"group guidance: in BotFather, disable privacy when needed; add the bot as admin; after permission changes, remove and re-add the bot to the group.",
		"Telegram gateway channel configured.",
	} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("stdout missing %q:\n%s", want, stdout)
		}
	}

	envBody, err := os.ReadFile(config.EnvPath())
	if err != nil {
		t.Fatalf("read .env: %v", err)
	}
	if !strings.Contains(string(envBody), "GORMES_TELEGRAM_BOT_TOKEN="+token) {
		t.Fatalf(".env missing preferred Telegram token:\n%s", envBody)
	}
	if strings.Contains(string(envBody), "GORMES_TELEGRAM_TOKEN=") {
		t.Fatalf(".env wrote legacy Telegram token name:\n%s", envBody)
	}
	cfgBody, err := os.ReadFile(config.ConfigPath())
	if err != nil {
		t.Fatalf("read config.toml: %v", err)
	}
	got := string(cfgBody)
	for _, want := range []string{
		"[telegram]",
		"allowed_user_ids = [6586915095, 12345]",
		"[telegram.home_channel]",
		"chat_id = '-1001234567890'",
		"thread_id = '42'",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("config.toml missing %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, token) {
		t.Fatalf("config.toml leaked Telegram token:\n%s", got)
	}

	cfg, err := config.Load(nil)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if cfg.Telegram.BotToken != token {
		t.Fatalf("Telegram.BotToken = %q, want configured token", cfg.Telegram.BotToken)
	}
	if cfg.Telegram.HomeChannel.ChatID != "-1001234567890" || cfg.Telegram.HomeChannel.ThreadID != "42" {
		t.Fatalf("Telegram.HomeChannel = %+v, want structured chat/thread", cfg.Telegram.HomeChannel)
	}
}

func TestSetupGatewayTelegramBubbleTeaCancelWritesNothing(t *testing.T) {
	home := t.TempDir()
	t.Setenv("GORMES_HOME", home)
	clearSetupGatewayTelegramEnv(t)
	token := "123456:abcdefghijklmnopqrstuvwxyzABCDE"

	fake := &setupCommandFakeSeams{isTTY: true}
	fake.runGatewaySetupWizard = func(*cobra.Command, config.Config) (setupGatewayWizardResult, error) {
		return setupGatewayWizardResult{
			SelectedPlatforms: []string{"telegram"},
			Telegram: &setupTelegramGatewayAnswers{
				Token:        token,
				AccessPolicy: "allowlist",
				AllowedUsers: "6586915095",
				HomeChatID:   "-1001234567890",
				Apply:        false,
			},
		}, nil
	}

	stdout, stderr, err := runSetupTestCommand(t, fake.seams(), "gateway")
	if err != nil {
		t.Fatalf("Execute() error = %v stdout=%s stderr=%s", err, stdout, stderr)
	}
	if !strings.Contains(stdout, "Telegram setup canceled; no files were written.") {
		t.Fatalf("stdout missing cancel evidence:\n%s", stdout)
	}
	for _, path := range []string{config.EnvPath(), config.ConfigPath()} {
		if _, err := os.Stat(path); err == nil {
			body, _ := os.ReadFile(path)
			t.Fatalf("cancel wrote %s:\n%s", filepath.Base(path), body)
		} else if !os.IsNotExist(err) {
			t.Fatalf("stat %s: %v", path, err)
		}
	}
}

func clearSetupGatewayTelegramEnv(t *testing.T) {
	t.Helper()
	for _, key := range []string{
		"GORMES_TELEGRAM_BOT_TOKEN",
		"GORMES_TELEGRAM_TOKEN",
		"GORMES_TELEGRAM_HOME_CHANNEL",
		"GORMES_TELEGRAM_CHAT_ID",
		"GORMES_TELEGRAM_HOME_CHANNEL_NAME",
		"GORMES_TELEGRAM_HOME_CHANNEL_THREAD_ID",
		"GORMES_TELEGRAM_ALLOWED_USERS",
		"GORMES_TELEGRAM_ALLOWED_CHATS",
		"HERMES_TELEGRAM_BOT_TOKEN",
		"HERMES_TELEGRAM_TOKEN",
		"HERMES_TELEGRAM_HOME_CHANNEL",
		"HERMES_TELEGRAM_CHAT_ID",
		"HERMES_TELEGRAM_HOME_CHANNEL_NAME",
		"HERMES_TELEGRAM_HOME_CHANNEL_THREAD_ID",
		"HERMES_TELEGRAM_ALLOWED_USERS",
		"HERMES_TELEGRAM_ALLOWED_CHATS",
		"TELEGRAM_BOT_TOKEN",
		"TELEGRAM_TOKEN",
		"TELEGRAM_HOME_CHANNEL",
		"TELEGRAM_CHAT_ID",
		"TELEGRAM_HOME_CHANNEL_NAME",
		"TELEGRAM_HOME_CHANNEL_THREAD_ID",
		"TELEGRAM_ALLOWED_USERS",
		"TELEGRAM_ALLOWED_CHATS",
	} {
		t.Setenv(key, "")
	}
}
