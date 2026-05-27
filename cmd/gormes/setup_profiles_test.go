package main

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	tea "github.com/charmbracelet/bubbletea"
)

// `gormes setup profiles` is a NEW Gormes-owned section. It must render under
// the SHIPPED boxed `│ Gormes Setup — Profiles │` chrome (reused, no third
// variant), not fall through to the unsupported-section path. Owned
// divergence: Hermes has no setup profiles section — no ~/.hermes/hermes
// wording.
func TestSetupProfilesSectionRendersBoxedHeader(t *testing.T) {
	t.Setenv("GORMES_HOME", t.TempDir())
	fake := &setupCommandFakeSeams{isTTY: false}

	stdout, stderr, _ := runSetupTestCommand(t, fake.seams(), "profiles")

	out := stdout + stderr
	if !strings.Contains(out, "Gormes Setup — Profiles") {
		t.Fatalf("setup profiles: missing boxed header title:\n%s", out)
	}
	if !strings.Contains(out, "┌") || !strings.Contains(out, "│") || !strings.Contains(out, "└") {
		t.Fatalf("setup profiles: header is not the shared box chrome:\n%s", out)
	}
	if strings.Contains(out, "Unsupported setup section") || strings.Contains(out, "unsupported") {
		t.Fatalf("setup profiles must be a known section, not unsupported:\n%s", out)
	}
	for _, forbidden := range []string{"hermes setup", "~/.hermes", "⚕ Hermes", "wrapper"} {
		if strings.Contains(out, forbidden) {
			t.Fatalf("setup profiles leaked Hermes-owned wording %q:\n%s", forbidden, out)
		}
	}
}

// The interactive flow lists known profiles with the active marker and can
// create a new profile, REUSING the real defaultProfileCommandSeams (it must
// not reimplement profile enumeration/creation — proven by the profile dir
// landing under ~/.gormes/profiles via the seam).
func TestSetupProfilesInteractiveListsAndCreates(t *testing.T) {
	home := t.TempDir()
	t.Setenv("GORMES_HOME", home)
	fake := &setupCommandFakeSeams{isTTY: true}

	// create "work", blank profile-select (active default), blank workspace,
	// blank channels (skip persistence this test).
	stdout, stderr, err := runSetupTestCommandWithInput(t, fake.seams(), "work\n\n\n\n", "profiles")
	if err != nil {
		t.Fatalf("setup profiles: Execute() error = %v stderr=%s", err, stderr)
	}

	out := stdout + stderr
	if !strings.Contains(out, "default") || !strings.Contains(out, "(active)") {
		t.Fatalf("must list known profiles with the active marker:\n%s", out)
	}
	if !strings.Contains(out, "work") {
		t.Fatalf("must reflect the created profile 'work':\n%s", out)
	}
	if _, statErr := os.Stat(filepath.Join(config.GormesHome(), "profiles", "work")); statErr != nil {
		t.Fatalf("create must reuse the profile seam (profiles/work dir missing): %v", statErr)
	}
	for _, forbidden := range []string{"~/.hermes", "hermes profile", "wrapper"} {
		if strings.Contains(out, forbidden) {
			t.Fatalf("leaked Hermes-owned wording %q:\n%s", forbidden, out)
		}
	}
}

// Selecting a profile and entering one-or-more workspace dirs persists them
// as a TOML ARRAY into THAT profile's own config.toml via the real
// internal/config writer round-trip, and config.Load reads them back into
// AgentsCfg.Defaults.Workspaces (default profile -> ~/.gormes/config.toml).
func TestSetupProfilesFromProfileScopedEnvCreatesUnderBaseHome(t *testing.T) {
	base := filepath.Join(t.TempDir(), ".gormes")
	activeProfileRoot := filepath.Join(base, "profiles", "active")
	if err := os.MkdirAll(activeProfileRoot, 0o700); err != nil {
		t.Fatalf("mkdir active profile: %v", err)
	}
	t.Setenv("GORMES_HOME", activeProfileRoot)
	fake := &setupCommandFakeSeams{isTTY: true}

	stdout, stderr, err := runSetupTestCommandWithInput(t, fake.seams(), "work\nwork\n/srv/work\n\n", "profiles")
	if err != nil {
		t.Fatalf("setup profiles: Execute() error = %v stdout=%s stderr=%s", err, stdout, stderr)
	}
	wantProfile := filepath.Join(base, "profiles", "work")
	if _, statErr := os.Stat(wantProfile); statErr != nil {
		t.Fatalf("created profile missing at base home %s: %v", wantProfile, statErr)
	}
	if _, statErr := os.Stat(filepath.Join(activeProfileRoot, "profiles", "work")); !os.IsNotExist(statErr) {
		t.Fatalf("setup profiles created nested profile under active profile, stat err=%v", statErr)
	}
	if _, readErr := os.ReadFile(filepath.Join(wantProfile, "config.toml")); readErr != nil {
		t.Fatalf("profile config must be written under base-home profile root: %v", readErr)
	}
	out := stdout + stderr
	for _, want := range []string{"memory_db: .../work/memory.db", "goncho_db: .../work/memory.db", "sessions_db: .../work/sessions.db"} {
		if !strings.Contains(out, want) {
			t.Fatalf("setup profiles output missing storage contract %q:\n%s", want, out)
		}
	}
}

func TestSetupProfilesPersistsWorkspaceListForDefaultProfile(t *testing.T) {
	home := t.TempDir()
	t.Setenv("GORMES_HOME", home)
	fake := &setupCommandFakeSeams{isTTY: true}

	// skip create, select default (blank -> active), two workspace dirs,
	// blank channels.
	in := "\n\n/ws/alpha,/ws/beta\n\n"
	stdout, stderr, err := runSetupTestCommandWithInput(t, fake.seams(), in, "profiles")
	if err != nil {
		t.Fatalf("setup profiles: Execute() error = %v stderr=%s", err, stderr)
	}
	out := stdout + stderr
	if !strings.Contains(out, "Profiles configuration complete!") {
		t.Fatalf("successful persist must print the completion footer:\n%s", out)
	}

	raw, rerr := os.ReadFile(config.ConfigPath())
	if rerr != nil {
		t.Fatalf("default profile config.toml must be written: %v", rerr)
	}
	body := string(raw)
	if !strings.Contains(body, "/ws/alpha") || !strings.Contains(body, "/ws/beta") {
		t.Fatalf("config.toml must contain both workspace dirs:\n%s", body)
	}
	if !strings.Contains(body, "[") || !strings.Contains(body, "]") {
		t.Fatalf("workspaces must be persisted as a TOML array:\n%s", body)
	}

	cfg, lerr := config.Load(nil)
	if lerr != nil {
		t.Fatalf("config.Load round-trip: %v", lerr)
	}
	got := cfg.Agents.Defaults.Workspaces
	if len(got) != 2 || got[0] != "/ws/alpha" || got[1] != "/ws/beta" {
		t.Fatalf("config.Load must round-trip the workspace list, got %#v", got)
	}
}

// A named profile's workspaces persist into THAT profile's own
// profiles/<name>/config.toml — never the default-home config.toml.
func TestSetupProfilesNamedProfileWritesOwnConfigNotDefaultHome(t *testing.T) {
	home := t.TempDir()
	t.Setenv("GORMES_HOME", home)
	fake := &setupCommandFakeSeams{isTTY: true}

	// create "work", select "work", set its workspaces, blank channels.
	in := "work\nwork\n/srv/work-a,/srv/work-b\n\n"
	stdout, stderr, err := runSetupTestCommandWithInput(t, fake.seams(), in, "profiles")
	if err != nil {
		t.Fatalf("setup profiles: Execute() error = %v stderr=%s", err, stderr)
	}

	namedCfg := filepath.Join(config.GormesHome(), "profiles", "work", "config.toml")
	raw, rerr := os.ReadFile(namedCfg)
	if rerr != nil {
		t.Fatalf("named profile config.toml must be written at %s: %v", namedCfg, rerr)
	}
	if !strings.Contains(string(raw), "/srv/work-a") || !strings.Contains(string(raw), "/srv/work-b") {
		t.Fatalf("named profile config must contain its workspace list:\n%s", raw)
	}

	defaultCfg := filepath.Join(config.GormesHome(), "config.toml")
	if db, derr := os.ReadFile(defaultCfg); derr == nil && strings.Contains(string(db), "/srv/work-a") {
		t.Fatalf("named profile workspaces must NOT leak into the default-home config:\n%s", db)
	}
	out := stdout + stderr
	if strings.Contains(out, home) {
		t.Fatalf("setup profiles output leaked raw profile config path rooted at %s:\n%s", home, out)
	}
	if !strings.Contains(out, ".../work/config.toml") {
		t.Fatalf("setup profiles output must identify the redacted profile config path:\n%s", out)
	}
}

func TestSetupProfilesFromProfileScopedEnvLoadsBaseControlCenterConfig(t *testing.T) {
	base := filepath.Join(t.TempDir(), ".gormes")
	workRoot := filepath.Join(base, "profiles", "work")
	if err := os.MkdirAll(workRoot, 0o700); err != nil {
		t.Fatalf("mkdir active profile root: %v", err)
	}
	if err := os.WriteFile(filepath.Join(base, "config.toml"), []byte(`config_version = 2

[profiles.main]
enabled = true
name = "Main desk"
`), 0o600); err != nil {
		t.Fatalf("write base v2 config: %v", err)
	}
	t.Setenv("GORMES_HOME", workRoot)
	fake := &setupCommandFakeSeams{isTTY: true}

	oldInputIsTerminal := setupInputIsTerminal
	oldRunner := runSetupProfilesTUI
	setupInputIsTerminal = func(*os.File) bool { return true }
	runSetupProfilesTUI = func(_ context.Context, _ *os.File, _ io.Writer, state setupProfilesTUIState) (setupProfilesTUIResult, error) {
		if !state.ControlCenter {
			t.Fatalf("setup profiles loaded legacy state from active profile; want base-home Control Center state: %+v", state)
		}
		if len(state.Profiles) != 1 || state.Profiles[0].Name != "main" {
			t.Fatalf("Control Center profiles = %+v, want root config profiles.main", state.Profiles)
		}
		return setupProfilesTUIResult{Discarded: true}, nil
	}
	t.Cleanup(func() {
		setupInputIsTerminal = oldInputIsTerminal
		runSetupProfilesTUI = oldRunner
	})

	cmd := newSetupCommandWithSeams(fake.seams())
	var stdout, stderr strings.Builder
	cmd.SetIn(os.Stdin)
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"profiles"})
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true
	if err := cmd.Execute(); err != nil {
		t.Fatalf("setup profiles: Execute() error = %v stdout=%s stderr=%s", err, stdout.String(), stderr.String())
	}
	if !strings.Contains(stdout.String()+stderr.String(), "Setup profiles draft discarded") {
		t.Fatalf("setup profiles did not apply Control Center result:\nstdout=%s\nstderr=%s", stdout.String(), stderr.String())
	}
}

func TestSetupProfilesControlCenterTUIShowsProviderCatalogAndStagesModelConfig(t *testing.T) {
	t.Setenv("GORMES_TULIN_OPENROUTER_API_KEY", "sk-tulin-openrouter")
	withOpenRouterModelCatalogFetcherForTest(t, func(context.Context) ([]string, error) {
		return []string{"zai/glm-4.6", "openai/gpt-5.2", "meta-llama/llama-4", "anthropic/claude-sonnet-4.5"}, nil
	})
	state := buildSetupProfilesControlCenterTUIState(config.Config{
		ConfigVersion: config.CurrentConfigVersion,
		Profiles: map[string]config.ProfileCfg{
			"tulin": {
				Enabled: true,
				Name:    "Tulin Sage",
				Providers: map[string]config.ProfileProviderCfg{
					"openrouter": {
						Enabled:       true,
						Credential:    "tulin-openrouter",
						DefaultModel:  "meta-llama/llama-4",
						AllowedModels: []string{"meta-llama/llama-4"},
					},
				},
			},
		},
		Credentials: map[string]config.CredentialCfg{
			"tulin-openrouter": {Kind: "provider", Provider: "openrouter", OwnerProfile: "tulin", SecretRef: &config.SecretRef{Source: config.SecretRefSourceEnv, ID: "GORMES_TULIN_OPENROUTER_API_KEY"}},
		},
	})
	m := newSetupProfilesModel(state)
	m.width = 200
	m.height = 80
	view := m.View()
	for _, want := range []string{
		"Providers: openrouter credential=tulin-openrouter model=meta-llama/llama-4 status=ready",
		"provider models openrouter: anthropic/claude-sonnet-4.5, meta-llama/llama-4, openai/gpt-5.2, zai/glm-4.6",
		"p assign provider credential/model",
	} {
		if !strings.Contains(view, want) {
			t.Fatalf("control center provider view missing %q:\n%s", want, view)
		}
	}
	if strings.Contains(view, "GORMES_TULIN_OPENROUTER_API_KEY") || strings.Contains(view, "sk-") {
		t.Fatalf("control center provider view leaked secret evidence:\n%s", view)
	}

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'p'}})
	m = updated.(setupProfilesModel)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("openrouter|tulin-openrouter|meta-llama/llama-4|meta-llama/llama-4,openai/gpt-5.2")})
	m = updated.(setupProfilesModel)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(setupProfilesModel)

	if !m.result.ProviderCredentialSet || m.result.ProviderID != "openrouter" || m.result.ProviderCredentialID != "tulin-openrouter" {
		t.Fatalf("provider credential result = %+v", m.result)
	}
	if m.result.ProviderDefaultModel != "meta-llama/llama-4" {
		t.Fatalf("provider default model result = %+v", m.result)
	}
	if got, want := m.result.ProviderAllowedModels, []string{"meta-llama/llama-4", "openai/gpt-5.2"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("provider allowed models result = %#v, want %#v", got, want)
	}
}

func TestSetupProfilesControlCenterTUIShowsChannelReadinessAndStagesAllowLists(t *testing.T) {
	state := buildSetupProfilesControlCenterTUIState(config.Config{
		ConfigVersion: config.CurrentConfigVersion,
		Profiles: map[string]config.ProfileCfg{
			"tulin": {
				Enabled: true,
				Name:    "Tulin Sage",
				Channels: map[string]config.ProfileChannelCfg{
					"telegram": {
						Enabled:        true,
						Credential:     "tulin-telegram",
						AllowedChats:   []string{"222"},
						AllowedUsers:   []string{"6586915095"},
						RequireMention: true,
						ToolProgress:   "compact",
					},
				},
			},
		},
		Credentials: map[string]config.CredentialCfg{
			"tulin-telegram": {Kind: "channel", Channel: "telegram", OwnerProfile: "tulin", SecretRef: &config.SecretRef{Source: config.SecretRefSourceEnv, ID: "GORMES_TULIN_TELEGRAM_BOT_TOKEN"}},
		},
	})
	m := newSetupProfilesModel(state)
	m.width = 200
	m.height = 80
	view := m.View()
	for _, want := range []string{
		"Channels: telegram credential=tulin-telegram chats=1 users=1 require_mention=true tool_progress=compact status=ready",
		"t assign channel credential/policy",
	} {
		if !strings.Contains(view, want) {
			t.Fatalf("control center channel view missing %q:\n%s", want, view)
		}
	}
	for _, forbidden := range []string{"GORMES_TULIN_TELEGRAM_BOT_TOKEN", "6586915095", "222", "bot-token"} {
		if strings.Contains(view, forbidden) {
			t.Fatalf("control center channel view leaked sensitive value %q:\n%s", forbidden, view)
		}
	}

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'t'}})
	m = updated.(setupProfilesModel)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("telegram|tulin-telegram|222,333|6586915095|true|compact")})
	m = updated.(setupProfilesModel)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(setupProfilesModel)

	if !m.result.ChannelCredentialSet || m.result.ChannelID != "telegram" || m.result.ChannelCredentialID != "tulin-telegram" {
		t.Fatalf("channel credential result = %+v", m.result)
	}
	if !m.result.ChannelPolicySet || !m.result.ChannelRequireMention || m.result.ChannelToolProgress != "compact" {
		t.Fatalf("channel policy result = %+v", m.result)
	}
	if got, want := m.result.ChannelAllowedChats, []string{"222", "333"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("channel allowed chats result = %#v, want %#v", got, want)
	}
	if got, want := m.result.ChannelAllowedUsers, []string{"6586915095"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("channel allowed users result = %#v, want %#v", got, want)
	}
}

func TestSetupProfilesControlCenterTUIKeyFlowStagesRenameCredentialsAndDiscard(t *testing.T) {
	state := buildSetupProfilesControlCenterTUIState(config.Config{
		ConfigVersion: config.CurrentConfigVersion,
		Profiles: map[string]config.ProfileCfg{
			"main": {Enabled: true, Name: ""},
		},
	})
	m := newSetupProfilesModel(state)

	view := m.View()
	for _, want := range []string{
		"Setup profiles",
		"main",
		"r rename display name",
		"p assign provider credential/model",
		"t assign channel credential/policy",
		"d discard draft",
	} {
		if !strings.Contains(view, want) {
			t.Fatalf("control center TUI missing %q:\n%s", want, view)
		}
	}
	for _, forbidden := range []string{"set active", "~/.gormes/profiles", "active_profile"} {
		if strings.Contains(view, forbidden) {
			t.Fatalf("control center TUI leaked legacy action %q:\n%s", forbidden, view)
		}
	}

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	m = updated.(setupProfilesModel)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(" Main desk ")})
	m = updated.(setupProfilesModel)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(setupProfilesModel)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'p'}})
	m = updated.(setupProfilesModel)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("openrouter:main-openrouter")})
	m = updated.(setupProfilesModel)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(setupProfilesModel)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'t'}})
	m = updated.(setupProfilesModel)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("telegram:main-telegram")})
	m = updated.(setupProfilesModel)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(setupProfilesModel)

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	m = updated.(setupProfilesModel)
	if cmd == nil {
		t.Fatal("discard returned nil command, want tea.Quit")
	}
	if _, ok := cmd().(tea.QuitMsg); !ok {
		t.Fatalf("discard command = %T, want tea.QuitMsg", cmd())
	}
	if !m.result.Discarded || m.result.Cancelled {
		t.Fatalf("discard result = %+v, want discarded without cancel", m.result)
	}
	if !m.result.DisplayNameSet || m.result.DisplayName != "Main desk" {
		t.Fatalf("display name result = %+v", m.result)
	}
	if !m.result.ProviderCredentialSet || m.result.ProviderID != "openrouter" || m.result.ProviderCredentialID != "main-openrouter" {
		t.Fatalf("provider credential result = %+v", m.result)
	}
	if !m.result.ChannelCredentialSet || m.result.ChannelID != "telegram" || m.result.ChannelCredentialID != "main-telegram" {
		t.Fatalf("channel credential result = %+v", m.result)
	}
}

func TestSetupProfilesV2TUIDiscardWritesNothing(t *testing.T) {
	home := t.TempDir()
	t.Setenv("GORMES_HOME", home)
	fake := &setupCommandFakeSeams{isTTY: true}
	if err := os.MkdirAll(home, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(config.ConfigPath(), []byte(`
config_version = 2

[profiles.main]
enabled = true
name = ""
`), 0o600); err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(config.ConfigPath())
	if err != nil {
		t.Fatalf("read before config: %v", err)
	}

	stdin, err := os.Open(os.DevNull)
	if err != nil {
		t.Fatalf("open devnull: %v", err)
	}
	defer stdin.Close()

	oldInputIsTerminal := setupInputIsTerminal
	oldRunner := runSetupProfilesTUI
	oldWriter := writeSetupProfilesControlCenterConfig
	writes := 0
	setupInputIsTerminal = func(file *os.File) bool { return file == stdin }
	runSetupProfilesTUI = func(_ context.Context, _ *os.File, _ io.Writer, _ setupProfilesTUIState) (setupProfilesTUIResult, error) {
		return setupProfilesTUIResult{
			Selected:              "main",
			DisplayName:           "Main desk",
			DisplayNameSet:        true,
			ProviderID:            "openrouter",
			ProviderCredentialID:  "main-openrouter",
			ProviderCredentialSet: true,
			Discarded:             true,
		}, nil
	}
	writeSetupProfilesControlCenterConfig = func(string, config.Config) error {
		writes++
		return nil
	}
	t.Cleanup(func() {
		setupInputIsTerminal = oldInputIsTerminal
		runSetupProfilesTUI = oldRunner
		writeSetupProfilesControlCenterConfig = oldWriter
	})

	cmd := newSetupCommandWithSeams(fake.seams())
	var stdout, stderr strings.Builder
	cmd.SetIn(stdin)
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"profiles"})
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true
	if err := cmd.Execute(); err != nil {
		t.Fatalf("setup profiles discard: Execute() error = %v stdout=%s stderr=%s", err, stdout.String(), stderr.String())
	}
	if writes != 0 {
		t.Fatalf("discard called root writer %d time(s), want 0", writes)
	}
	after, err := os.ReadFile(config.ConfigPath())
	if err != nil {
		t.Fatalf("read after config: %v", err)
	}
	if string(after) != string(before) {
		t.Fatalf("discard changed config.toml:\nbefore:\n%s\nafter:\n%s", before, after)
	}
	if out := stdout.String() + stderr.String(); !strings.Contains(out, "Setup profiles draft discarded") {
		t.Fatalf("discard output missing confirmation:\n%s", out)
	}
}

func TestSetupProfilesControlCenterTUIKeyFlowStagesLegacyMigrationForApply(t *testing.T) {
	state := setupProfilesTUIState{
		ControlCenter:         true,
		MigrationAvailable:    true,
		MigrationPreviewLines: []string{"add profiles.main", "add credentials.main-openrouter secret_ref=env:GORMES_MAIN_OPENROUTER_API_KEY redacted=true", "legacy active_profile=tulin is compatibility state only"},
		Profiles:              []setupProfileView{{Name: "main"}},
	}
	m := newSetupProfilesModel(state)
	view := m.View()
	for _, want := range []string{"Migration preview", "add profiles.main", "m stage legacy migration", "s apply draft"} {
		if !strings.Contains(view, want) {
			t.Fatalf("migration control center view missing %q:\n%s", want, view)
		}
	}
	for _, forbidden := range []string{"sk-", "bot-token", "discord-token"} {
		if strings.Contains(view, forbidden) {
			t.Fatalf("migration preview leaked secret-looking value %q:\n%s", forbidden, view)
		}
	}

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'m'}})
	m = updated.(setupProfilesModel)
	if cmd != nil {
		t.Fatalf("stage migration command = %T, want nil before explicit Apply", cmd())
	}
	if !m.result.MigrateLegacyConfig {
		t.Fatalf("migration result = %+v, want staged migration", m.result)
	}
	if view := m.View(); !strings.Contains(view, "legacy migration staged for Apply") {
		t.Fatalf("staged migration view missing confirmation:\n%s", view)
	}

	updated, cmd = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
	m = updated.(setupProfilesModel)
	if cmd == nil {
		t.Fatal("apply returned nil command, want tea.Quit")
	}
	if _, ok := cmd().(tea.QuitMsg); !ok {
		t.Fatalf("apply command = %T, want tea.QuitMsg", cmd())
	}
	if !m.result.MigrateLegacyConfig || m.result.Discarded || m.result.Cancelled {
		t.Fatalf("final migration result = %+v", m.result)
	}
}

func TestSetupProfilesLegacyControlCenterMigrationAppliesViaInternalConfig(t *testing.T) {
	home := seedSetupProfilesLegacyMigrationHome(t)
	t.Setenv("GORMES_HOME", home)
	fake := &setupCommandFakeSeams{isTTY: true}

	stdin, err := os.Open(os.DevNull)
	if err != nil {
		t.Fatalf("open devnull: %v", err)
	}
	defer stdin.Close()

	oldInputIsTerminal := setupInputIsTerminal
	oldRunner := runSetupProfilesTUI
	setupInputIsTerminal = func(file *os.File) bool { return file == stdin }
	runSetupProfilesTUI = func(_ context.Context, gotStdin *os.File, _ io.Writer, state setupProfilesTUIState) (setupProfilesTUIResult, error) {
		if gotStdin != stdin {
			t.Fatalf("TUI stdin = %v, want injected stdin", gotStdin)
		}
		if !state.ControlCenter || !state.MigrationAvailable {
			t.Fatalf("legacy config must open Profile Control Center migration state, got %+v", state)
		}
		preview := strings.Join(state.MigrationPreviewLines, "\n")
		for _, want := range []string{"add profiles.main", "add profiles.tulin", "credentials.main-openrouter", "legacy active_profile=tulin"} {
			if !strings.Contains(preview, want) {
				t.Fatalf("migration preview missing %q:\n%s", want, preview)
			}
		}
		for _, forbidden := range []string{"sk-main-secret", "bot-token-main", "sk-tulin-secret", "discord-token-tulin"} {
			if strings.Contains(preview, forbidden) {
				t.Fatalf("migration preview leaked %q:\n%s", forbidden, preview)
			}
		}
		return setupProfilesTUIResult{MigrateLegacyConfig: true}, nil
	}
	t.Cleanup(func() {
		setupInputIsTerminal = oldInputIsTerminal
		runSetupProfilesTUI = oldRunner
	})

	cmd := newSetupCommandWithSeams(fake.seams())
	var stdout, stderr strings.Builder
	cmd.SetIn(stdin)
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"profiles"})
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true
	if err := cmd.Execute(); err != nil {
		t.Fatalf("setup profiles migration: Execute() error = %v stdout=%s stderr=%s", err, stdout.String(), stderr.String())
	}

	out := stdout.String() + stderr.String()
	for _, want := range []string{"Setup profiles migration:", "Applied legacy profile config migration", "Backup:"} {
		if !strings.Contains(out, want) {
			t.Fatalf("migration apply output missing %q:\n%s", want, out)
		}
	}
	for _, forbidden := range []string{"Create a new profile?", "~/.gormes/profiles", "sk-main-secret", "bot-token-main", "discord-token-tulin"} {
		if strings.Contains(out, forbidden) {
			t.Fatalf("migration output leaked legacy prompt/secret %q:\n%s", forbidden, out)
		}
	}
	if strings.Contains(out, home) {
		t.Fatalf("migration output leaked raw root path rooted at %s:\n%s", home, out)
	}
	if !strings.Contains(out, ".../") || !strings.Contains(out, "config.toml") {
		t.Fatalf("migration output must include redacted config/backup paths:\n%s", out)
	}
	cfg, err := config.Load(nil)
	if err != nil {
		t.Fatalf("config.Load migrated: %v", err)
	}
	if !cfg.ProfileConfigV2Available() {
		t.Fatalf("migrated config did not enable profile config v2: %+v", cfg)
	}
	if _, ok := cfg.Profiles["main"]; !ok {
		t.Fatalf("migrated profiles missing main: %+v", cfg.Profiles)
	}
	if _, ok := cfg.Profiles["tulin"]; !ok {
		t.Fatalf("migrated profiles missing tulin: %+v", cfg.Profiles)
	}
	if _, err := os.Stat(filepath.Join(home, "profiles", "tulin", "config.toml")); err != nil {
		t.Fatalf("legacy profile config should be preserved: %v", err)
	}
}

func TestSetupProfilesV2TUICancelWritesNothing(t *testing.T) {
	home := t.TempDir()
	t.Setenv("GORMES_HOME", home)
	fake := &setupCommandFakeSeams{isTTY: true}
	if err := os.MkdirAll(home, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(config.ConfigPath(), []byte(`
config_version = 2

[profiles.main]
enabled = true
name = ""
`), 0o600); err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(config.ConfigPath())
	if err != nil {
		t.Fatalf("read before config: %v", err)
	}

	stdin, err := os.Open(os.DevNull)
	if err != nil {
		t.Fatalf("open devnull: %v", err)
	}
	defer stdin.Close()

	oldInputIsTerminal := setupInputIsTerminal
	oldRunner := runSetupProfilesTUI
	oldWriter := writeSetupProfilesControlCenterConfig
	writes := 0
	setupInputIsTerminal = func(file *os.File) bool { return file == stdin }
	runSetupProfilesTUI = func(_ context.Context, _ *os.File, _ io.Writer, _ setupProfilesTUIState) (setupProfilesTUIResult, error) {
		return setupProfilesTUIResult{Cancelled: true, Selected: "main", DisplayName: "Main desk", DisplayNameSet: true}, nil
	}
	writeSetupProfilesControlCenterConfig = func(string, config.Config) error {
		writes++
		return nil
	}
	t.Cleanup(func() {
		setupInputIsTerminal = oldInputIsTerminal
		runSetupProfilesTUI = oldRunner
		writeSetupProfilesControlCenterConfig = oldWriter
	})

	cmd := newSetupCommandWithSeams(fake.seams())
	var stdout, stderr strings.Builder
	cmd.SetIn(stdin)
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"profiles"})
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true
	if err := cmd.Execute(); err != nil {
		t.Fatalf("setup profiles cancel: Execute() error = %v stdout=%s stderr=%s", err, stdout.String(), stderr.String())
	}
	if writes != 0 {
		t.Fatalf("cancel called root writer %d time(s), want 0", writes)
	}
	after, err := os.ReadFile(config.ConfigPath())
	if err != nil {
		t.Fatalf("read after config: %v", err)
	}
	if string(after) != string(before) {
		t.Fatalf("cancel changed config.toml:\nbefore:\n%s\nafter:\n%s", before, after)
	}
	if out := stdout.String() + stderr.String(); !strings.Contains(out, "Setup canceled.") {
		t.Fatalf("cancel output missing confirmation:\n%s", out)
	}
}

func TestSetupProfilesV2TUIAppliesOneRootConfigTransaction(t *testing.T) {
	home := t.TempDir()
	t.Setenv("GORMES_HOME", home)
	fake := &setupCommandFakeSeams{isTTY: true}
	if err := os.MkdirAll(home, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(config.ConfigPath(), []byte(`
config_version = 2

[profiles.main]
enabled = true
name = ""
`), 0o600); err != nil {
		t.Fatal(err)
	}

	stdin, err := os.Open(os.DevNull)
	if err != nil {
		t.Fatalf("open devnull: %v", err)
	}
	defer stdin.Close()

	oldInputIsTerminal := setupInputIsTerminal
	oldRunner := runSetupProfilesTUI
	setupInputIsTerminal = func(file *os.File) bool { return file == stdin }
	runSetupProfilesTUI = func(_ context.Context, gotStdin *os.File, _ io.Writer, state setupProfilesTUIState) (setupProfilesTUIResult, error) {
		if gotStdin != stdin {
			t.Fatalf("TUI stdin = %v, want injected stdin", gotStdin)
		}
		if len(state.Profiles) != 1 || state.Profiles[0].Name != "main" || state.Profiles[0].Active {
			t.Fatalf("v2 TUI state = %+v, want root profile service main without legacy active marker", state)
		}
		return setupProfilesTUIResult{
			CreateName:            "tulin",
			Selected:              "tulin",
			DisplayName:           "Tulin Sage",
			DisplayNameSet:        true,
			Workspaces:            []string{"/workspace/tulin"},
			WorkspacesSet:         true,
			Channels:              []string{"telegram"},
			ChannelsSet:           true,
			ProviderID:            "openrouter",
			ProviderCredentialID:  "tulin-openrouter",
			ProviderDefaultModel:  "meta-llama/llama-4",
			ProviderAllowedModels: []string{"meta-llama/llama-4", "openai/gpt-5.2"},
			ProviderCredentialSet: true,
			ChannelID:             "telegram",
			ChannelCredentialID:   "tulin-telegram",
			ChannelAllowedChats:   []string{"222", "333"},
			ChannelAllowedUsers:   []string{"6586915095"},
			ChannelRequireMention: true,
			ChannelToolProgress:   "compact",
			ChannelPolicySet:      true,
			ChannelCredentialSet:  true,
		}, nil
	}
	t.Cleanup(func() {
		setupInputIsTerminal = oldInputIsTerminal
		runSetupProfilesTUI = oldRunner
	})

	cmd := newSetupCommandWithSeams(fake.seams())
	var stdout, stderr strings.Builder
	cmd.SetIn(stdin)
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"profiles"})
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true
	if err := cmd.Execute(); err != nil {
		t.Fatalf("setup profiles: Execute() error = %v stdout=%s stderr=%s", err, stdout.String(), stderr.String())
	}

	out := stdout.String() + stderr.String()
	if !strings.Contains(out, "Setup profiles draft:") || !strings.Contains(out, "Applied") {
		t.Fatalf("v2 setup profiles must preview and apply the control center draft:\n%s", out)
	}
	if strings.Contains(out, "Active profile set") || strings.Contains(out, "profiles/tulin/config.toml") {
		t.Fatalf("v2 setup profiles leaked legacy profile-home/active behavior:\n%s", out)
	}
	if strings.Contains(out, home) {
		t.Fatalf("v2 setup profiles output leaked raw root config path rooted at %s:\n%s", home, out)
	}
	if !strings.Contains(out, ".../") || !strings.Contains(out, "config.toml") {
		t.Fatalf("v2 setup profiles output must identify the redacted root config path:\n%s", out)
	}
	if _, err := os.Stat(filepath.Join(home, "active_profile")); !os.IsNotExist(err) {
		t.Fatalf("v2 setup profiles must not write active_profile, stat err=%v", err)
	}
	if _, err := os.Stat(filepath.Join(home, "profiles", "tulin", "config.toml")); !os.IsNotExist(err) {
		t.Fatalf("v2 setup profiles must not write per-profile config.toml, stat err=%v", err)
	}
	if info, err := os.Stat(filepath.Join(home, "profiles", "tulin", "home")); err != nil || !info.IsDir() {
		t.Fatalf("v2 setup profiles must materialize the new profile runtime home without per-profile config: info=%+v err=%v", info, err)
	}

	cfg, err := config.Load(nil)
	if err != nil {
		t.Fatalf("config.Load: %v", err)
	}
	tulin, ok := cfg.Profiles["tulin"]
	if !ok || !tulin.Enabled || tulin.Name != "Tulin Sage" {
		t.Fatalf("profiles.tulin = %+v, ok=%t; want enabled root profile with display name", tulin, ok)
	}
	if len(tulin.Workspaces) != 1 || tulin.Workspaces[0] != "/workspace/tulin" {
		t.Fatalf("profiles.tulin.workspaces = %#v", tulin.Workspaces)
	}
	if provider, ok := tulin.Providers["openrouter"]; !ok || !provider.Enabled || provider.Credential != "tulin-openrouter" || provider.DefaultModel != "meta-llama/llama-4" || !reflect.DeepEqual(provider.AllowedModels, []string{"meta-llama/llama-4", "openai/gpt-5.2"}) {
		t.Fatalf("profiles.tulin.providers.openrouter = %+v, ok=%t", provider, ok)
	}
	if channel, ok := tulin.Channels["telegram"]; !ok || !channel.Enabled || channel.Credential != "tulin-telegram" || !reflect.DeepEqual(channel.AllowedChats, []string{"222", "333"}) || !reflect.DeepEqual(channel.AllowedUsers, []string{"6586915095"}) || !channel.RequireMention || channel.ToolProgress != "compact" {
		t.Fatalf("profiles.tulin.channels.telegram = %+v, ok=%t", channel, ok)
	}
	if cred := cfg.Credentials["tulin-openrouter"]; cred.Kind != "provider" || cred.Provider != "openrouter" || cred.OwnerProfile != "tulin" || cred.SecretRef == nil || cred.SecretRef.ID != "GORMES_TULIN_OPENROUTER_API_KEY" {
		t.Fatalf("credentials.tulin-openrouter = %+v", cred)
	}
	if cred := cfg.Credentials["tulin-telegram"]; cred.Kind != "channel" || cred.Channel != "telegram" || cred.OwnerProfile != "tulin" || cred.SecretRef == nil || cred.SecretRef.ID != "GORMES_TULIN_TELEGRAM_BOT_TOKEN" {
		t.Fatalf("credentials.tulin-telegram = %+v", cred)
	}
}

// Non-interactive / no-TTY must NOT emit a false "Profiles configuration
// complete!" footer and must not write config (the shipped chrome suppresses
// the footer when the section returns an error).
func TestSetupProfilesNoTTYNoFalseSuccess(t *testing.T) {
	home := t.TempDir()
	t.Setenv("GORMES_HOME", home)
	fake := &setupCommandFakeSeams{isTTY: false}

	stdout, stderr, _ := runSetupTestCommand(t, fake.seams(), "profiles")

	out := stdout + stderr
	if strings.Contains(out, "Profiles configuration complete!") {
		t.Fatalf("no-TTY setup profiles must NOT print a false completion footer:\n%s", out)
	}
	if _, err := os.Stat(filepath.Join(config.GormesHome(), "config.toml")); err == nil {
		t.Fatalf("no-TTY setup profiles must not write config")
	}
}

// Child 2/3: AgentDefaultsCfg.Channels is a Gormes-owned per-profile channel
// list (symmetric with the shipped Workspaces field). It persists through the
// real config.WriteTOMLValue round-trip as a TOML array and round-trips back
// via config.Load.
func seedSetupProfilesLegacyMigrationHome(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	root := []byte(`_config_version = 1

[hermes]
endpoint = "https://openrouter.ai/api/v1"
provider = "openrouter"
model = "openrouter/auto"
api_key = "sk-main-secret"

[telegram]
bot_token = "bot-token-main"
allowed_chat_id = 12345
allowed_user_ids = [6586915095]
tool_progress = "new"

[agents.defaults]
workspaces = ["/srv/main", "/srv/shared"]
channels = ["telegram"]
`)
	if err := os.WriteFile(filepath.Join(home, "config.toml"), root, 0o600); err != nil {
		t.Fatalf("write root config: %v", err)
	}
	if err := os.WriteFile(filepath.Join(home, "active_profile"), []byte("tulin\n"), 0o600); err != nil {
		t.Fatalf("write active_profile: %v", err)
	}
	profileDir := filepath.Join(home, "profiles", "tulin")
	if err := os.MkdirAll(profileDir, 0o700); err != nil {
		t.Fatalf("mkdir profile dir: %v", err)
	}
	profile := []byte(`_config_version = 1

[hermes]
provider = "openai"
model = "gpt-5.3-codex"
api_key = "sk-tulin-secret"

[discord]
token = "discord-token-tulin"
allowed_channel_id = "98765"

[agents.defaults]
workspace = "/srv/tulin"
channels = ["discord"]
`)
	if err := os.WriteFile(filepath.Join(profileDir, "config.toml"), profile, 0o600); err != nil {
		t.Fatalf("write profile config: %v", err)
	}
	return home
}

func TestAgentDefaultsChannelsConfigRoundTrip(t *testing.T) {
	home := t.TempDir()
	t.Setenv("GORMES_HOME", home)

	if err := config.WriteTOMLValue(config.ConfigPath(), "agents.defaults.channels", "telegram,discord"); err != nil {
		t.Fatalf("WriteTOMLValue agents.defaults.channels: %v", err)
	}
	raw, rerr := os.ReadFile(config.ConfigPath())
	if rerr != nil {
		t.Fatalf("read config: %v", rerr)
	}
	if !strings.Contains(string(raw), "[") || !strings.Contains(string(raw), "]") {
		t.Fatalf("channels must persist as a TOML array:\n%s", raw)
	}

	cfg, lerr := config.Load(nil)
	if lerr != nil {
		t.Fatalf("config.Load round-trip: %v", lerr)
	}
	got := cfg.Agents.Defaults.Channels
	if len(got) != 2 || got[0] != "telegram" || got[1] != "discord" {
		t.Fatalf("config.Load must round-trip the channel list, got %#v", got)
	}
}

// Child 2/3 TB2: selecting the default profile and entering channels persists
// them as a TOML array into ~/.gormes/config.toml, round-tripping via
// config.Load into AgentsCfg.Defaults.Channels.
func TestSetupProfilesPersistsChannelListForDefaultProfile(t *testing.T) {
	home := t.TempDir()
	t.Setenv("GORMES_HOME", home)
	fake := &setupCommandFakeSeams{isTTY: true}

	// skip create, select default, blank workspace, channels telegram,discord.
	in := "\n\n\ntelegram,discord\n"
	stdout, stderr, err := runSetupTestCommandWithInput(t, fake.seams(), in, "profiles")
	if err != nil {
		t.Fatalf("setup profiles: Execute() error = %v stderr=%s", err, stderr)
	}
	out := stdout + stderr
	if !strings.Contains(out, "Set 2 channel(s) for profile \"default\"") {
		t.Fatalf("must confirm 2 channels set:\n%s", out)
	}
	if !strings.Contains(out, "Profiles configuration complete!") {
		t.Fatalf("successful persist must print the completion footer:\n%s", out)
	}

	cfg, lerr := config.Load(nil)
	if lerr != nil {
		t.Fatalf("config.Load round-trip: %v", lerr)
	}
	got := cfg.Agents.Defaults.Channels
	if len(got) != 2 || got[0] != "telegram" || got[1] != "discord" {
		t.Fatalf("config.Load must round-trip the channel list, got %#v", got)
	}
}

// Child 2/3 TB3: a named profile's channels land in its own
// profiles/<name>/config.toml, never the default-home config.
func TestSetupProfilesNamedProfileChannelsWriteOwnConfig(t *testing.T) {
	home := t.TempDir()
	t.Setenv("GORMES_HOME", home)
	fake := &setupCommandFakeSeams{isTTY: true}

	// create work, select work, blank workspace, channels slack,whatsapp.
	in := "work\nwork\n\nslack,whatsapp\n"
	_, stderr, err := runSetupTestCommandWithInput(t, fake.seams(), in, "profiles")
	if err != nil {
		t.Fatalf("setup profiles: Execute() error = %v stderr=%s", err, stderr)
	}
	namedCfg := filepath.Join(config.GormesHome(), "profiles", "work", "config.toml")
	raw, rerr := os.ReadFile(namedCfg)
	if rerr != nil {
		t.Fatalf("named profile config.toml must be written: %v", rerr)
	}
	if !strings.Contains(string(raw), "slack") || !strings.Contains(string(raw), "whatsapp") {
		t.Fatalf("named profile config must contain its channel list:\n%s", raw)
	}
	if db, derr := os.ReadFile(filepath.Join(config.GormesHome(), "config.toml")); derr == nil && strings.Contains(string(db), "whatsapp") {
		t.Fatalf("named profile channels must NOT leak into default-home config:\n%s", db)
	}
}

// Child 2/3 TB4: unknown channels are skipped with a Gormes-owned notice and
// never persisted; only valid channels land.
func TestSetupProfilesSkipsUnknownChannels(t *testing.T) {
	home := t.TempDir()
	t.Setenv("GORMES_HOME", home)
	fake := &setupCommandFakeSeams{isTTY: true}

	in := "\n\n\ntelegram,bogus\n"
	stdout, stderr, err := runSetupTestCommandWithInput(t, fake.seams(), in, "profiles")
	if err != nil {
		t.Fatalf("setup profiles: Execute() error = %v stderr=%s", err, stderr)
	}
	out := stdout + stderr
	if !strings.Contains(out, "Skipping unknown channel \"bogus\"") {
		t.Fatalf("unknown channel must produce a Gormes-owned skip notice:\n%s", out)
	}
	cfg, lerr := config.Load(nil)
	if lerr != nil {
		t.Fatalf("config.Load: %v", lerr)
	}
	got := cfg.Agents.Defaults.Channels
	if len(got) != 1 || got[0] != "telegram" {
		t.Fatalf("only valid channels must persist, got %#v", got)
	}
}
