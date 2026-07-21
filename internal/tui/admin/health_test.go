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

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/core/agenttemplate"
	"github.com/TrebuchetDynamics/gormes-agent/internal/memory"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/doctor"
	"github.com/charmbracelet/lipgloss"
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

func TestAdminHealth_SetupScreenBoundsCrampedLongRows(t *testing.T) {
	long := strings.Repeat("x", 220)
	screen := NewSetupHealthScreen(WithHealthSource(healthSourceFunc(func(context.Context) ([]HealthItem, error) {
		return []HealthItem{
			{ID: "provider", Status: doctor.StatusFail, Title: "provider configuration " + long, Detail: "endpoint missing " + long, Fixable: true},
			{ID: "auth", Status: doctor.StatusFail, Title: "auth credentials " + long, Detail: "credential pool missing " + long, Fixable: true},
		}, nil
	})))
	updated, _ := screen.Update(tea.WindowSizeMsg{Width: 32, Height: 8})
	screen = updated.(*SetupHealthScreen)

	view := screen.View()
	lines := strings.Split(view, "\n")
	if len(lines) > 8 {
		t.Fatalf("setup health view height = %d, want <= 8:\n%s", len(lines), view)
	}
	for _, line := range lines {
		if got := lipgloss.Width(line); got > 32 {
			t.Fatalf("setup health line width %d exceeds 32:\n%q\n\nfull output:\n%s", got, line, view)
		}
	}
	collapsed := strings.Join(strings.Fields(view), " ")
	for _, want := range []string{"Setup health", "provider", "[Fix]", "omitted", "resize"} {
		if !strings.Contains(collapsed, want) {
			t.Fatalf("setup health view missing %q:\n%s", want, view)
		}
	}
}

func TestAdminHealth_ShellForwardsResizeToSetupScreen(t *testing.T) {
	long := strings.Repeat("x", 220)
	screen := NewSetupHealthScreen(WithHealthSource(healthSourceFunc(func(context.Context) ([]HealthItem, error) {
		return []HealthItem{{ID: "provider", Status: doctor.StatusFail, Title: "provider configuration " + long, Detail: "endpoint missing " + long, Fixable: true}}, nil
	})))
	shell := New(screen)
	updated, _ := shell.Update(tea.WindowSizeMsg{Width: 32, Height: 8})
	shell = updated.(*Shell)

	view := shell.View()
	lines := strings.Split(view, "\n")
	if len(lines) > 8 {
		t.Fatalf("shell setup view height = %d, want <= 8:\n%s", len(lines), view)
	}
	for _, line := range lines {
		if got := lipgloss.Width(line); got > 32 {
			t.Fatalf("shell setup line width %d exceeds 32:\n%q\n\nfull output:\n%s", got, line, view)
		}
	}
	collapsed := strings.Join(strings.Fields(view), " ")
	for _, want := range []string{"Setup health", "[Fix]", "omitted", "resize"} {
		if !strings.Contains(collapsed, want) {
			t.Fatalf("shell setup view missing %q:\n%s", want, view)
		}
	}
}

func TestAdminHealth_ShellAppliesExistingSizeWhenSwitchingToSetup(t *testing.T) {
	long := strings.Repeat("x", 220)
	setup := NewSetupHealthScreen(WithHealthSource(healthSourceFunc(func(context.Context) ([]HealthItem, error) {
		return []HealthItem{{ID: "provider", Status: doctor.StatusFail, Title: "provider configuration " + long, Detail: "endpoint missing " + long, Fixable: true}}, nil
	})))
	shell := New(&stubScreen{name: "Other"}, setup)
	updated, _ := shell.Update(tea.WindowSizeMsg{Width: 32, Height: 8})
	shell = updated.(*Shell)
	updated, _ = shell.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'2'}})
	shell = updated.(*Shell)

	view := shell.View()
	for _, line := range strings.Split(view, "\n") {
		if got := lipgloss.Width(line); got > 32 {
			t.Fatalf("switched setup line width %d exceeds 32:\n%q\n\nfull output:\n%s", got, line, view)
		}
	}
	collapsed := strings.Join(strings.Fields(view), " ")
	for _, want := range []string{"Setup health", "[Fix]", "omitted", "resize"} {
		if !strings.Contains(collapsed, want) {
			t.Fatalf("switched setup view missing %q:\n%s", want, view)
		}
	}
}

func TestAdminHealth_ShellProviderFixWizardFitsTerminalHeight(t *testing.T) {
	isolateHealthHome(t)
	screen := NewSetupHealthScreen(WithHealthSource(healthSourceFunc(func(context.Context) ([]HealthItem, error) {
		return []HealthItem{{ID: healthItemProvider, Status: doctor.StatusFail, Title: "no provider", Detail: "hermes.endpoint is not configured", Fixable: true}}, nil
	})))
	shell := New(screen)
	updated, _ := shell.Update(tea.WindowSizeMsg{Width: 32, Height: 8})
	shell = updated.(*Shell)
	updated, _ = shell.Update(tea.KeyMsg{Type: tea.KeyEnter})
	shell = updated.(*Shell)

	view := shell.View()
	lines := strings.Split(view, "\n")
	if len(lines) > 8 {
		t.Fatalf("setup fix shell view height = %d, want <= 8:\n%s", len(lines), view)
	}
	for _, line := range lines {
		if got := lipgloss.Width(line); got > 32 {
			t.Fatalf("setup fix shell line width %d exceeds 32:\n%q\n\nfull output:\n%s", got, line, view)
		}
	}
	collapsed := strings.Join(strings.Fields(view), " ")
	for _, want := range []string{"Provider setup", "OpenAI", "Enter submit"} {
		if !strings.Contains(collapsed, want) {
			t.Fatalf("setup fix shell view missing %q:\n%s", want, view)
		}
	}
}

func TestAdminHealth_ProviderFixWizardBoundsCrampedLongDefaults(t *testing.T) {
	long := strings.Repeat("x", 220)
	cfg := config.Config{}
	cfg.Hermes.Provider = "openai-codex"
	cfg.Hermes.Endpoint = "https://chatgpt.com/backend-api/codex/" + long
	cfg.Hermes.Model = "gpt-5.2-codex-" + long
	fix := newProviderFixState(cfg)
	fix.resize(28, 8)

	view := fix.View()
	assertCrampedSetupView(t, view, 28, 8, []string{"Provider setup", "OpenAI Codex", "Enter submit"})

	done, err := fix.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if err != nil || done {
		t.Fatalf("provider select done=%v err=%v", done, err)
	}
	view = fix.View()
	assertCrampedSetupView(t, view, 28, 8, []string{"Provider setup", "Endpoint URL", "omitted", "resize", "Enter submit"})
}

func assertCrampedSetupView(t *testing.T, view string, width, height int, wants []string) {
	t.Helper()
	lines := strings.Split(view, "\n")
	if len(lines) > height {
		t.Fatalf("view height = %d, want <= %d:\n%s", len(lines), height, view)
	}
	for _, line := range lines {
		if got := lipgloss.Width(line); got > width {
			t.Fatalf("line width %d exceeds %d:\n%q\n\nfull output:\n%s", got, width, line, view)
		}
	}
	collapsed := strings.Join(strings.Fields(view), " ")
	for _, want := range wants {
		if !strings.Contains(collapsed, want) {
			t.Fatalf("view missing %q:\n%s", want, view)
		}
	}
}

func TestAdminHealth_FixActionRunsWizardAndRefreshesRow(t *testing.T) {
	isolateHealthHome(t)

	screen := NewSetupHealthScreen()
	shell := New(screen)
	tm := teatest.NewTestModel(t, shell, teatest.WithInitialTermSize(96, 32))
	uiTimeout := 10 * time.Second

	tm.Send(tea.KeyMsg{Type: tea.KeyDown})  // profile -> provider
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter}) // open fix wizard
	teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
		return bytes.Contains(out, []byte("Provider setup"))
	}, teatest.WithDuration(uiTimeout), teatest.WithCheckInterval(10*time.Millisecond))

	tm.Send(tea.KeyMsg{Type: tea.KeyEnter}) // provider=openai
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter}) // endpoint default
	tm.Type("sk-admin-fix")
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter}) // api key
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter}) // model default

	teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
		return bytes.Contains(out, []byte("✓ provider configured")) &&
			bytes.Contains(out, []byte("✓ auth credentials present"))
	}, teatest.WithDuration(uiTimeout), teatest.WithCheckInterval(10*time.Millisecond))
	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlC})
	tm.WaitFinished(t, teatest.WithFinalTimeout(uiTimeout))

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
