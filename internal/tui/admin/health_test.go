package admin

import (
	"bytes"
	"context"
	"slices"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/exp/teatest"

	"github.com/TrebuchetDynamics/gormes-agent/internal/agenttemplate"
	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/doctor"
	"github.com/TrebuchetDynamics/gormes-agent/internal/memory"
)

func TestAdminHealth_RendersMissingProviderAndAuthCallouts(t *testing.T) {
	isolateHealthHome(t)

	screen := NewSetupHealthScreen()
	got := screen.View()

	for _, want := range []string{
		"Setup health",
		"✗ no provider",
		"✗ no auth credentials",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("health screen missing %q:\n%s", want, got)
		}
	}
	if strings.Count(got, "[Fix]") < 2 {
		t.Fatalf("health screen = %q, want fix actions for provider and auth rows", got)
	}
}

func TestAdminHealth_RendersHealthyChecksWithoutFix(t *testing.T) {
	home := isolateHealthHome(t)
	configureHealthyHome(t, home)

	screen := NewSetupHealthScreen()
	got := screen.View()
	if strings.Contains(got, "[Fix]") {
		t.Fatalf("healthy screen rendered fix action:\n%s", got)
	}
	for _, item := range screen.Items() {
		if item.Status != doctor.StatusPass {
			t.Fatalf("health item %q status = %s, want PASS\n%s", item.ID, item.Status, got)
		}
		if item.Fixable {
			t.Fatalf("health item %q is fixable in healthy state\n%s", item.ID, got)
		}
	}
}

func TestAdminHealth_FixActionRunsWizardAndRefreshesRow(t *testing.T) {
	isolateHealthHome(t)

	screen := NewSetupHealthScreen()
	shell := New(screen)
	tm := teatest.NewTestModel(t, shell, teatest.WithInitialTermSize(96, 32))

	tm.Send(tea.KeyMsg{Type: tea.KeyDown})  // profile -> provider
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter}) // open fix wizard
	teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
		return bytes.Contains(out, []byte("Provider setup"))
	}, teatest.WithDuration(2*time.Second), teatest.WithCheckInterval(10*time.Millisecond))

	tm.Send(tea.KeyMsg{Type: tea.KeyEnter}) // provider=openai
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter}) // endpoint default
	tm.Type("sk-admin-fix")
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter}) // api key
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter}) // model default

	teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
		return bytes.Contains(out, []byte("✓ provider configured")) &&
			bytes.Contains(out, []byte("✓ auth credentials present"))
	}, teatest.WithDuration(2*time.Second), teatest.WithCheckInterval(10*time.Millisecond))
	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlC})
	tm.WaitFinished(t, teatest.WithFinalTimeout(2*time.Second))

	got := screen.View()
	for _, want := range []string{"✓ provider configured", "✓ auth credentials present"} {
		if !strings.Contains(got, want) {
			t.Fatalf("health screen missing %q after fix:\n%s", want, got)
		}
	}

	pool, _, err := config.LoadCredentialPool(config.CredentialPoolOptions{Provider: "openai"})
	if err != nil {
		t.Fatalf("load credential pool: %v", err)
	}
	if !slices.ContainsFunc(pool.Entries(), func(entry config.PooledCredential) bool {
		return entry.AccessToken == "sk-admin-fix"
	}) {
		t.Fatalf("credential pool missing entered key: %+v", pool.Entries())
	}
}

func TestAdminHealth_RefreshKeyRecomputesChecks(t *testing.T) {
	isolateHealthHome(t)

	screen := NewSetupHealthScreen()
	if got := screen.View(); !strings.Contains(got, "✗ no provider") {
		t.Fatalf("initial health screen missing provider failure:\n%s", got)
	}

	mustWriteTOML(t, "hermes.provider", "openai")
	mustWriteTOML(t, "hermes.endpoint", "https://api.openai.com/v1")
	mustWriteTOML(t, "hermes.model", "gpt-test")

	screen = updateHealthScreen(t, screen, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	if got := screen.View(); !strings.Contains(got, "✓ provider configured") {
		t.Fatalf("refresh did not recompute provider row:\n%s", got)
	}
}

func isolateHealthHome(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("GORMES_HOME", home)
	for _, key := range []string{
		"GORMES_ENDPOINT",
		"GORMES_MODEL",
		"GORMES_API_KEY",
		"GORMES_TELEGRAM_TOKEN",
		"TELEGRAM_BOT_TOKEN",
		"TELEGRAM_TOKEN",
		"GORMES_DISCORD_TOKEN",
		"GORMES_SLACK_BOT_TOKEN",
		"GORMES_SLACK_APP_TOKEN",
	} {
		t.Setenv(key, "")
	}
	return home
}

func configureHealthyHome(t *testing.T, home string) {
	t.Helper()
	mustWriteTOML(t, "hermes.provider", "openai")
	mustWriteTOML(t, "hermes.endpoint", "https://api.openai.com/v1")
	mustWriteTOML(t, "hermes.model", "gpt-test")
	if err := config.SaveCredentialPoolEntries(config.CredentialPoolOptions{Provider: "openai"}, []config.PooledCredential{{
		ID:          "openai-manual-1",
		Label:       "admin-test",
		AuthType:    config.CredentialAuthAPIKey,
		Source:      "manual",
		AccessToken: "sk-admin-health",
		LastStatus:  config.CredentialStatusOK,
	}}); err != nil {
		t.Fatalf("save credential pool: %v", err)
	}

	cfg, err := config.Load(nil)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	agent, ok := cfg.Agents.AgentByID(cfg.Agents.DefaultAgentID())
	if !ok {
		t.Fatalf("default agent missing after config load")
	}
	if _, err := agenttemplate.ApplyDefaultTemplates(agenttemplate.WriteOptions{TargetDir: agent.AgentDir}); err != nil {
		t.Fatalf("seed default agent template: %v", err)
	}

	store, err := memory.OpenSqlite(config.MemoryDBPath(), 0, nil)
	if err != nil {
		t.Fatalf("create memory db under %s: %v", home, err)
	}
	if err := store.Close(context.Background()); err != nil {
		t.Fatalf("close memory db: %v", err)
	}

	t.Setenv("GORMES_TELEGRAM_TOKEN", "tg-health-token")
	t.Setenv("GORMES_DISCORD_TOKEN", "discord-health-token")
	t.Setenv("GORMES_SLACK_BOT_TOKEN", "xoxb-health-token")
	t.Setenv("GORMES_SLACK_APP_TOKEN", "xapp-health-token")
}

func mustWriteTOML(t *testing.T, key, value string) {
	t.Helper()
	if err := config.WriteTOMLValue(config.ConfigPath(), key, value); err != nil {
		t.Fatalf("write %s: %v", key, err)
	}
}

func updateHealthScreen(t *testing.T, screen *SetupHealthScreen, msg tea.Msg) *SetupHealthScreen {
	t.Helper()
	next, cmd := screen.Update(msg)
	if cmd != nil {
		next, _ = next.Update(cmd())
	}
	health, ok := next.(*SetupHealthScreen)
	if !ok {
		t.Fatalf("Update returned %T, want *SetupHealthScreen", next)
	}
	return health
}
