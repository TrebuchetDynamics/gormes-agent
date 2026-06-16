package gormescli

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli"
	"github.com/spf13/cobra"
)

var setupProfilesInputIsTerminal = StdinIsTerminal

type SetupProfilesOptions struct {
	NonInteractive   bool
	IsTTY            func() bool
	RequiresTTYError error
	ProfileSeams     SetupProfilesSeams
}

type SetupProfilesSeams struct {
	ReadActiveProfileName func() (string, error)
	ValidateProfileName   func(string) error
	ResolveProfileRoot    func(string) (string, error)
	WriteActiveProfile    func(string) error
	CreateProfile         func(name string, cloneAll bool) (cli.ProfileCreateResult, error)
	ListKnownProfiles     func() ([]string, error)
}

func SetSetupProfilesInputIsTerminalForTest(fn func(*os.File) bool) func() {
	old := setupProfilesInputIsTerminal
	if fn == nil {
		setupProfilesInputIsTerminal = StdinIsTerminal
	} else {
		setupProfilesInputIsTerminal = fn
	}
	return func() { setupProfilesInputIsTerminal = old }
}

func SetRunSetupProfilesTUIForTest(fn func(context.Context, *os.File, io.Writer, setupProfilesTUIState) (setupProfilesTUIResult, error)) func() {
	old := runSetupProfilesTUI
	if fn == nil {
		runSetupProfilesTUI = runSetupProfilesTUIDefault
	} else {
		runSetupProfilesTUI = fn
	}
	return func() { runSetupProfilesTUI = old }
}

func SetWriteSetupProfilesControlCenterConfigForTest(fn func(string, config.Config) error) func() {
	old := writeSetupProfilesControlCenterConfig
	if fn == nil {
		writeSetupProfilesControlCenterConfig = config.WriteProfileConfigV2
	} else {
		writeSetupProfilesControlCenterConfig = fn
	}
	return func() { writeSetupProfilesControlCenterConfig = old }
}

type SetupProfileView = setupProfileView
type SetupProfilesTUIState = setupProfilesTUIState
type SetupProfilesTUIResult = setupProfilesTUIResult
type SetupProfilesModel = setupProfilesModel

func BuildSetupProfilesControlCenterTUIState(cfg config.Config) setupProfilesTUIState {
	return buildSetupProfilesControlCenterTUIState(cfg)
}

func NewSetupProfilesModelForTest(state setupProfilesTUIState) setupProfilesModel {
	return newSetupProfilesModel(state)
}

func SetupProfilesDisplayNameForTest(profile setupProfileView) string {
	return setupProfilesDisplayName(profile)
}

// runSetupProfilesSection is the Gormes-owned `gormes setup profiles` section
// (owned divergence: Hermes has no setup profiles section — Hermes profiles
// are separate ~/.hermes-<name> homes via hermes_cli/profiles.py, never a
// setup section). It reuses the profile command seams
// (defaultProfileCommandSeams) for enumeration/creation and the real
// internal/config TOML round-trip (config.WriteTOMLValue) to persist a
// per-profile workspace LIST into the SELECTED profile's own config.toml.
// Interactive profile editing uses the rich TUI when available. Non-interactive
// mode performs the safe main profile bootstrap so install/setup smoke tests
// and headless hosts can reach a profile-rooted layout without prompting.
func RunSetupProfilesSection(cmd *cobra.Command, opts SetupProfilesOptions) error {
	if opts.NonInteractive {
		return runSetupProfilesNonInteractive(cmd)
	}
	if opts.IsTTY != nil && !opts.IsTTY() {
		return opts.setupRequiresTTYError()
	}
	return runSetupProfilesInteractive(cmd, opts.profileSeams())
}

func runSetupProfilesNonInteractive(cmd *cobra.Command) error {
	restoreHome, err := scopeSetupProfilesBaseHome()
	if err != nil {
		return err
	}
	defer restoreHome()

	cfg, err := config.Load(nil)
	if err != nil {
		return fmt.Errorf("setup profiles non-interactive: load config: %w", err)
	}
	if cfg.Profiles == nil {
		cfg.Profiles = map[string]config.ProfileCfg{}
	}
	main := cfg.Profiles[config.DefaultProfileID]
	if !main.Enabled {
		main.Enabled = true
	}
	cfg.Profiles[config.DefaultProfileID] = main

	if err := materializeSetupProfilesControlCenterMainProfile(); err != nil {
		return err
	}
	if writeSetupProfilesControlCenterConfig == nil {
		return fmt.Errorf("profile control center root config writer unavailable")
	}
	if err := writeSetupProfilesControlCenterConfig(config.ConfigPath(), cfg); err != nil {
		return fmt.Errorf("setup profiles non-interactive: write config: %w", err)
	}

	out := cmd.OutOrStdout()
	fmt.Fprintln(out, "Setup profiles non-interactive:")
	fmt.Fprintf(out, "  - materialized profile %q at %s\n", config.DefaultProfileID, setupRedactedFilePath(filepath.Join(config.GormesBaseHome(), "profiles", config.DefaultProfileID)))
	fmt.Fprintf(out, "  - wrote profile registry to %s\n", setupRedactedProfileConfigPath(config.GormesHome()))
	return nil
}

func scopeSetupProfilesBaseHome() (func(), error) {
	rawHome, hadHome := os.LookupEnv("GORMES_HOME")
	currentHome := config.GormesHome()
	baseHome := config.GormesBaseHomeFor(currentHome)
	if currentHome == baseHome {
		return func() {}, nil
	}
	if err := os.Setenv("GORMES_HOME", baseHome); err != nil {
		return nil, fmt.Errorf("setup profiles: scope base home: %w", err)
	}
	return func() {
		if hadHome {
			_ = os.Setenv("GORMES_HOME", rawHome)
		} else {
			_ = os.Unsetenv("GORMES_HOME")
		}
	}, nil
}

func runSetupProfilesInteractive(cmd *cobra.Command, pseams SetupProfilesSeams) error {
	restoreHome, err := scopeSetupProfilesBaseHome()
	if err != nil {
		return err
	}
	defer restoreHome()

	out := cmd.OutOrStdout()
	known, err := pseams.ListKnownProfiles()
	if err != nil {
		return fmt.Errorf("list profiles: %w", err)
	}
	active := config.DefaultProfileID
	if pseams.ReadActiveProfileName != nil {
		if a, aerr := pseams.ReadActiveProfileName(); aerr == nil && strings.TrimSpace(a) != "" {
			active = strings.TrimSpace(a)
		}
	}
	if handled, err := maybeRunSetupProfilesTUI(cmd, pseams, known, active); handled || err != nil {
		return err
	}
	listProfiles := func(names []string) {
		fmt.Fprintln(out, "\nKnown profiles:")
		for _, name := range names {
			marker := ""
			if name == active {
				marker = " (active)"
			}
			fmt.Fprintf(out, "  - %s%s\n", name, marker)
		}
	}

	fmt.Fprintln(out, "\nManage Gormes profiles and their workspaces.")
	listProfiles(known)

	newName, err := promptSetupProfilesString(cmd, "\nCreate a new profile? Enter a name (blank to skip): ", "")
	if err != nil {
		return err
	}
	if newName = strings.TrimSpace(newName); newName != "" {
		if pseams.ValidateProfileName != nil {
			if verr := pseams.ValidateProfileName(newName); verr != nil {
				return fmt.Errorf("invalid profile name %q: %w", newName, verr)
			}
		}
		if pseams.CreateProfile == nil {
			return fmt.Errorf("profile creation seam unavailable")
		}
		if _, cerr := pseams.CreateProfile(newName, false); cerr != nil {
			return fmt.Errorf("create profile %q: %w", newName, cerr)
		}
		fmt.Fprintf(out, "Created profile %q (~/.gormes/profiles/%s).\n", newName, newName)
		if refreshed, rerr := pseams.ListKnownProfiles(); rerr == nil {
			known = refreshed
		}
		listProfiles(known)
	}

	selected, err := promptSetupProfilesString(cmd, fmt.Sprintf("\nSelect a profile to set workspaces for [%s]: ", active), active)
	if err != nil {
		return err
	}
	selected = strings.TrimSpace(selected)
	if selected == "" {
		selected = active
	}
	if pseams.ResolveProfileRoot == nil {
		return fmt.Errorf("profile root seam unavailable")
	}
	root, err := pseams.ResolveProfileRoot(selected)
	if err != nil {
		return fmt.Errorf("resolve profile %q: %w", selected, err)
	}
	writeSetupProfileStorageSummary(out, root)

	profileConfigPath := filepath.Join(root, "config.toml")

	wsInput, err := promptSetupProfilesString(cmd, "Workspace directories (comma-separated, blank to keep current): ", "")
	if err != nil {
		return err
	}
	if strings.TrimSpace(wsInput) == "" {
		fmt.Fprintf(out, "No workspace change for profile %q.\n", selected)
	} else {
		if err := config.WriteTOMLValue(profileConfigPath, "agents.defaults.workspaces", wsInput); err != nil {
			return fmt.Errorf("persist workspaces for profile %q: %w", selected, err)
		}
		fmt.Fprintf(out, "Set %d workspace(s) for profile %q in %s.\n",
			len(parseSetupWorkspaceList(wsInput)), selected, setupRedactedProfileConfigPath(root))
	}

	chInput, err := promptSetupProfilesString(cmd, "Messaging channels (comma-separated: telegram,whatsapp,discord,slack — blank to keep): ", "")
	if err != nil {
		return err
	}
	if strings.TrimSpace(chInput) == "" {
		fmt.Fprintf(out, "No channel change for profile %q.\n", selected)
		return nil
	}
	validChannels, unknownChannels := parseSetupChannelList(chInput)
	for _, u := range unknownChannels {
		fmt.Fprintf(out, "Skipping unknown channel %q (known: %s).\n", u, setupKnownChannelsLabel())
	}
	if len(validChannels) == 0 {
		fmt.Fprintf(out, "No valid channels for profile %q.\n", selected)
		return nil
	}
	if err := config.WriteTOMLValue(profileConfigPath, "agents.defaults.channels", strings.Join(validChannels, ",")); err != nil {
		return fmt.Errorf("persist channels for profile %q: %w", selected, err)
	}
	fmt.Fprintf(out, "Set %d channel(s) for profile %q in %s.\n", len(validChannels), selected, setupRedactedProfileConfigPath(root))
	return nil
}

// knownSetupChannels is the Gormes-owned messaging-channel set the profiles
// section accepts. Per-channel credential/token/QR/whatsapp-pairing setup is
// intentionally out of scope here — this records WHICH channels a profile
// uses, not their credentials.
var knownSetupChannels = map[string]struct{}{
	"telegram": {},
	"whatsapp": {},
	"discord":  {},
	"slack":    {},
	"navivox":  {},
}

func setupKnownChannelsLabel() string {
	channels := make([]string, 0, len(knownSetupChannels))
	for channel := range knownSetupChannels {
		channels = append(channels, channel)
	}
	sort.Strings(channels)
	return strings.Join(channels, ", ")
}

// SetupProfilesKnownChannelsLabel exposes the setup-profiles channel label to
// transitional root adapters while the profiles TUI continues moving inward.
func SetupProfilesKnownChannelsLabel() string {
	return setupKnownChannelsLabel()
}

// parseSetupChannelList splits comma-separated channel input (reusing the
// workspace-list splitter for symmetry) into validated known channels
// (lowercased) and unknown tokens that are skipped, never persisted.
func parseSetupChannelList(value string) (valid, unknown []string) {
	for _, part := range parseSetupWorkspaceList(value) {
		c := strings.ToLower(part)
		if _, ok := knownSetupChannels[c]; ok {
			valid = append(valid, c)
		} else {
			unknown = append(unknown, part)
		}
	}
	return valid, unknown
}

// ParseSetupProfilesChannelList exposes setup-profiles channel parsing to
// transitional root adapters while the profiles TUI continues moving inward.
func ParseSetupProfilesChannelList(value string) (valid, unknown []string) {
	return parseSetupChannelList(value)
}

// parseSetupWorkspaceList splits the comma-separated workspace input the same
// way the internal/config writer coerces agents.defaults.workspaces, so the
// confirmation count matches what is persisted.
func parseSetupWorkspaceList(value string) []string {
	out := make([]string, 0)
	for _, part := range strings.Split(value, ",") {
		if p := strings.TrimSpace(part); p != "" {
			out = append(out, p)
		}
	}
	return out
}

// ParseSetupProfilesWorkspaceList exposes setup-profiles workspace parsing to
// transitional root adapters while the profiles TUI continues moving inward.
func ParseSetupProfilesWorkspaceList(value string) []string {
	return parseSetupWorkspaceList(value)
}

func (opts SetupProfilesOptions) setupRequiresTTYError() error {
	if opts.RequiresTTYError != nil {
		return opts.RequiresTTYError
	}
	return fmt.Errorf("setup_requires_tty")
}

func (opts SetupProfilesOptions) profileSeams() SetupProfilesSeams {
	if opts.ProfileSeams.ListKnownProfiles != nil {
		return opts.ProfileSeams
	}
	return defaultSetupProfilesSeams()
}

func defaultSetupProfilesSeams() SetupProfilesSeams {
	baseHome := config.GormesBaseHome()
	activePath := filepath.Join(baseHome, "active_profile")
	return SetupProfilesSeams{
		ReadActiveProfileName: func() (string, error) {
			return cli.ReadActiveProfile(activePath)
		},
		ValidateProfileName: cli.ValidateProfileName,
		ResolveProfileRoot: func(name string) (string, error) {
			return cli.ResolveProfileRuntimeRoot(baseHome, name)
		},
		WriteActiveProfile: func(name string) error {
			return cli.WriteActiveProfile(activePath, name)
		},
		CreateProfile: func(name string, cloneAll bool) (cli.ProfileCreateResult, error) {
			if name == config.DefaultProfileID {
				return cli.ProfileCreateResult{}, cli.ErrProfileCreateDefaultReserved
			}
			sourceRoot := ""
			if cloneAll {
				var err error
				sourceRoot, err = cli.ResolveProfileRuntimeRoot(baseHome, config.DefaultProfileID)
				if err != nil {
					return cli.ProfileCreateResult{}, err
				}
			}
			return cli.CreateProfile(cli.ProfileCreateOptions{
				Name:       name,
				TargetRoot: filepath.Join(baseHome, "profiles", name),
				SourceRoot: sourceRoot,
				CloneAll:   cloneAll,
			})
		},
		ListKnownProfiles: defaultSetupProfilesListKnownProfiles,
	}
}

func defaultSetupProfilesListKnownProfiles() ([]string, error) {
	baseHome := config.GormesBaseHome()
	known := []string{config.DefaultProfileID}
	seen := map[string]struct{}{config.DefaultProfileID: {}}
	addName := func(name string) {
		name = strings.TrimSpace(name)
		if name == "" {
			return
		}
		if _, dup := seen[name]; dup {
			return
		}
		if err := cli.ValidateProfileName(name); err != nil {
			return
		}
		seen[name] = struct{}{}
		known = append(known, name)
	}
	if cfg, err := loadSetupProfilesConfigFromBaseHome(baseHome); err == nil {
		for name := range cfg.Profiles {
			addName(name)
		}
	}
	entries, err := os.ReadDir(filepath.Join(baseHome, "profiles"))
	if err != nil {
		return known, nil
	}
	for _, entry := range entries {
		if entry.IsDir() {
			addName(entry.Name())
		}
	}
	return known, nil
}

func loadSetupProfilesConfigFromBaseHome(baseHome string) (config.Config, error) {
	baseHome = strings.TrimSpace(baseHome)
	if baseHome == "" {
		return config.Load(nil)
	}
	currentHome := config.GormesHome()
	if filepath.Clean(currentHome) == filepath.Clean(baseHome) {
		return config.Load(nil)
	}
	rawHome, hadHome := os.LookupEnv("GORMES_HOME")
	if err := os.Setenv("GORMES_HOME", baseHome); err != nil {
		return config.Config{}, fmt.Errorf("profile config: scope base home: %w", err)
	}
	defer func() {
		if hadHome {
			_ = os.Setenv("GORMES_HOME", rawHome)
		} else {
			_ = os.Unsetenv("GORMES_HOME")
		}
	}()
	return config.Load(nil)
}

func promptSetupProfilesString(cmd *cobra.Command, prompt, defaultVal string) (string, error) {
	fmt.Fprint(cmd.OutOrStdout(), prompt)
	var input string
	_, err := fmt.Fscanln(cmd.InOrStdin(), &input)
	if err != nil {
		if err.Error() == "unexpected newline" || strings.Contains(err.Error(), "expected") {
			return defaultVal, nil
		}
		return "", err
	}
	return strings.TrimSpace(input), nil
}
