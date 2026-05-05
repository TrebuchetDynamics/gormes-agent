package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
)

func TestGatewaySecretRefRuntimeActivationResolvesBeforeStartupUsesConfig(t *testing.T) {
	resolver := &gatewaySecretRefTestResolver{values: map[string]string{
		"GORMES_API_KEY":        "sk-gateway-runtime",
		"GORMES_TELEGRAM_TOKEN": "123456:runtime-telegram",
	}}
	cfg := config.Config{
		Hermes: config.HermesCfg{
			Endpoint: "https://provider.example/v1",
			APIKey:   "stale-provider-key",
			APIKeyRef: &config.SecretRef{
				Source: config.SecretRefSourceEnv,
				ID:     "GORMES_API_KEY",
			},
		},
		Telegram: config.TelegramCfg{
			BotToken:      "stale-telegram-token",
			AllowedChatID: 42,
			BotTokenRef: &config.SecretRef{
				Source: config.SecretRefSourceEnv,
				ID:     "GORMES_TELEGRAM_TOKEN",
			},
		},
	}

	activated, snapshot, err := activateGatewaySecretRuntime(context.Background(), cfg, resolver)
	if err != nil {
		t.Fatalf("activateGatewaySecretRuntime: %v", err)
	}
	if activated.Hermes.APIKey != "sk-gateway-runtime" {
		t.Fatalf("activated provider api key = %q, want resolved runtime key", activated.Hermes.APIKey)
	}
	if activated.Telegram.BotToken != "123456:runtime-telegram" {
		t.Fatalf("activated telegram token = %q, want resolved runtime token", activated.Telegram.BotToken)
	}
	for _, leak := range []string{"sk-gateway-runtime", "123456:runtime-telegram", "stale-provider-key", "stale-telegram-token"} {
		if strings.Contains(snapshot.String()+errString(err), leak) {
			t.Fatalf("gateway SecretRef activation leaked %q\nsnapshot=%s\nerr=%v", leak, snapshot.String(), err)
		}
	}
	if resolver.Count("GORMES_API_KEY") != 1 || resolver.Count("GORMES_TELEGRAM_TOKEN") != 1 {
		t.Fatalf("resolver calls = %+v, want one eager resolution per active ref", resolver.calls)
	}
}

func TestDoctorSecretRefRuntimeStatusReportsResolvedAndInactiveWithoutLeaking(t *testing.T) {
	setupOneshotFlagTestEnv(t)
	t.Setenv("GORMES_PROVIDER_SECRET", "sk-doctor-runtime")
	t.Setenv("GORMES_SLACK_BOT_SECRET", "xoxb-doctor-runtime")
	t.Setenv("GORMES_SLACK_APP_SECRET", "xapp-doctor-runtime")
	writeOneshotFlagConfig(t, []byte(`
[hermes]
endpoint = "https://provider.example/v1"
api_key = "stale-doctor-provider"

[hermes.api_key_ref]
source = "env"
id = "GORMES_PROVIDER_SECRET"

[slack]
enabled = true
allowed_channel_id = "C123"
bot_token = "stale-doctor-slack-bot"
app_token = "stale-doctor-slack-app"

[slack.bot_token_ref]
source = "env"
id = "GORMES_SLACK_BOT_SECRET"

[slack.app_token_ref]
source = "env"
id = "GORMES_SLACK_APP_SECRET"

[discord.token_ref]
source = "env"
id = "MISSING_DISCORD_SECRET"
`))
	if err := os.MkdirAll(filepath.Dir(config.MemoryDBPath()), 0o755); err != nil {
		t.Fatalf("mkdir memory db dir: %v", err)
	}

	cmd := newRootCommandWithRuntime(rootRuntime{})
	stdout, stderr, err := executeOneshotFlagCommand(cmd, "doctor", "--offline")
	if err != nil {
		t.Fatalf("doctor --offline: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	output := stdout + stderr
	for _, want := range []string{
		"[PASS] SecretRef runtime: resolved=3 inactive=1 unavailable=0",
		"hermes.api_key: resolved",
		"slack.bot_token: resolved",
		"slack.app_token: resolved",
		"discord.token: inactive",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("doctor output missing %q:\n%s", want, output)
		}
	}
	for _, leak := range []string{"sk-doctor-runtime", "xoxb-doctor-runtime", "xapp-doctor-runtime", "stale-doctor-provider", "stale-doctor-slack-bot", "stale-doctor-slack-app"} {
		if strings.Contains(output, leak) {
			t.Fatalf("doctor output leaked %q:\n%s", leak, output)
		}
	}
}

type gatewaySecretRefTestResolver struct {
	values map[string]string
	calls  map[string]int
}

func (r *gatewaySecretRefTestResolver) ResolveString(ref config.SecretRef) (string, config.SecretRefEvidence, error) {
	if r.calls == nil {
		r.calls = map[string]int{}
	}
	r.calls[ref.ID]++
	provider := ref.Provider
	if provider == "" {
		provider = config.DefaultSecretProviderAlias
	}
	evidence := config.SecretRefEvidence{
		Source:   string(ref.Source),
		Provider: provider,
		ID:       ref.ID,
		Redacted: true,
	}
	if value, ok := r.values[ref.ID]; ok && strings.TrimSpace(value) != "" {
		evidence.Code = config.SecretRefEvidenceResolved
		return value, evidence, nil
	}
	evidence.Code = config.SecretRefEvidenceMissing
	return "", evidence, errGatewaySecretRefTestMissing(ref.ID)
}

func (r *gatewaySecretRefTestResolver) Count(id string) int {
	if r == nil || r.calls == nil {
		return 0
	}
	return r.calls[id]
}

type errGatewaySecretRefTestMissing string

func (e errGatewaySecretRefTestMissing) Error() string {
	return "missing secret ref " + string(e)
}

func errString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}
