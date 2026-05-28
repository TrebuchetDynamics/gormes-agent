package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/cli"
	profilemodule "github.com/TrebuchetDynamics/gormes-agent/internal/cli/gormescli/modules/profiles"
	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	gatewaymodule "github.com/TrebuchetDynamics/gormes-agent/internal/gateway"
	providermodule "github.com/TrebuchetDynamics/gormes-agent/internal/provider"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tui"
	"github.com/charmbracelet/bubbles/filepicker"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/pelletier/go-toml/v2"
	"github.com/spf13/cobra"
)

type setupProfileView struct {
	Name           string
	DisplayName    string
	Root           string
	Active         bool
	Workspaces     []string
	Channels       []string
	ChannelDetails []setupChannelView
	Providers      []setupProviderView
}

type setupChannelView struct {
	ID                  string
	CredentialID        string
	Status              string
	AllowedChatCount    int
	AllowedUserCount    int
	RequireMention      bool
	ToolProgress        string
	SecretRefConfigured bool
	SecretRefSource     string
	Evidence            []string
}

type setupProviderView struct {
	ID            string
	CredentialID  string
	DefaultModel  string
	AllowedModels []string
	Models        []string
	Status        string
	Warnings      []string
	Evidence      []string
}

type setupProfilesTUIState struct {
	Active                string
	ControlCenter         bool
	MigrationAvailable    bool
	MigrationPreviewLines []string
	Profiles              []setupProfileView
}

type setupProfilesTUIResult struct {
	Cancelled             bool
	Discarded             bool
	MigrateLegacyConfig   bool
	CreateName            string
	Selected              string
	DisplayNameSet        bool
	DisplayName           string
	SetActive             bool
	WorkspacesSet         bool
	Workspaces            []string
	ChannelsSet           bool
	Channels              []string
	ProviderCredentialSet bool
	ProviderID            string
	ProviderCredentialID  string
	ProviderDefaultModel  string
	ProviderAllowedModels []string
	ChannelCredentialSet  bool
	ChannelPolicySet      bool
	ChannelID             string
	ChannelCredentialID   string
	ChannelAllowedChats   []string
	ChannelAllowedUsers   []string
	ChannelRequireMention bool
	ChannelToolProgress   string
}

type setupProfilesMode string

const (
	setupProfilesModeBrowse             setupProfilesMode = "browse"
	setupProfilesModeAddProfile         setupProfilesMode = "add_profile"
	setupProfilesModeDisplayName        setupProfilesMode = "display_name"
	setupProfilesModeWorkspaces         setupProfilesMode = "workspaces"
	setupProfilesModeWorkspacePath      setupProfilesMode = "workspace_path"
	setupProfilesModeWorkspaceBrowser   setupProfilesMode = "workspace_browser"
	setupProfilesModeChannels           setupProfilesMode = "channels"
	setupProfilesModeProviderCredential setupProfilesMode = "provider_credential"
	setupProfilesModeChannelCredential  setupProfilesMode = "channel_credential"
)

type setupProfilesModel struct {
	state                   setupProfilesTUIState
	selected                int
	mode                    setupProfilesMode
	input                   string
	inputField              textinput.Model
	width                   int
	height                  int
	channelDraft            map[string]bool
	channelIndex            int
	commandCursor           int
	workspaceDraft          []string
	workspaceIndex          int
	workspaceEditingIndex   int
	workspaceBrowserPath    string
	workspaceBrowserEntries []string
	workspaceBrowserIndex   int
	workspacePicker         filepicker.Model
	channelList             list.Model
	result                  setupProfilesTUIResult
	err                     error
}

const setupProfilesCommandSaveSelected = "save_selected"

type setupProfilesCommandAction struct {
	key   rune
	label string
	id    string
}

var runSetupProfilesTUI = runSetupProfilesTUIDefault
var writeSetupProfilesControlCenterConfig = config.WriteProfileConfigV2
var applySetupProfilesControlCenterMigration = config.ApplyProfileConfigV2Migration

var setupProfilesChannelChoices = []string{"telegram", "whatsapp", "discord", "slack", "navivox"}

type setupChannelListItem string

func (i setupChannelListItem) FilterValue() string { return string(i) }
func (i setupChannelListItem) Title() string       { return string(i) }
func (i setupChannelListItem) Description() string { return "profile channel" }

func maybeRunSetupProfilesTUI(cmd *cobra.Command, pseams profileCommandSeams, known []string, active string) (bool, error) {
	stdin, ok := cmd.InOrStdin().(*os.File)
	if !ok || !setupInputIsTerminal(stdin) {
		return false, nil
	}
	if cfg, err := config.Load(nil); err == nil {
		if cfg.ProfileConfigV2Available() {
			result, runErr := runSetupProfilesTUI(cmd.Context(), stdin, cmd.OutOrStdout(), buildSetupProfilesControlCenterTUIState(cfg))
			if runErr != nil {
				if bubbleTeaPickShouldFallback(runErr) {
					return false, nil
				}
				return true, runErr
			}
			if result.Cancelled {
				fmt.Fprintln(cmd.OutOrStdout(), "Setup canceled.")
				return true, nil
			}
			return true, applySetupProfilesControlCenterTUIResult(cmd, cfg, result)
		}
		if state, ok := buildSetupProfilesMigrationTUIState(); ok {
			result, runErr := runSetupProfilesTUI(cmd.Context(), stdin, cmd.OutOrStdout(), state)
			if runErr != nil {
				if bubbleTeaPickShouldFallback(runErr) {
					return false, nil
				}
				return true, runErr
			}
			if result.Cancelled {
				fmt.Fprintln(cmd.OutOrStdout(), "Setup canceled.")
				return true, nil
			}
			return true, applySetupProfilesControlCenterTUIResult(cmd, cfg, result)
		}
	}
	result, err := runSetupProfilesTUI(cmd.Context(), stdin, cmd.OutOrStdout(), buildSetupProfilesTUIState(pseams, known, active))
	if err != nil {
		if bubbleTeaPickShouldFallback(err) {
			return false, nil
		}
		return true, err
	}
	if result.Cancelled {
		fmt.Fprintln(cmd.OutOrStdout(), "Setup canceled.")
		return true, nil
	}
	return true, applySetupProfilesTUIResult(cmd, pseams, active, result)
}

func buildSetupProfilesControlCenterTUIState(cfg config.Config) setupProfilesTUIState {
	model := profilemodule.BuildControlCenterModel(cfg, profilemodule.ControlCenterModelOptions{})
	providerViews := setupProfilesProviderViews(cfg)
	channelViews := setupProfilesChannelViews(cfg)
	profiles := make([]setupProfileView, 0, len(model.Profiles))
	for _, profile := range model.Profiles {
		view := setupProfileView{Name: profile.ID, DisplayName: profile.DisplayName, Workspaces: controlCenterSetupWorkspacePaths(profile.Workspaces), ChannelDetails: channelViews[profile.ID], Providers: providerViews[profile.ID]}
		for _, channel := range profile.Channels {
			if channel.Enabled {
				view.Channels = append(view.Channels, channel.ID)
			}
		}
		profiles = append(profiles, view)
	}
	return setupProfilesTUIState{ControlCenter: true, Profiles: profiles}
}

func setupProfilesChannelViews(cfg config.Config) map[string][]setupChannelView {
	report := gatewaymodule.BuildProfileChannelReadiness(cfg)
	out := map[string][]setupChannelView{}
	for _, binding := range report.Bindings {
		status := "degraded"
		if binding.Ready {
			status = "ready"
		}
		out[binding.ProfileID] = append(out[binding.ProfileID], setupChannelView{
			ID:                  binding.Channel,
			CredentialID:        binding.CredentialID,
			Status:              status,
			AllowedChatCount:    binding.AllowedChatCount,
			AllowedUserCount:    binding.AllowedUserCount,
			RequireMention:      binding.RequireMention,
			ToolProgress:        binding.ToolProgress,
			SecretRefConfigured: binding.SecretRefConfigured,
			SecretRefSource:     binding.SecretRefSource,
			Evidence:            setupProfilesChannelEvidenceCodes(binding.Evidence),
		})
	}
	return out
}

func setupProfilesChannelEvidenceCodes(evidence []gatewaymodule.ProfileChannelReadinessEvidence) []string {
	out := make([]string, 0, len(evidence))
	for _, item := range evidence {
		if strings.TrimSpace(item.Code) != "" {
			out = append(out, item.Code)
		}
	}
	return out
}

func setupProfilesProviderViews(cfg config.Config) map[string][]setupProviderView {
	reports := providermodule.BuildProfileProviderReadiness(cfg, providermodule.ProfileProviderReadinessOptions{Catalogs: setupProfilesProviderCatalogs(cfg)})
	out := map[string][]setupProviderView{}
	for _, report := range reports {
		out[report.ProfileID] = append(out[report.ProfileID], setupProviderView{
			ID:            report.ProviderID,
			CredentialID:  report.CredentialID,
			DefaultModel:  report.DefaultModel,
			AllowedModels: append([]string(nil), report.AllowedModels...),
			Models:        append([]string(nil), report.Models...),
			Status:        string(report.Status),
			Warnings:      append([]string(nil), report.Warnings...),
			Evidence:      append([]string(nil), report.Evidence...),
		})
	}
	return out
}

func setupProfilesProviderCatalogs(cfg config.Config) map[string]providermodule.ProviderModelCatalogFunc {
	seen := map[string]bool{}
	for _, profile := range cfg.Profiles {
		if !profile.Enabled {
			continue
		}
		for providerID, providerCfg := range profile.Providers {
			providerID = strings.ToLower(strings.TrimSpace(providerID))
			if providerID != "" && providerCfg.Enabled {
				seen[providerID] = true
			}
		}
	}
	catalogs := map[string]providermodule.ProviderModelCatalogFunc{}
	for providerID := range seen {
		id := providerID
		catalogs[id] = func() ([]string, error) {
			set := defaultModelPickerSuggestionSet(id)
			if len(set.Models) == 0 && set.DegradedReason != "" {
				return nil, fmt.Errorf("%s", set.DegradedReason)
			}
			return append([]string(nil), set.Models...), nil
		}
	}
	return catalogs
}

func buildSetupProfilesMigrationTUIState() (setupProfilesTUIState, bool) {
	plan, err := config.PlanProfileConfigV2Migration(config.ProfileMigrationV2Options{})
	if err != nil || plan.NoOp {
		return setupProfilesTUIState{}, false
	}
	profiles := make([]setupProfileView, 0, len(plan.ProfileAdditions))
	for _, profile := range plan.ProfileAdditions {
		profiles = append(profiles, setupProfileView{Name: profile.ID, DisplayName: profile.DisplayName, Workspaces: append([]string(nil), profile.Workspaces...), Channels: append([]string(nil), profile.Channels...)})
	}
	if len(profiles) == 0 {
		profiles = []setupProfileView{{Name: config.DefaultProfileID}}
	}
	return setupProfilesTUIState{ControlCenter: true, MigrationAvailable: true, MigrationPreviewLines: append([]string(nil), plan.PreviewLines...), Profiles: profiles}, true
}

func controlCenterSetupWorkspacePaths(workspaces []profilemodule.ControlCenterWorkspace) []string {
	out := make([]string, 0, len(workspaces))
	for _, workspace := range workspaces {
		out = append(out, workspace.Path)
	}
	return out
}

func buildSetupProfilesTUIState(pseams profileCommandSeams, known []string, active string) setupProfilesTUIState {
	sorted := append([]string(nil), known...)
	sort.Strings(sorted)
	profiles := make([]setupProfileView, 0, len(sorted))
	for _, name := range sorted {
		view := setupProfileView{Name: name, Active: name == active}
		if pseams.ResolveProfileRoot != nil {
			if root, err := pseams.ResolveProfileRoot(name); err == nil {
				view.Root = root
				view.Workspaces, view.Channels = readSetupProfileDefaults(root)
			}
		}
		profiles = append(profiles, view)
	}
	return setupProfilesTUIState{Active: active, Profiles: profiles}
}

func applySetupProfilesControlCenterTUIResult(cmd *cobra.Command, cfg config.Config, result setupProfilesTUIResult) error {
	out := cmd.OutOrStdout()
	if result.Discarded {
		fmt.Fprintln(out, "Setup profiles draft discarded; no config changes written.")
		return nil
	}
	if result.MigrateLegacyConfig {
		return applySetupProfilesLegacyMigration(out)
	}
	draft := profilemodule.NewControlCenterDraft(cfg)
	createName := strings.TrimSpace(result.CreateName)
	if createName != "" {
		name := strings.TrimSpace(result.DisplayName)
		if name == "" {
			name = createName
		}
		if err := draft.AddProfile(createName, name); err != nil {
			return err
		}
	}
	selected := strings.TrimSpace(result.Selected)
	if selected == "" {
		selected = config.DefaultProfileID
	}
	if result.DisplayNameSet {
		if err := draft.SetProfileDisplayName(selected, result.DisplayName); err != nil {
			return err
		}
	}
	if result.WorkspacesSet {
		if err := draft.SetProfileWorkspaces(selected, normalizeSetupProfilesTUIValues(result.Workspaces)); err != nil {
			return err
		}
	}
	if result.ProviderCredentialSet {
		credential := setupProfilesProviderCredential(selected, result.ProviderID)
		if err := draft.SetCredential(result.ProviderCredentialID, credential); err != nil {
			return err
		}
		if err := draft.AssignProviderCredential(selected, result.ProviderID, result.ProviderCredentialID); err != nil {
			return err
		}
		if result.ProviderDefaultModel != "" || len(result.ProviderAllowedModels) > 0 {
			if err := draft.SetProfileProviderModels(selected, result.ProviderID, result.ProviderDefaultModel, result.ProviderAllowedModels); err != nil {
				return err
			}
		}
	}
	if result.ChannelCredentialSet {
		credential := setupProfilesChannelCredential(selected, result.ChannelID)
		if err := draft.SetCredential(result.ChannelCredentialID, credential); err != nil {
			return err
		}
		if err := draft.AssignChannelCredential(selected, result.ChannelID, result.ChannelCredentialID); err != nil {
			return err
		}
		if result.ChannelPolicySet {
			if err := draft.SetProfileChannelPolicy(selected, result.ChannelID, result.ChannelAllowedChats, result.ChannelAllowedUsers, result.ChannelRequireMention, result.ChannelToolProgress); err != nil {
				return err
			}
		}
	}
	if result.ChannelsSet {
		validChannels, unknownChannels := parseSetupChannelList(strings.Join(result.Channels, ","))
		for _, u := range unknownChannels {
			fmt.Fprintf(out, "Skipping unknown channel %q (known: %s).\n", u, setupKnownChannelsLabel())
		}
		if err := draft.SetProfileChannels(selected, validChannels); err != nil {
			return err
		}
	}
	changes := draft.Preview()
	fmt.Fprintln(out, "Setup profiles draft:")
	for _, line := range profilemodule.RenderControlCenterDraftPreview(changes) {
		fmt.Fprintf(out, "  - %s\n", line)
	}
	if len(changes) == 0 {
		fmt.Fprintln(out, "No profile setup changes selected.")
		return nil
	}
	applied, changes, err := draft.Apply()
	if err != nil {
		return err
	}
	if writeSetupProfilesControlCenterConfig == nil {
		return fmt.Errorf("profile control center root config writer unavailable")
	}
	if err := materializeSetupProfilesControlCenterMainProfile(); err != nil {
		return err
	}
	if createName != "" {
		if err := materializeSetupProfilesControlCenterProfile(createName); err != nil {
			return err
		}
	}
	configPath := config.ConfigPath()
	if err := writeSetupProfilesControlCenterConfig(configPath, applied); err != nil {
		return fmt.Errorf("apply profile control center draft: %w", err)
	}
	fmt.Fprintf(out, "Applied %d profile control center change(s) to %s.\n", len(changes), setupRedactedProfileConfigPath(config.GormesHome()))
	return nil
}

func materializeSetupProfilesControlCenterMainProfile() error {
	if _, err := cli.MaterializeMainProfileContextScaffold(cli.ProfileContextScaffoldOptions{BaseHome: config.GormesBaseHome()}); err != nil {
		return fmt.Errorf("materialize main profile context: %w", err)
	}
	return nil
}

func materializeSetupProfilesControlCenterProfile(name string) error {
	seams := defaultProfileCommandSeams()
	if seams.CreateProfile == nil {
		return fmt.Errorf("profile creation seam unavailable")
	}
	if _, err := seams.CreateProfile(name, false); err != nil && !errors.Is(err, cli.ErrProfileCreateTargetExists) {
		return fmt.Errorf("create profile runtime home %q: %w", name, err)
	}
	return nil
}

func applySetupProfilesLegacyMigration(out io.Writer) error {
	if applySetupProfilesControlCenterMigration == nil {
		return fmt.Errorf("profile control center migration seam unavailable")
	}
	result, err := applySetupProfilesControlCenterMigration(config.ProfileMigrationV2Options{})
	if err != nil {
		return fmt.Errorf("apply legacy profile config migration: %w", err)
	}
	fmt.Fprintln(out, "Setup profiles migration:")
	for _, line := range result.Plan.PreviewLines {
		fmt.Fprintf(out, "  - %s\n", line)
	}
	if result.NoOp {
		fmt.Fprintf(out, "No legacy profile config migration needed for %s.\n", setupRedactedFilePath(result.Path))
		return nil
	}
	if result.Wrote {
		fmt.Fprintf(out, "Applied legacy profile config migration to %s.\n", setupRedactedFilePath(result.Path))
		if result.BackupPath != "" {
			fmt.Fprintf(out, "Backup: %s\n", setupRedactedFilePath(result.BackupPath))
		}
	}
	return nil
}

func applySetupProfilesTUIResult(cmd *cobra.Command, pseams profileCommandSeams, active string, result setupProfilesTUIResult) error {
	out := cmd.OutOrStdout()
	createName := strings.TrimSpace(result.CreateName)
	if createName != "" {
		if pseams.ValidateProfileName != nil {
			if err := pseams.ValidateProfileName(createName); err != nil {
				return fmt.Errorf("invalid profile name %q: %w", createName, err)
			}
		}
		if pseams.CreateProfile == nil {
			return fmt.Errorf("profile creation seam unavailable")
		}
		if _, err := pseams.CreateProfile(createName, false); err != nil {
			return fmt.Errorf("create profile %q: %w", createName, err)
		}
		fmt.Fprintf(out, "Created profile %q (~/.gormes/profiles/%s).\n", createName, createName)
	}

	selected := strings.TrimSpace(result.Selected)
	if selected == "" {
		selected = strings.TrimSpace(active)
	}
	if selected == "" {
		selected = config.DefaultProfileID
	}
	if pseams.ResolveProfileRoot == nil {
		return fmt.Errorf("profile root seam unavailable")
	}
	root, err := pseams.ResolveProfileRoot(selected)
	if err != nil {
		return fmt.Errorf("resolve profile %q: %w", selected, err)
	}
	writeSetupProfileStorageSummary(out, root)
	if result.SetActive {
		if pseams.WriteActiveProfile == nil {
			return fmt.Errorf("profile active seam unavailable")
		}
		if pseams.ValidateProfileName != nil {
			if err := pseams.ValidateProfileName(selected); err != nil {
				return fmt.Errorf("invalid profile name %q: %w", selected, err)
			}
		}
		if err := pseams.WriteActiveProfile(selected); err != nil {
			return fmt.Errorf("set active profile %q: %w", selected, err)
		}
		fmt.Fprintf(out, "Active profile set to %q.\n", selected)
	}

	profileConfigPath := filepath.Join(root, "config.toml")
	profileConfigDisplayPath := setupRedactedProfileConfigPath(root)
	if result.WorkspacesSet {
		workspaces := normalizeSetupProfilesTUIValues(result.Workspaces)
		if err := config.WriteTOMLValue(profileConfigPath, "agents.defaults.workspaces", strings.Join(workspaces, ",")); err != nil {
			return fmt.Errorf("persist workspaces for profile %q: %w", selected, err)
		}
		fmt.Fprintf(out, "Set %d workspace(s) for profile %q in %s.\n", len(workspaces), selected, profileConfigDisplayPath)
	}
	if result.ChannelsSet {
		validChannels, unknownChannels := parseSetupChannelList(strings.Join(result.Channels, ","))
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
		fmt.Fprintf(out, "Set %d channel(s) for profile %q in %s.\n", len(validChannels), selected, profileConfigDisplayPath)
	}
	if createName == "" && !result.SetActive && !result.WorkspacesSet && !result.ChannelsSet {
		fmt.Fprintln(out, "No profile setup changes selected.")
	}
	return nil
}

func writeSetupProfileStorageSummary(out io.Writer, root string) {
	redacted := setupRedactedProfileRoot(root)
	fmt.Fprintf(out, "memory_db: %s\n", filepath.ToSlash(filepath.Join(redacted, "memory.db")))
	fmt.Fprintf(out, "goncho_db: %s\n", filepath.ToSlash(filepath.Join(redacted, "memory.db")))
	fmt.Fprintf(out, "sessions_db: %s\n", filepath.ToSlash(filepath.Join(redacted, "sessions.db")))
}

func setupRedactedProfileRoot(root string) string {
	base := filepath.Base(filepath.Clean(strings.TrimSpace(root)))
	if base == "" || base == "." || base == string(filepath.Separator) {
		return "..."
	}
	return ".../" + base
}

func setupRedactedProfileConfigPath(root string) string {
	return filepath.ToSlash(filepath.Join(setupRedactedProfileRoot(root), "config.toml"))
}

func setupRedactedFilePath(path string) string {
	cleaned := filepath.Clean(strings.TrimSpace(path))
	if cleaned == "" || cleaned == "." || cleaned == string(filepath.Separator) {
		return "..."
	}
	parent := filepath.Base(filepath.Dir(cleaned))
	base := filepath.Base(cleaned)
	if parent == "." || parent == string(filepath.Separator) || parent == "" {
		return ".../" + base
	}
	return filepath.ToSlash(filepath.Join("...", parent, base))
}

func setupProfilesProviderCredential(profileID, providerID string) config.CredentialCfg {
	profileID = strings.ToLower(strings.TrimSpace(profileID))
	providerID = strings.ToLower(strings.TrimSpace(providerID))
	return config.CredentialCfg{
		Kind:         "provider",
		Provider:     providerID,
		OwnerProfile: profileID,
		SecretRef:    &config.SecretRef{Source: config.SecretRefSourceEnv, ID: setupProfilesProviderEnv(profileID, providerID)},
	}
}

func setupProfilesChannelCredential(profileID, channelID string) config.CredentialCfg {
	profileID = strings.ToLower(strings.TrimSpace(profileID))
	channelID = strings.ToLower(strings.TrimSpace(channelID))
	return config.CredentialCfg{
		Kind:         "channel",
		Channel:      channelID,
		OwnerProfile: profileID,
		SecretRef:    &config.SecretRef{Source: config.SecretRefSourceEnv, ID: setupProfilesChannelEnv(profileID, channelID)},
	}
}

func setupProfilesProviderEnv(profileID, providerID string) string {
	return "GORMES_" + setupProfilesEnvPart(profileID) + "_" + setupProfilesEnvPart(providerID) + "_API_KEY"
}

func setupProfilesChannelEnv(profileID, channelID string) string {
	prefix := "GORMES_" + setupProfilesEnvPart(profileID) + "_"
	if strings.EqualFold(channelID, "telegram") {
		return prefix + "TELEGRAM_BOT_TOKEN"
	}
	if strings.EqualFold(channelID, "slack_app") {
		return prefix + "SLACK_APP_TOKEN"
	}
	return prefix + setupProfilesEnvPart(channelID) + "_TOKEN"
}

func setupProfilesEnvPart(value string) string {
	value = strings.ToUpper(strings.TrimSpace(value))
	return strings.NewReplacer("-", "_", ".", "_", "/", "_").Replace(value)
}

func normalizeSetupProfilesTUIValues(values []string) []string {
	normalized := make([]string, 0, len(values))
	seen := map[string]struct{}{}
	for _, value := range values {
		for _, part := range parseSetupWorkspaceList(value) {
			if _, ok := seen[part]; ok {
				continue
			}
			seen[part] = struct{}{}
			normalized = append(normalized, part)
		}
	}
	return normalized
}

func readSetupProfileDefaults(root string) ([]string, []string) {
	raw, err := os.ReadFile(filepath.Join(root, "config.toml"))
	if err != nil {
		return nil, nil
	}
	var doc struct {
		Agents struct {
			Defaults struct {
				Workspaces []string `toml:"workspaces"`
				Channels   []string `toml:"channels"`
			} `toml:"defaults"`
		} `toml:"agents"`
	}
	if err := toml.Unmarshal(raw, &doc); err != nil {
		return nil, nil
	}
	return doc.Agents.Defaults.Workspaces, doc.Agents.Defaults.Channels
}

func runSetupProfilesTUIDefault(ctx context.Context, stdin *os.File, out io.Writer, state setupProfilesTUIState) (setupProfilesTUIResult, error) {
	model, err := tea.NewProgram(
		newSetupProfilesModel(state),
		tea.WithContext(ctx),
		tea.WithInput(stdin),
		tea.WithOutput(out),
		tea.WithAltScreen(),
	).Run()
	if err != nil {
		return setupProfilesTUIResult{}, err
	}
	profilesModel, ok := model.(setupProfilesModel)
	if !ok {
		return setupProfilesTUIResult{}, fmt.Errorf("setup profiles TUI returned %T", model)
	}
	return profilesModel.result, profilesModel.err
}

func newSetupProfilesTextInput() textinput.Model {
	input := textinput.New()
	input.Prompt = ""
	input.CharLimit = 0
	input.Focus()
	return input
}

func newSetupProfilesChannelList(width, height int) list.Model {
	items := make([]list.Item, 0, len(setupProfilesChannelChoices))
	for _, channel := range setupProfilesChannelChoices {
		items = append(items, setupChannelListItem(channel))
	}
	delegate := list.NewDefaultDelegate()
	delegate.ShowDescription = false
	delegate.SetSpacing(0)
	model := list.New(items, delegate, max(20, width), max(3, height))
	model.Title = "Channels"
	model.SetShowTitle(false)
	model.SetShowStatusBar(false)
	model.SetShowPagination(false)
	model.SetFilteringEnabled(false)
	model.SetShowHelp(false)
	return model
}

func newSetupProfilesWorkspacePicker(path string, height int) filepicker.Model {
	picker := filepicker.New()
	picker.CurrentDirectory = path
	picker.DirAllowed = true
	picker.FileAllowed = false
	picker.ShowPermissions = false
	picker.ShowSize = false
	picker.ShowHidden = false
	picker.AutoHeight = false
	picker.SetHeight(max(3, height))
	picker.Cursor = "❯"
	picker.KeyMap.Select = key.NewBinding(key.WithKeys(" "), key.WithHelp("space", "select"))
	picker.KeyMap.Open = key.NewBinding(key.WithKeys("enter", "l", "right"), key.WithHelp("enter", "open"))
	return picker
}

func newSetupProfilesModel(state setupProfilesTUIState) setupProfilesModel {
	if strings.TrimSpace(state.Active) == "" {
		state.Active = config.DefaultProfileID
	}
	if len(state.Profiles) == 0 {
		state.Profiles = []setupProfileView{{Name: config.DefaultProfileID, Active: true}}
	}
	selected := 0
	for i := range state.Profiles {
		if state.Profiles[i].Name == state.Active || state.Profiles[i].Active {
			selected = i
			break
		}
	}
	return setupProfilesModel{
		state:        state,
		selected:     selected,
		mode:         setupProfilesModeBrowse,
		inputField:   newSetupProfilesTextInput(),
		width:        80,
		height:       24,
		channelDraft: make(map[string]bool),
		channelList:  newSetupProfilesChannelList(80, 8),
	}
}

func (m setupProfilesModel) Init() tea.Cmd {
	return nil
}

func (m setupProfilesModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if size, ok := msg.(tea.WindowSizeMsg); ok {
		m.width = size.Width
		m.height = size.Height
		m.channelList.SetSize(max(20, size.Width), max(3, min(8, size.Height-6)))
		m.workspacePicker.SetHeight(max(3, min(10, size.Height-6)))
		return m, nil
	}
	if m.mode == setupProfilesModeWorkspaceBrowser {
		picker, cmd := m.workspacePicker.Update(msg)
		m.workspacePicker = picker
		if _, ok := msg.(tea.KeyMsg); !ok {
			return m, cmd
		}
	}
	key, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}
	switch key.Type {
	case tea.KeyCtrlC:
		m.result.Cancelled = true
		return m, tea.Quit
	case tea.KeyEsc:
		if m.mode == setupProfilesModeWorkspaceBrowser || m.mode == setupProfilesModeWorkspacePath {
			m.mode = setupProfilesModeWorkspaces
			m = m.setInput("")
			return m, nil
		}
		if m.mode != setupProfilesModeBrowse {
			m.mode = setupProfilesModeBrowse
			m = m.setInput("")
			return m, nil
		}
		m.result.Cancelled = true
		return m, tea.Quit
	case tea.KeyEnter:
		return m.handleEnter()
	}
	if m.isTextInputMode() {
		return m.updateInputField(key)
	}
	switch key.Type {
	case tea.KeySpace:
		if m.mode == setupProfilesModeWorkspaceBrowser {
			if m.workspacePicker.Path != "" {
				return m.selectWorkspaceBrowserPath(m.workspacePicker.Path), nil
			}
			return m.selectWorkspaceBrowserEntry(), nil
		}
		if m.mode == setupProfilesModeChannels {
			channel := setupProfilesSelectedChannel(m.channelList)
			if channel != "" {
				m.channelDraft[channel] = !m.channelDraft[channel]
			}
		}
		return m, nil
	case tea.KeyUp:
		if m.mode == setupProfilesModeWorkspaces && m.workspaceIndex > 0 {
			m.workspaceIndex--
		} else if m.mode == setupProfilesModeChannels {
			var cmd tea.Cmd
			m.channelList, cmd = m.channelList.Update(key)
			m.channelIndex = m.channelList.Index()
			return m, cmd
		} else if m.mode == setupProfilesModeBrowse && m.selected > 0 {
			m.selected--
		}
		return m, nil
	case tea.KeyDown:
		if m.mode == setupProfilesModeWorkspaces && m.workspaceIndex < len(m.workspaceDraft)-1 {
			m.workspaceIndex++
		} else if m.mode == setupProfilesModeChannels {
			var cmd tea.Cmd
			m.channelList, cmd = m.channelList.Update(key)
			m.channelIndex = m.channelList.Index()
			return m, cmd
		} else if m.mode == setupProfilesModeBrowse && m.selected < len(m.state.Profiles)-1 {
			m.selected++
		}
		return m, nil
	case tea.KeyLeft:
		if m.mode == setupProfilesModeBrowse {
			m = m.moveCommandCursor(-1)
		}
		return m, nil
	case tea.KeyRight:
		if m.mode == setupProfilesModeBrowse {
			m = m.moveCommandCursor(1)
		}
		return m, nil
	case tea.KeyRunes:
		return m.handleRunes(key.Runes)
	}
	return m, nil
}

func (m setupProfilesModel) isTextInputMode() bool {
	switch m.mode {
	case setupProfilesModeAddProfile, setupProfilesModeDisplayName, setupProfilesModeWorkspacePath, setupProfilesModeProviderCredential, setupProfilesModeChannelCredential:
		return true
	default:
		return false
	}
}

func (m setupProfilesModel) updateInputField(msg tea.Msg) (tea.Model, tea.Cmd) {
	input := m.inputField
	input.Width = max(1, m.viewWidth()-4)
	input.Focus()
	if key, ok := msg.(tea.KeyMsg); ok && key.Type == tea.KeySpace {
		input.SetValue(input.Value() + " ")
		input.CursorEnd()
		m.inputField = input
		m.input = input.Value()
		return m, nil
	}
	var cmd tea.Cmd
	input, cmd = input.Update(msg)
	m.inputField = input
	m.input = input.Value()
	return m, cmd
}

func (m setupProfilesModel) setInput(value string) setupProfilesModel {
	m.input = value
	m.inputField.SetValue(value)
	m.inputField.CursorEnd()
	return m
}

func (m setupProfilesModel) handleRunes(runes []rune) (tea.Model, tea.Cmd) {
	if m.mode == setupProfilesModeWorkspaces {
		if len(runes) != 1 {
			return m, nil
		}
		switch runes[0] {
		case 'j':
			if m.workspaceIndex < len(m.workspaceDraft)-1 {
				m.workspaceIndex++
			}
		case 'k':
			if m.workspaceIndex > 0 {
				m.workspaceIndex--
			}
		case 'a':
			m.mode = setupProfilesModeWorkspacePath
			m.workspaceEditingIndex = -1
			m = m.setInput("")
		case 'e':
			if len(m.workspaceDraft) > 0 {
				m.mode = setupProfilesModeWorkspacePath
				m.workspaceEditingIndex = m.workspaceIndex
				m = m.setInput(m.workspaceDraft[m.workspaceIndex])
			}
		case 'x':
			m = m.removeSelectedWorkspace()
		case 'p':
			m = m.setSelectedWorkspacePrimary()
		case 'f':
			return m.openWorkspaceBrowserPath(m.initialWorkspaceBrowserPath())
		case 'b':
			m.mode = setupProfilesModeBrowse
		}
		return m, nil
	}
	if m.mode == setupProfilesModeWorkspaceBrowser {
		if len(runes) != 1 {
			return m, nil
		}
		switch runes[0] {
		case 'b', 'q':
			m.mode = setupProfilesModeWorkspaces
			return m, nil
		case 'u':
			return m.openWorkspaceBrowserPath(filepath.Dir(m.workspacePicker.CurrentDirectory))
		default:
			var cmd tea.Cmd
			m.workspacePicker, cmd = m.workspacePicker.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: runes})
			return m, cmd
		}
	}
	if m.mode == setupProfilesModeChannels {
		if len(runes) != 1 {
			return m, nil
		}
		switch runes[0] {
		case 'j', 'k':
			msg := tea.KeyMsg{Type: tea.KeyDown}
			if runes[0] == 'k' {
				msg = tea.KeyMsg{Type: tea.KeyUp}
			}
			var cmd tea.Cmd
			m.channelList, cmd = m.channelList.Update(msg)
			m.channelIndex = m.channelList.Index()
			return m, cmd
		case 'q':
			m.mode = setupProfilesModeBrowse
		}
		return m, nil
	}
	if m.mode != setupProfilesModeBrowse {
		m = m.setInput(m.input + string(runes))
		return m, nil
	}
	if len(runes) != 1 {
		return m, nil
	}
	switch runes[0] {
	case 'j':
		if m.selected < len(m.state.Profiles)-1 {
			m.selected++
		}
	case 'k':
		if m.selected > 0 {
			m.selected--
		}
	case 'n':
		m.mode = setupProfilesModeAddProfile
		m = m.setInput("")
	case 'r':
		if m.state.ControlCenter {
			m.mode = setupProfilesModeDisplayName
			m = m.setInput(strings.TrimSpace(m.currentProfile().DisplayName))
		}
	case 'w':
		m = m.openWorkspaceEditor()
	case 'c':
		m.mode = setupProfilesModeChannels
		m.channelIndex = 0
		m.channelDraft = make(map[string]bool)
		for _, channel := range m.currentProfile().Channels {
			m.channelDraft[channel] = true
		}
	case 'p':
		if m.state.ControlCenter {
			m.mode = setupProfilesModeProviderCredential
			m = m.setInput("")
		}
	case 't':
		if m.state.ControlCenter {
			m.mode = setupProfilesModeChannelCredential
			m = m.setInput("")
		}
	case 'a':
		if !m.state.ControlCenter {
			profile := m.currentProfile()
			for i := range m.state.Profiles {
				m.state.Profiles[i].Active = i == m.selected
			}
			m.state.Active = profile.Name
			m.result.Selected = profile.Name
			m.result.SetActive = true
		}
	case 'm':
		if m.state.ControlCenter && m.state.MigrationAvailable {
			m.result.MigrateLegacyConfig = true
		}
	case 'd':
		if m.state.ControlCenter {
			m.result.Selected = m.currentProfile().Name
			m.result.Discarded = true
			return m, tea.Quit
		}
	case 's':
		if m.result.Selected == "" {
			m.result.Selected = m.currentProfile().Name
		}
		return m, tea.Quit
	case 'q':
		m.result.Cancelled = true
		return m, tea.Quit
	}
	return m, nil
}

func (m setupProfilesModel) handleEnter() (tea.Model, tea.Cmd) {
	value := strings.TrimSpace(m.input)
	switch m.mode {
	case setupProfilesModeBrowse:
		return m.activateCommand(m.currentCommandAction())
	case setupProfilesModeAddProfile:
		if value != "" {
			id, displayName := parseSetupProfilesNewProfileInput(value)
			view := setupProfileView{Name: id, DisplayName: displayName, Root: id}
			m.state.Profiles = append(m.state.Profiles, view)
			m.selected = len(m.state.Profiles) - 1
			m.result.CreateName = id
			m.result.Selected = id
			if m.state.ControlCenter {
				m.result.DisplayNameSet = true
				m.result.DisplayName = displayName
			}
		}
	case setupProfilesModeDisplayName:
		m.result.Selected = m.currentProfile().Name
		m.result.DisplayNameSet = true
		m.result.DisplayName = value
		m.state.Profiles[m.selected].DisplayName = value
	case setupProfilesModeWorkspaces:
		m.result.Selected = m.currentProfile().Name
		m.result.WorkspacesSet = true
		m.result.Workspaces = normalizeSetupProfilesTUIValues(m.workspaceDraft)
		m.state.Profiles[m.selected].Workspaces = append([]string(nil), m.result.Workspaces...)
	case setupProfilesModeWorkspacePath:
		if value != "" {
			if m.workspaceEditingIndex >= 0 && m.workspaceEditingIndex < len(m.workspaceDraft) {
				m.workspaceDraft[m.workspaceEditingIndex] = value
			} else {
				m.workspaceDraft = append(m.workspaceDraft, value)
				m.workspaceIndex = len(m.workspaceDraft) - 1
			}
		}
		m.mode = setupProfilesModeWorkspaces
		m = m.setInput("")
		return m, nil
	case setupProfilesModeWorkspaceBrowser:
		var cmd tea.Cmd
		m.workspacePicker, cmd = m.workspacePicker.Update(tea.KeyMsg{Type: tea.KeyEnter})
		return m, cmd
	case setupProfilesModeChannels:
		m.result.Selected = m.currentProfile().Name
		m.result.ChannelsSet = true
		m.result.Channels = nil
		for _, channel := range setupProfilesChannelChoices {
			if m.channelDraft[channel] {
				m.result.Channels = append(m.result.Channels, channel)
			}
		}
		m.state.Profiles[m.selected].Channels = append([]string(nil), m.result.Channels...)
	case setupProfilesModeProviderCredential:
		providerID, credentialID, defaultModel, allowedModels := parseSetupProfilesProviderAssignment(value)
		if providerID != "" && credentialID != "" {
			m.result.Selected = m.currentProfile().Name
			m.result.ProviderCredentialSet = true
			m.result.ProviderID = providerID
			m.result.ProviderCredentialID = credentialID
			m.result.ProviderDefaultModel = defaultModel
			m.result.ProviderAllowedModels = allowedModels
		}
	case setupProfilesModeChannelCredential:
		channelID, credentialID, allowedChats, allowedUsers, requireMention, toolProgress, policySet := parseSetupProfilesChannelAssignment(value)
		if channelID != "" && credentialID != "" {
			m.result.Selected = m.currentProfile().Name
			m.result.ChannelCredentialSet = true
			m.result.ChannelPolicySet = policySet
			m.result.ChannelID = channelID
			m.result.ChannelCredentialID = credentialID
			m.result.ChannelAllowedChats = allowedChats
			m.result.ChannelAllowedUsers = allowedUsers
			m.result.ChannelRequireMention = requireMention
			m.result.ChannelToolProgress = toolProgress
		}
	}
	m.mode = setupProfilesModeBrowse
	m = m.setInput("")
	return m, nil
}

func (m setupProfilesModel) moveCommandCursor(delta int) setupProfilesModel {
	actions := m.commandActions()
	if len(actions) == 0 {
		m.commandCursor = 0
		return m
	}
	m.commandCursor = (m.commandCursor + delta) % len(actions)
	if m.commandCursor < 0 {
		m.commandCursor += len(actions)
	}
	return m
}

func (m setupProfilesModel) currentCommandAction() setupProfilesCommandAction {
	actions := m.commandActions()
	if len(actions) == 0 {
		return setupProfilesCommandAction{id: setupProfilesCommandSaveSelected, label: "Enter save selected profile"}
	}
	if m.commandCursor < 0 || m.commandCursor >= len(actions) {
		return actions[0]
	}
	return actions[m.commandCursor]
}

func (m setupProfilesModel) commandActions() []setupProfilesCommandAction {
	if m.state.ControlCenter {
		actions := []setupProfilesCommandAction{
			{id: setupProfilesCommandSaveSelected, label: "choose"},
			{key: 'n', label: "n new"},
			{key: 'r', label: "r rename"},
			{key: 'w', label: "w workspaces"},
			{key: 'c', label: "c channels"},
			{key: 'p', label: "p provider"},
			{key: 't', label: "t policy"},
		}
		if m.state.MigrationAvailable {
			actions = append(actions, setupProfilesCommandAction{key: 'm', label: "m migrate"})
		}
		return append(actions,
			setupProfilesCommandAction{key: 's', label: "s apply"},
			setupProfilesCommandAction{key: 'd', label: "d discard"},
			setupProfilesCommandAction{key: 'q', label: "q quit"},
		)
	}
	return []setupProfilesCommandAction{
		{id: setupProfilesCommandSaveSelected, label: "Enter save selected profile"},
		{key: 'n', label: "n add profile"},
		{key: 'w', label: "w edit workspaces"},
		{key: 'c', label: "c edit channels"},
		{key: 'a', label: "a set active"},
		{key: 's', label: "s save"},
		{key: 'q', label: "q cancel"},
	}
}

func (m setupProfilesModel) activateCommand(action setupProfilesCommandAction) (tea.Model, tea.Cmd) {
	if action.id == setupProfilesCommandSaveSelected {
		if m.result.Selected == "" {
			m.result.Selected = m.currentProfile().Name
		}
		return m, tea.Quit
	}
	if action.key == 0 {
		return m, nil
	}
	return m.handleRunes([]rune{action.key})
}

func (m setupProfilesModel) writeCommandActions(b *strings.Builder) {
	fmt.Fprintln(b, "Actions: ←/→ select, Enter run, ↑/↓ profile")
	for i, action := range m.commandActions() {
		cursor := " "
		if i == m.commandCursor {
			cursor = ">"
		}
		fmt.Fprintf(b, "%s %s\n", cursor, action.label)
	}
}

func (m setupProfilesModel) writeCommandBar(b *strings.Builder) {
	fmt.Fprintln(b, "Actions: ←/→ select, Enter run, ↑/↓ profile")
	parts := make([]string, 0, len(m.commandActions()))
	for i, action := range m.commandActions() {
		label := action.label
		if i == m.commandCursor {
			label = "[" + label + "]"
		}
		parts = append(parts, label)
	}
	fmt.Fprintln(b, strings.Join(parts, "  "))
}

func (m setupProfilesModel) writeInputChrome(b *strings.Builder, label string, hint string) {
	fmt.Fprintln(b)
	input := m.inputField
	input.Width = max(1, m.viewWidth()-4)
	input.Placeholder = "…"
	input.SetValue(m.input)
	if m.viewWidth() < 40 {
		value := m.input
		if strings.TrimSpace(value) == "" {
			value = "…"
		}
		fmt.Fprintf(b, "%s: %s\n", label, value)
		return
	}
	fmt.Fprintln(b, tui.RenderTextInputChrome(tui.TextInputChrome{
		Width: m.viewWidth(),
		Label: label,
		Hint:  hint + " · Enter save · Esc back",
		Value: input.View(),
		Skin:  tui.DefaultHermesSkin(),
	}))
}

func (m setupProfilesModel) openWorkspaceEditor() setupProfilesModel {
	m.mode = setupProfilesModeWorkspaces
	m = m.setInput("")
	m.workspaceDraft = append([]string(nil), m.currentProfile().Workspaces...)
	m.workspaceIndex = 0
	m.workspaceEditingIndex = -1
	return m
}

func (m setupProfilesModel) removeSelectedWorkspace() setupProfilesModel {
	if len(m.workspaceDraft) == 0 || m.workspaceIndex < 0 || m.workspaceIndex >= len(m.workspaceDraft) {
		return m
	}
	m.workspaceDraft = append(m.workspaceDraft[:m.workspaceIndex], m.workspaceDraft[m.workspaceIndex+1:]...)
	if m.workspaceIndex >= len(m.workspaceDraft) && m.workspaceIndex > 0 {
		m.workspaceIndex--
	}
	return m
}

func (m setupProfilesModel) setSelectedWorkspacePrimary() setupProfilesModel {
	if len(m.workspaceDraft) == 0 || m.workspaceIndex <= 0 || m.workspaceIndex >= len(m.workspaceDraft) {
		return m
	}
	selected := m.workspaceDraft[m.workspaceIndex]
	copy(m.workspaceDraft[1:m.workspaceIndex+1], m.workspaceDraft[0:m.workspaceIndex])
	m.workspaceDraft[0] = selected
	m.workspaceIndex = 0
	return m
}

func (m setupProfilesModel) initialWorkspaceBrowserPath() string {
	workspaces := m.workspaceDraft
	if len(workspaces) == 0 {
		workspaces = m.currentProfile().Workspaces
	}
	for _, workspace := range workspaces {
		workspace = strings.TrimSpace(workspace)
		if workspace == "" {
			continue
		}
		if info, err := os.Stat(workspace); err == nil && info.IsDir() {
			return workspace
		}
		if parent := filepath.Dir(workspace); parent != "." {
			if info, err := os.Stat(parent); err == nil && info.IsDir() {
				return parent
			}
		}
	}
	if wd, err := os.Getwd(); err == nil && wd != "" {
		return wd
	}
	return config.GormesHome()
}

func (m setupProfilesModel) openWorkspaceBrowserPath(path string) (setupProfilesModel, tea.Cmd) {
	path = strings.TrimSpace(path)
	if path == "" {
		path = m.initialWorkspaceBrowserPath()
	}
	abs, err := filepath.Abs(path)
	if err == nil {
		path = abs
	}
	entries, err := os.ReadDir(path)
	if err != nil {
		m.err = err
		m.mode = setupProfilesModeWorkspaces
		return m, nil
	}
	m.workspaceBrowserPath = path
	m.workspaceBrowserEntries = m.workspaceBrowserEntries[:0]
	for _, entry := range entries {
		if entry.IsDir() {
			m.workspaceBrowserEntries = append(m.workspaceBrowserEntries, entry.Name())
		}
	}
	sort.Strings(m.workspaceBrowserEntries)
	m.workspaceBrowserIndex = 0
	m.workspacePicker = newSetupProfilesWorkspacePicker(path, min(10, m.viewHeight()-6))
	m.mode = setupProfilesModeWorkspaceBrowser
	m.err = nil
	return m, m.workspacePicker.Init()
}

func (m setupProfilesModel) selectWorkspaceBrowserEntry() setupProfilesModel {
	if len(m.workspaceBrowserEntries) == 0 {
		m.mode = setupProfilesModeWorkspaces
		return m
	}
	return m.selectWorkspaceBrowserPath(filepath.Join(m.workspaceBrowserPath, m.workspaceBrowserEntries[m.workspaceBrowserIndex]))
}

func (m setupProfilesModel) selectWorkspaceBrowserPath(selected string) setupProfilesModel {
	selected = strings.TrimSpace(selected)
	if selected == "" {
		m.mode = setupProfilesModeWorkspaces
		return m
	}
	m.workspaceDraft = append(m.workspaceDraft, selected)
	m.workspaceIndex = len(m.workspaceDraft) - 1
	m = m.setInput("")
	m.mode = setupProfilesModeWorkspaces
	return m
}

func (m setupProfilesModel) writeWorkspaceEditor(b *strings.Builder) {
	fmt.Fprintln(b, "\nWorkspace editor")
	fmt.Fprintln(b, "A workspace is a project folder this profile can use. The first workspace is primary/default.")
	fmt.Fprintln(b, "Labels use the folder basename until workspace label persistence is added to profile config.")
	fmt.Fprintln(b, "Actions: a Add path, f Browse folders, e Edit path, x Remove, p Set primary, Up/Down or j/k move, Enter Save, b/Esc Back.")
	if len(m.workspaceDraft) == 0 {
		fmt.Fprintln(b, "  (no workspaces yet)")
		return
	}
	for i, workspace := range m.workspaceDraft {
		cursor := " "
		if i == m.workspaceIndex {
			cursor = ">"
		}
		primary := ""
		if i == 0 {
			primary = " (primary)"
		}
		fmt.Fprintf(b, "%s %s — %s%s\n", cursor, setupWorkspaceLabel(workspace), workspace, primary)
	}
}

func setupWorkspaceLabel(path string) string {
	label := strings.TrimSpace(filepath.Base(path))
	if label == "" || label == "." || label == string(filepath.Separator) {
		return path
	}
	return label
}

func (m setupProfilesModel) writeChannelsList(b *strings.Builder) {
	fmt.Fprintln(b, "\nChannels")
	fmt.Fprintln(b, "Space toggle  j/k or Up/Down move  Enter done  q back")
	fmt.Fprintln(b, "Channels attach this profile agent to Gormes messaging channels. Navivox routes through the Gormes Navivox channel, not directly to Goncho.")
	m.channelList.SetSize(max(20, m.viewWidth()), max(3, min(8, m.viewHeight()-8)))
	view := strings.TrimRight(m.channelList.View(), "\n")
	if strings.TrimSpace(view) != "" {
		for _, line := range strings.Split(view, "\n") {
			trimmed := strings.TrimSpace(line)
			for _, channel := range setupProfilesChannelChoices {
				if strings.Contains(trimmed, channel) && m.channelDraft[channel] {
					line += "  ✓"
					break
				}
			}
			fmt.Fprintln(b, line)
		}
	}
}

func setupProfilesSelectedChannel(model list.Model) string {
	item, ok := model.SelectedItem().(setupChannelListItem)
	if !ok {
		return ""
	}
	return string(item)
}

func (m setupProfilesModel) writeWorkspaceBrowser(b *strings.Builder) {
	fmt.Fprintln(b, "\nWorkspace folder browser")
	fmt.Fprintf(b, "Current folder: %s\n", m.workspacePicker.CurrentDirectory)
	fmt.Fprintln(b, "Use ↑/↓ or j/k to choose, Enter open, Space to select, u parent, b/q or Esc back.")
	view := strings.TrimRight(m.workspacePicker.View(), "\n")
	if strings.TrimSpace(view) == "" || strings.Contains(view, "No Files Found") {
		if len(m.workspaceBrowserEntries) == 0 {
			fmt.Fprintln(b, "  (no child folders)")
			return
		}
		for i, entry := range m.workspaceBrowserEntries {
			cursor := " "
			if i == m.workspaceBrowserIndex {
				cursor = ">"
			}
			fmt.Fprintf(b, "%s %s/\n", cursor, entry)
		}
		return
	}
	fmt.Fprintln(b, view)
}

func appendSetupWorkspaceInput(current, path string) string {
	current = strings.TrimSpace(current)
	path = strings.TrimSpace(path)
	if path == "" {
		return current
	}
	if current == "" {
		return path
	}
	return current + "," + path
}

func (m setupProfilesModel) View() string {
	if m.state.ControlCenter {
		return m.controlCenterView()
	}
	var b strings.Builder
	fmt.Fprintln(&b, "Gormes profile setup")
	fmt.Fprintln(&b)
	fmt.Fprintln(&b, "Profiles")
	for i, profile := range m.state.Profiles {
		prefix := " "
		if i == m.selected {
			prefix = ">"
		}
		active := ""
		if profile.Active || profile.Name == m.state.Active {
			active = " (active)"
		}
		fmt.Fprintf(&b, "%s %s%s\n", prefix, profile.Name, active)
	}
	profile := m.currentProfile()
	fmt.Fprintln(&b)
	fmt.Fprintln(&b, "Selected profile")
	fmt.Fprintf(&b, "Name: %s\n", profile.Name)
	if profile.Root != "" {
		fmt.Fprintf(&b, "Root: %s\n", setupRedactedProfileRoot(profile.Root))
	}
	fmt.Fprintf(&b, "Workspaces: %s\n", setupProfilesWorkspaceListOrEmpty(profile.Workspaces))
	fmt.Fprintf(&b, "Channels: %s\n", setupProfilesListOrEmpty(profile.Channels))
	fmt.Fprintln(&b)
	fmt.Fprintln(&b, "Details")
	fmt.Fprintln(&b, "  add creates a new ~/.gormes/profiles/<name> home")
	fmt.Fprintln(&b, "  workspaces and channels save into the selected profile config.toml")
	fmt.Fprintln(&b, "  set active updates the sticky active_profile marker")
	fmt.Fprintln(&b)
	fmt.Fprintln(&b, "Commands")
	fmt.Fprintln(&b, "j/k or Up/Down move profile")
	m.writeCommandActions(&b)
	switch m.mode {
	case setupProfilesModeAddProfile:
		m.writeInputChrome(&b, "New profile", "profile-name | optional display name")
	case setupProfilesModeDisplayName:
		m.writeInputChrome(&b, "Display name", "friendly name shown in channel routing")
	case setupProfilesModeWorkspaces:
		m.writeWorkspaceEditor(&b)
	case setupProfilesModeWorkspacePath:
		m.writeInputChrome(&b, "Workspace path", "absolute folder path · f browse")
	case setupProfilesModeWorkspaceBrowser:
		m.writeWorkspaceBrowser(&b)
	case setupProfilesModeProviderCredential:
		m.writeInputChrome(&b, "Provider credential", "provider:credential_id")
	case setupProfilesModeChannelCredential:
		m.writeInputChrome(&b, "Channel credential", "channel:credential_id")
	case setupProfilesModeChannels:
		m.writeChannelsList(&b)
	}
	return setupProfilesWrapView(b.String(), m.viewWidth(), m.viewHeight())
}

func (m setupProfilesModel) controlCenterView() string {
	if m.state.MigrationAvailable {
		return m.controlCenterMigrationView()
	}
	if m.isTextInputMode() {
		return m.controlCenterInputView()
	}
	var b strings.Builder
	profile := m.currentProfile()
	fmt.Fprintln(&b, "Profile Control Center — Setup profiles")
	fmt.Fprintln(&b, "Profiles are agents: name them, attach workspaces, then connect channels.")
	fmt.Fprintln(&b)
	fmt.Fprintln(&b, "Profiles")
	for i, candidate := range m.state.Profiles {
		prefix := " "
		if i == m.selected {
			prefix = ">"
		}
		fmt.Fprintf(&b, "%s %s — %s\n", prefix, candidate.Name, setupProfilesDisplayName(candidate))
	}
	fmt.Fprintln(&b)
	fmt.Fprintf(&b, "Agent: %s — %s\n", profile.Name, setupProfilesDisplayName(profile))
	fmt.Fprintf(&b, "Display name: %s\n", setupProfilesDisplayName(profile))
	fmt.Fprintf(&b, "Workspaces: %s\n", setupProfilesWorkspaceSummary(profile.Workspaces))
	fmt.Fprintf(&b, "Channels: %s\n", setupProfilesChannelsSummary(profile.ChannelDetails, profile.Channels))
	fmt.Fprintf(&b, "Providers: %s\n", setupProfilesProvidersSummary(profile.Providers))
	fmt.Fprintf(&b, "Setup progress: %s\n", setupProfilesProgressBar(profile, max(12, m.viewWidth()-16)))
	fmt.Fprintln(&b)
	fmt.Fprintln(&b, "Tip: r rename, w edit workspace list, c connect Telegram/WhatsApp/Navivox. Drafts save only on Apply.")
	fmt.Fprintln(&b)
	m.writeCommandBar(&b)
	switch m.mode {
	case setupProfilesModeAddProfile:
		m.writeInputChrome(&b, "New profile", "profile-name | optional display name")
	case setupProfilesModeDisplayName:
		m.writeInputChrome(&b, "Display name", "friendly name shown in channel routing")
	case setupProfilesModeWorkspaces:
		m.writeWorkspaceEditor(&b)
	case setupProfilesModeWorkspacePath:
		m.writeInputChrome(&b, "Workspace path", "absolute folder path · f browse")
	case setupProfilesModeWorkspaceBrowser:
		m.writeWorkspaceBrowser(&b)
	case setupProfilesModeProviderCredential:
		m.writeInputChrome(&b, "Provider credential/model", "provider|credential_id|default_model|allowed_models")
	case setupProfilesModeChannelCredential:
		m.writeInputChrome(&b, "Channel credential/policy", "channel|credential_id|allowed_chats|allowed_users|require_mention|tool_progress")
	case setupProfilesModeChannels:
		m.writeChannelsList(&b)
	}
	return setupProfilesWrapView(b.String(), m.viewWidth(), m.viewHeight())
}

func (m setupProfilesModel) controlCenterInputView() string {
	var b strings.Builder
	profile := m.currentProfile()
	fmt.Fprintln(&b, "Profile Control Center — Setup profiles")
	fmt.Fprintf(&b, "Agent: %s — %s\n", profile.Name, setupProfilesDisplayName(profile))
	fmt.Fprintln(&b, "Draft input — Enter save, Esc back")
	switch m.mode {
	case setupProfilesModeAddProfile:
		m.writeInputChrome(&b, "New profile", "profile-name | optional display name")
	case setupProfilesModeDisplayName:
		m.writeInputChrome(&b, "Display name", "friendly name shown in channel routing")
	case setupProfilesModeWorkspacePath:
		m.writeInputChrome(&b, "Workspace path", "absolute folder path · f browse")
	case setupProfilesModeProviderCredential:
		m.writeInputChrome(&b, "Provider credential/model", "provider|credential_id|default_model|allowed_models")
	case setupProfilesModeChannelCredential:
		m.writeInputChrome(&b, "Channel credential/policy", "channel|credential_id|allowed_chats|allowed_users|require_mention|tool_progress")
	}
	return setupProfilesWrapView(b.String(), m.viewWidth(), m.viewHeight())
}

func (m setupProfilesModel) controlCenterMigrationView() string {
	var b strings.Builder
	fmt.Fprintln(&b, "Profile Control Center")
	fmt.Fprintln(&b)
	fmt.Fprintln(&b, "Migration preview")
	for _, line := range m.state.MigrationPreviewLines {
		fmt.Fprintf(&b, "  - %s\n", line)
	}
	if m.result.MigrateLegacyConfig {
		fmt.Fprintln(&b, "legacy migration staged for Apply")
	}
	fmt.Fprintln(&b)
	fmt.Fprintln(&b, "Commands")
	fmt.Fprintln(&b, "j/k or Up/Down move profile")
	m.writeCommandActions(&b)
	return setupProfilesWrapView(b.String(), m.viewWidth(), m.viewHeight())
}

func setupProfilesDisplayName(profile setupProfileView) string {
	name := strings.TrimSpace(profile.DisplayName)
	if name != "" {
		return name
	}
	switch strings.ToLower(strings.TrimSpace(profile.Name)) {
	case "default", "main", "gormes":
		return "Gormes"
	case "":
		return "(unnamed)"
	default:
		return strings.TrimSpace(profile.Name)
	}
}

func parseSetupProfilesNewProfileInput(value string) (string, string) {
	value = strings.TrimSpace(value)
	id, displayName, ok := strings.Cut(value, "|")
	id = strings.TrimSpace(id)
	if !ok {
		return id, id
	}
	displayName = strings.TrimSpace(displayName)
	if displayName == "" {
		displayName = id
	}
	return id, displayName
}

func parseSetupProfilesProviderAssignment(value string) (string, string, string, []string) {
	value = strings.TrimSpace(value)
	if strings.Contains(value, "|") {
		parts := strings.Split(value, "|")
		if len(parts) < 2 {
			return "", "", "", nil
		}
		providerID := strings.ToLower(strings.TrimSpace(parts[0]))
		credentialID := strings.TrimSpace(parts[1])
		defaultModel := ""
		if len(parts) > 2 {
			defaultModel = strings.TrimSpace(parts[2])
		}
		var allowedModels []string
		if len(parts) > 3 {
			allowedModels = parseSetupProfilesCSV(parts[3])
		}
		return providerID, credentialID, defaultModel, allowedModels
	}
	providerID, credentialID := parseSetupProfilesCredentialAssignment(value)
	return providerID, credentialID, "", nil
}

func parseSetupProfilesChannelAssignment(value string) (string, string, []string, []string, bool, string, bool) {
	value = strings.TrimSpace(value)
	if strings.Contains(value, "|") {
		parts := strings.Split(value, "|")
		if len(parts) < 2 {
			return "", "", nil, nil, false, "", false
		}
		channelID := strings.ToLower(strings.TrimSpace(parts[0]))
		credentialID := strings.TrimSpace(parts[1])
		var allowedChats []string
		if len(parts) > 2 {
			allowedChats = parseSetupProfilesCSV(parts[2])
		}
		var allowedUsers []string
		if len(parts) > 3 {
			allowedUsers = parseSetupProfilesCSV(parts[3])
		}
		requireMention := false
		if len(parts) > 4 && strings.TrimSpace(parts[4]) != "" {
			parsed, err := strconv.ParseBool(strings.TrimSpace(parts[4]))
			if err == nil {
				requireMention = parsed
			}
		}
		toolProgress := ""
		if len(parts) > 5 {
			toolProgress = strings.ToLower(strings.TrimSpace(parts[5]))
		}
		return channelID, credentialID, allowedChats, allowedUsers, requireMention, toolProgress, len(parts) > 2
	}
	channelID, credentialID := parseSetupProfilesCredentialAssignment(value)
	return channelID, credentialID, nil, nil, false, "", false
}

func parseSetupProfilesCredentialAssignment(value string) (string, string) {
	left, right, ok := strings.Cut(strings.TrimSpace(value), ":")
	if !ok {
		return "", ""
	}
	return strings.ToLower(strings.TrimSpace(left)), strings.TrimSpace(right)
}

func parseSetupProfilesCSV(value string) []string {
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	seen := map[string]bool{}
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" || seen[part] {
			continue
		}
		seen[part] = true
		out = append(out, part)
	}
	return out
}

func (m setupProfilesModel) currentProfile() setupProfileView {
	if len(m.state.Profiles) == 0 {
		return setupProfileView{Name: config.DefaultProfileID, Active: true}
	}
	if m.selected < 0 || m.selected >= len(m.state.Profiles) {
		return m.state.Profiles[0]
	}
	return m.state.Profiles[m.selected]
}

func (m setupProfilesModel) viewWidth() int {
	if m.width <= 0 {
		return 80
	}
	return max(1, m.width)
}

func (m setupProfilesModel) viewHeight() int {
	if m.height <= 0 {
		return 24
	}
	return max(1, m.height)
}

func setupProfilesWrapView(view string, width, height int) string {
	if width <= 0 {
		width = 80
	}
	var out []string
	for _, line := range strings.Split(view, "\n") {
		out = append(out, setupProfilesWrapLine(strings.TrimRight(line, " \t"), width)...)
	}
	out = setupProfilesClampHeight(out, width, height)
	for i, line := range out {
		line = setupProfilesTrimToWidth(line, width)
		if pad := width - lipgloss.Width(line); pad > 0 {
			line += strings.Repeat(" ", pad)
		}
		out[i] = line
	}
	vp := viewport.New(width, max(1, min(height, len(out))))
	vp.SetContent(strings.Join(out, "\n"))
	return vp.View()
}

func setupProfilesWrapLine(line string, width int) []string {
	if line == "" {
		return []string{""}
	}
	var lines []string
	for lipgloss.Width(line) > width {
		cut := setupProfilesWrapCut(line, width)
		lines = append(lines, strings.TrimRight(line[:cut], " \t"))
		line = strings.TrimLeft(line[cut:], " \t")
		if line == "" {
			break
		}
	}
	if line != "" {
		lines = append(lines, line)
	}
	if len(lines) <= 3 {
		return lines
	}
	return append(lines[:2], setupProfilesTrimToWidth("… value truncated; resize for full setup text", width))
}

func setupProfilesWrapCut(line string, width int) int {
	lastSpace := -1
	used := 0
	for i, r := range line {
		if r == ' ' || r == '\t' {
			lastSpace = i
		}
		rw := lipgloss.Width(string(r))
		if used+rw > width {
			if lastSpace > 0 {
				return lastSpace
			}
			if i > 0 {
				return i
			}
			return i + len(string(r))
		}
		used += rw
	}
	return len(line)
}

func setupProfilesClampHeight(lines []string, width, height int) []string {
	if height <= 0 || len(lines) <= height {
		return lines
	}
	if height <= 2 {
		return []string{setupProfilesTrimToWidth("terminal too small; resize", width)}
	}
	omitted := len(lines) - height + 1
	marker := setupProfilesTrimToWidth(fmt.Sprintf("… %d omitted; resize", omitted), width)
	tailCount := 2
	if height < 6 {
		tailCount = 1
	}
	headCount := height - tailCount - 1
	if headCount < 1 {
		headCount = 1
	}
	tailStart := len(lines) - tailCount
	var explicitTail []string
	for i := len(lines) - 1; i >= 0; i-- {
		if strings.TrimSpace(lines[i]) == "Channels" {
			if height <= 6 {
				explicitTail = setupProfilesCompactChannelTail(lines[i:], width)
			} else {
				tailStart = i
				tailCount = min(6, len(lines)-tailStart)
			}
			break
		}
		if strings.Contains(lines[i], "directories:") && i > 0 {
			tailStart = i - 1
			break
		}
		if strings.Contains(lines[i], "Workspace") || strings.Contains(lines[i], "New profile") || strings.Contains(lines[i], "Display name") || strings.Contains(lines[i], "credential") {
			tailStart = i
			break
		}
		if strings.TrimSpace(lines[i]) == "Commands" {
			tailStart = i
			tailCount = min(12, len(lines)-tailStart)
			break
		}
	}
	if len(explicitTail) > 0 {
		headCount = height - len(explicitTail) - 1
		if headCount < 1 {
			headCount = 1
		}
		out := append([]string(nil), lines[:headCount]...)
		out = append(out, marker)
		out = append(out, explicitTail...)
		return out
	}
	headCount = height - tailCount - 1
	if headCount < 1 {
		headCount = 1
	}
	if tailStart+tailCount > len(lines) {
		tailStart = max(0, len(lines)-tailCount)
	}
	out := append([]string(nil), lines[:headCount]...)
	out = append(out, marker)
	out = append(out, lines[tailStart:tailStart+tailCount]...)
	return out
}

func setupProfilesCompactChannelTail(lines []string, width int) []string {
	out := []string{"Channels"}
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, ">") || strings.HasPrefix(trimmed, "│") || setupProfilesLineMentionsChannel(trimmed) {
			out = append(out, line)
			break
		}
	}
	for _, line := range lines {
		if strings.Contains(line, "Space toggle") {
			out = append(out, setupProfilesTrimToWidth(line, width))
			break
		}
	}
	return out
}

func setupProfilesLineMentionsChannel(line string) bool {
	for _, channel := range setupProfilesChannelChoices {
		if strings.Contains(line, channel) {
			return true
		}
	}
	return false
}

func setupProfilesTrimToWidth(text string, width int) string {
	if width <= 0 || lipgloss.Width(text) <= width {
		return text
	}
	if width == 1 {
		return "…"
	}
	ellipsis := "…"
	limit := width - lipgloss.Width(ellipsis)
	used := 0
	for i, r := range text {
		rw := lipgloss.Width(string(r))
		if used+rw > limit {
			return strings.TrimRight(text[:i], " \t") + ellipsis
		}
		used += rw
	}
	return text
}

func setupProfilesListOrEmpty(values []string) string {
	if len(values) == 0 {
		return "(none)"
	}
	return strings.Join(values, ", ")
}

func setupProfilesWorkspaceSummary(values []string) string {
	clean := normalizeSetupProfilesTUIValues(values)
	if len(clean) == 0 {
		return "none — press w to add"
	}
	if len(clean) == 1 {
		return fmt.Sprintf("1 primary: %s", clean[0])
	}
	return fmt.Sprintf("%d total, primary: %s", len(clean), clean[0])
}

func setupProfilesWorkspaceListOrEmpty(values []string) string {
	if len(values) == 0 {
		return "(none)"
	}
	out := make([]string, 0, len(values))
	for i, value := range values {
		label := strings.TrimSpace(value)
		if label == "" {
			continue
		}
		if i == 0 {
			label += " (primary)"
		}
		out = append(out, label)
	}
	if len(out) == 0 {
		return "(none)"
	}
	return strings.Join(out, ", ")
}

func setupProfilesProgressBar(profile setupProfileView, width int) string {
	checks := 0
	complete := 0
	checks++
	if setupProfilesDisplayName(profile) != "" {
		complete++
	}
	checks++
	if len(normalizeSetupProfilesTUIValues(profile.Workspaces)) > 0 {
		complete++
	}
	checks++
	if len(profile.ChannelDetails) > 0 || len(normalizeSetupProfilesTUIValues(profile.Channels)) > 0 {
		complete++
	}
	checks++
	if len(profile.Providers) > 0 {
		complete++
	}
	if checks == 0 {
		return ""
	}
	bar := progress.New(progress.WithoutPercentage())
	bar.Width = max(4, width)
	return bar.ViewAs(float64(complete) / float64(checks))
}

func setupProfilesChannelsSummary(channelDetails []setupChannelView, fallback []string) string {
	if len(channelDetails) == 0 {
		channels := normalizeSetupProfilesTUIValues(fallback)
		if len(channels) == 0 {
			return "none — press c to connect"
		}
		return strings.Join(channels, ", ")
	}
	ready, degraded := 0, 0
	labels := make([]string, 0, len(channelDetails))
	for _, channel := range channelDetails {
		if channel.Status == "ready" {
			ready++
		} else {
			degraded++
		}
		label := channel.ID
		if channel.CredentialID != "" {
			label += " ✓"
		}
		labels = append(labels, label)
	}
	status := fmt.Sprintf("%d ready", ready)
	if degraded > 0 {
		status += fmt.Sprintf(", %d needs setup", degraded)
	}
	return fmt.Sprintf("%s (%s)", strings.Join(labels, ", "), status)
}

func setupProfilesChannelsListOrEmpty(channelDetails []setupChannelView, fallback []string) string {
	if len(channelDetails) == 0 {
		return setupProfilesListOrEmpty(fallback)
	}
	parts := make([]string, 0, len(channelDetails))
	for _, channel := range channelDetails {
		part := channel.ID
		if channel.CredentialID != "" {
			part += " credential=" + channel.CredentialID
		}
		part += fmt.Sprintf(" chats=%d users=%d", channel.AllowedChatCount, channel.AllowedUserCount)
		if channel.RequireMention {
			part += " require_mention=true"
		}
		if channel.ToolProgress != "" {
			part += " tool_progress=" + channel.ToolProgress
		}
		if channel.Status != "" {
			part += " status=" + channel.Status
		}
		parts = append(parts, part)
	}
	return strings.Join(parts, "; ")
}

func setupProfilesProvidersSummary(providers []setupProviderView) string {
	if len(providers) == 0 {
		return "none — press p to assign"
	}
	ready, degraded := 0, 0
	labels := make([]string, 0, len(providers))
	for _, provider := range providers {
		if provider.Status == "ready" {
			ready++
		} else {
			degraded++
		}
		label := provider.ID
		if provider.DefaultModel != "" {
			label += " " + provider.DefaultModel
		}
		labels = append(labels, label)
	}
	status := fmt.Sprintf("%d ready", ready)
	if degraded > 0 {
		status += fmt.Sprintf(", %d needs setup", degraded)
	}
	return fmt.Sprintf("%s (%s)", strings.Join(labels, ", "), status)
}

func setupProfilesProvidersListOrEmpty(providers []setupProviderView) string {
	if len(providers) == 0 {
		return "(none)"
	}
	parts := make([]string, 0, len(providers))
	for _, provider := range providers {
		part := provider.ID
		if provider.CredentialID != "" {
			part += " credential=" + provider.CredentialID
		}
		if provider.DefaultModel != "" {
			part += " model=" + provider.DefaultModel
		}
		if provider.Status != "" {
			part += " status=" + provider.Status
		}
		parts = append(parts, part)
	}
	return strings.Join(parts, "; ")
}
