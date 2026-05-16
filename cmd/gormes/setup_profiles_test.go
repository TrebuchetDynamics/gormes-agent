package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
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
	_ = stdout
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
