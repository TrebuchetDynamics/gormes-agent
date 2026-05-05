package runtime

import (
	"context"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
)

func TestSecretRefRuntimeGatewayActivationResolvesActiveRefsAndIgnoresInactive(t *testing.T) {
	resolver := &fakeGatewaySecretResolver{values: map[string]string{
		"GORMES_API_KEY":        "sk-runtime-provider",
		"GORMES_TELEGRAM_TOKEN": "123456:telegram-token",
	}}
	cfg := config.Config{
		Hermes: config.HermesCfg{
			Endpoint: "https://provider.example/v1",
			APIKey:   "stale-plaintext-provider",
			APIKeyRef: &config.SecretRef{
				Source: config.SecretRefSourceEnv,
				ID:     "GORMES_API_KEY",
			},
		},
		Telegram: config.TelegramCfg{
			BotToken:      "stale-plaintext-telegram",
			AllowedChatID: 42,
			BotTokenRef: &config.SecretRef{
				Source: config.SecretRefSourceEnv,
				ID:     "GORMES_TELEGRAM_TOKEN",
			},
		},
		Slack: config.SlackCfg{
			Enabled: false,
			BotTokenRef: &config.SecretRef{
				Source: config.SecretRefSourceEnv,
				ID:     "MISSING_SLACK_BOT_TOKEN",
			},
		},
	}

	activation, err := ActivateGatewaySecretRefs(context.Background(), cfg, GatewaySecretRuntimeOptions{
		Resolver: resolver,
	})
	if err != nil {
		t.Fatalf("ActivateGatewaySecretRefs: %v", err)
	}

	if activation.Config.Hermes.APIKey != "sk-runtime-provider" {
		t.Fatalf("resolved provider key = %q, want runtime secret", activation.Config.Hermes.APIKey)
	}
	if activation.Config.Telegram.BotToken != "123456:telegram-token" {
		t.Fatalf("resolved telegram token = %q, want runtime secret", activation.Config.Telegram.BotToken)
	}
	if activation.Config.Slack.BotToken != "" {
		t.Fatalf("inactive slack bot token = %q, want empty", activation.Config.Slack.BotToken)
	}
	if resolver.Count("GORMES_API_KEY") != 1 || resolver.Count("GORMES_TELEGRAM_TOKEN") != 1 {
		t.Fatalf("resolver counts = %+v, want each active ref resolved once", resolver.calls)
	}
	if resolver.Count("MISSING_SLACK_BOT_TOKEN") != 0 {
		t.Fatalf("inactive slack ref resolved %d times, want 0", resolver.Count("MISSING_SLACK_BOT_TOKEN"))
	}

	assertSecretRuntimeEntry(t, activation.Snapshot, "hermes.api_key", true, config.SecretRefEvidenceResolved)
	assertSecretRuntimeEntry(t, activation.Snapshot, "telegram.bot_token", true, config.SecretRefEvidenceResolved)
	assertSecretRuntimeEntry(t, activation.Snapshot, "slack.bot_token", false, SecretRuntimeEvidenceIgnoredInactiveSurface)

	rendered := activation.Snapshot.String()
	for _, leak := range []string{"sk-runtime-provider", "123456:telegram-token", "stale-plaintext-provider", "stale-plaintext-telegram"} {
		if strings.Contains(rendered, leak) {
			t.Fatalf("snapshot leaked secret %q:\n%s", leak, rendered)
		}
	}
}

func TestSecretRefRuntimeGatewayActivationFailsClosedForActiveUnresolvedRef(t *testing.T) {
	resolver := &fakeGatewaySecretResolver{}
	cfg := config.Config{
		Hermes: config.HermesCfg{
			Endpoint: "https://provider.example/v1",
			APIKey:   "stale-plaintext-provider",
			APIKeyRef: &config.SecretRef{
				Source: config.SecretRefSourceEnv,
				ID:     "MISSING_API_KEY",
			},
		},
		Telegram: config.TelegramCfg{BotToken: "123456:telegram-token"},
	}

	activation, err := ActivateGatewaySecretRefs(context.Background(), cfg, GatewaySecretRuntimeOptions{
		Resolver: resolver,
	})
	if err == nil {
		t.Fatalf("ActivateGatewaySecretRefs err = nil, activation=%+v", activation)
	}
	if activation.Config.Hermes.APIKey == "stale-plaintext-provider" {
		t.Fatalf("activation fell back to stale plaintext api key")
	}
	if strings.Contains(err.Error(), "stale-plaintext-provider") {
		t.Fatalf("activation error leaked plaintext: %v", err)
	}
	assertSecretRuntimeEntry(t, activation.Snapshot, "hermes.api_key", false, config.SecretRefEvidenceMissing)
}

func TestSecretRefRuntimeGatewayReloadKeepsLastGoodSnapshotOnFailure(t *testing.T) {
	resolver := &fakeGatewaySecretResolver{values: map[string]string{
		"GORMES_API_KEY":        "sk-runtime-provider",
		"GORMES_TELEGRAM_TOKEN": "123456:telegram-token",
	}}
	controller := NewGatewaySecretRuntimeController(GatewaySecretRuntimeOptions{Resolver: resolver})
	goodCfg := config.Config{
		Hermes: config.HermesCfg{
			Endpoint: "https://provider.example/v1",
			APIKeyRef: &config.SecretRef{
				Source: config.SecretRefSourceEnv,
				ID:     "GORMES_API_KEY",
			},
		},
		Telegram: config.TelegramCfg{
			AllowedChatID: 42,
			BotTokenRef: &config.SecretRef{
				Source: config.SecretRefSourceEnv,
				ID:     "GORMES_TELEGRAM_TOKEN",
			},
		},
	}
	activated, err := controller.Activate(context.Background(), goodCfg)
	if err != nil {
		t.Fatalf("Activate good config: %v", err)
	}
	if activated.Snapshot.Generation != 1 || activated.Config.Hermes.APIKey != "sk-runtime-provider" {
		t.Fatalf("good activation = %+v, want generation 1 resolved provider", activated)
	}

	badCfg := goodCfg
	badCfg.Hermes.APIKeyRef = &config.SecretRef{Source: config.SecretRefSourceEnv, ID: "MISSING_API_KEY"}
	failed, err := controller.Reload(context.Background(), badCfg)
	if err == nil {
		t.Fatalf("Reload bad config err = nil, activation=%+v", failed)
	}
	if failed.Snapshot.Generation != 1 {
		t.Fatalf("failed reload generation = %d, want last-good generation 1", failed.Snapshot.Generation)
	}
	if failed.Config.Hermes.APIKey != "sk-runtime-provider" {
		t.Fatalf("failed reload provider key = %q, want last-good resolved key", failed.Config.Hermes.APIKey)
	}
	if failed.Snapshot.Entries["hermes.api_key"].Ref.ID != "GORMES_API_KEY" {
		t.Fatalf("failed reload snapshot = %+v, want last-good SecretRef id", failed.Snapshot.Entries["hermes.api_key"])
	}
	if strings.Contains(failed.Snapshot.String()+err.Error(), "sk-runtime-provider") {
		t.Fatalf("failed reload leaked last-good secret:\nsnapshot=%s\nerr=%v", failed.Snapshot.String(), err)
	}
}

func assertSecretRuntimeEntry(t *testing.T, snapshot SecretRuntimeSnapshot, path string, resolved bool, code string) {
	t.Helper()
	entry, ok := snapshot.Entries[path]
	if !ok {
		t.Fatalf("snapshot missing %s: %+v", path, snapshot.Entries)
	}
	if entry.Resolved != resolved || entry.Evidence.Code != code {
		t.Fatalf("snapshot[%s] = %+v, want resolved=%t code=%s", path, entry, resolved, code)
	}
	if !entry.Evidence.Redacted {
		t.Fatalf("snapshot[%s] evidence not redacted: %+v", path, entry.Evidence)
	}
}

type fakeGatewaySecretResolver struct {
	values map[string]string
	calls  map[string]int
}

func (r *fakeGatewaySecretResolver) ResolveString(ref config.SecretRef) (string, config.SecretRefEvidence, error) {
	if r.calls == nil {
		r.calls = map[string]int{}
	}
	r.calls[ref.ID]++
	ref.Provider = strings.TrimSpace(ref.Provider)
	if ref.Provider == "" {
		ref.Provider = config.DefaultSecretProviderAlias
	}
	evidence := config.SecretRefEvidence{
		Source:   string(ref.Source),
		Provider: ref.Provider,
		ID:       ref.ID,
		Redacted: true,
	}
	if value, ok := r.values[ref.ID]; ok && strings.TrimSpace(value) != "" {
		evidence.Code = config.SecretRefEvidenceResolved
		return value, evidence, nil
	}
	evidence.Code = config.SecretRefEvidenceMissing
	return "", evidence, errFakeGatewaySecretMissing(ref.ID)
}

func (r *fakeGatewaySecretResolver) Count(id string) int {
	if r == nil || r.calls == nil {
		return 0
	}
	return r.calls[id]
}

type errFakeGatewaySecretMissing string

func (e errFakeGatewaySecretMissing) Error() string {
	return "missing secret ref " + string(e)
}
