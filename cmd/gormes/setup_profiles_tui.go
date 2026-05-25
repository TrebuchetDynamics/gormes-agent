package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	profilemodule "github.com/TrebuchetDynamics/gormes-agent/internal/cli/gormescli/modules/profiles"
	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	gatewaymodule "github.com/TrebuchetDynamics/gormes-agent/internal/gateway"
	providermodule "github.com/TrebuchetDynamics/gormes-agent/internal/provider"
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
	setupProfilesModeChannels           setupProfilesMode = "channels"
	setupProfilesModeProviderCredential setupProfilesMode = "provider_credential"
	setupProfilesModeChannelCredential  setupProfilesMode = "channel_credential"
)

type setupProfilesModel struct {
	state        setupProfilesTUIState
	selected     int
	mode         setupProfilesMode
	input        string
	width        int
	height       int
	channelDraft map[string]bool
	channelIndex int
	result       setupProfilesTUIResult
	err          error
}

var runSetupProfilesTUI = runSetupProfilesTUIDefault
var writeSetupProfilesControlCenterConfig = config.WriteProfileConfigV2
var applySetupProfilesControlCenterMigration = config.ApplyProfileConfigV2Migration

var setupProfilesChannelChoices = []string{"telegram", "whatsapp", "discord", "slack"}

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
		fmt.Fprintln(out, "Profile Control Center draft discarded; no config changes written.")
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
			fmt.Fprintf(out, "Skipping unknown channel %q (known: telegram, whatsapp, discord, slack).\n", u)
		}
		if err := draft.SetProfileChannels(selected, validChannels); err != nil {
			return err
		}
	}
	changes := draft.Preview()
	fmt.Fprintln(out, "Profile Control Center draft:")
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
	if err := writeSetupProfilesControlCenterConfig(config.ConfigPath(), applied); err != nil {
		return fmt.Errorf("apply profile control center draft: %w", err)
	}
	fmt.Fprintf(out, "Applied %d profile control center change(s) to %s.\n", len(changes), config.ConfigPath())
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
	fmt.Fprintln(out, "Profile Control Center migration:")
	for _, line := range result.Plan.PreviewLines {
		fmt.Fprintf(out, "  - %s\n", line)
	}
	if result.NoOp {
		fmt.Fprintf(out, "No legacy profile config migration needed for %s.\n", result.Path)
		return nil
	}
	if result.Wrote {
		fmt.Fprintf(out, "Applied legacy profile config migration to %s.\n", result.Path)
		if result.BackupPath != "" {
			fmt.Fprintf(out, "Backup: %s\n", result.BackupPath)
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
		selected = "default"
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
	if result.WorkspacesSet {
		workspaces := normalizeSetupProfilesTUIValues(result.Workspaces)
		if err := config.WriteTOMLValue(profileConfigPath, "agents.defaults.workspaces", strings.Join(workspaces, ",")); err != nil {
			return fmt.Errorf("persist workspaces for profile %q: %w", selected, err)
		}
		fmt.Fprintf(out, "Set %d workspace(s) for profile %q in %s.\n", len(workspaces), selected, profileConfigPath)
	}
	if result.ChannelsSet {
		validChannels, unknownChannels := parseSetupChannelList(strings.Join(result.Channels, ","))
		for _, u := range unknownChannels {
			fmt.Fprintf(out, "Skipping unknown channel %q (known: telegram, whatsapp, discord, slack).\n", u)
		}
		if len(validChannels) == 0 {
			fmt.Fprintf(out, "No valid channels for profile %q.\n", selected)
			return nil
		}
		if err := config.WriteTOMLValue(profileConfigPath, "agents.defaults.channels", strings.Join(validChannels, ",")); err != nil {
			return fmt.Errorf("persist channels for profile %q: %w", selected, err)
		}
		fmt.Fprintf(out, "Set %d channel(s) for profile %q in %s.\n", len(validChannels), selected, profileConfigPath)
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

func newSetupProfilesModel(state setupProfilesTUIState) setupProfilesModel {
	if strings.TrimSpace(state.Active) == "" {
		state.Active = "default"
	}
	if len(state.Profiles) == 0 {
		state.Profiles = []setupProfileView{{Name: "default", Active: true}}
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
		width:        80,
		height:       24,
		channelDraft: make(map[string]bool),
	}
}

func (m setupProfilesModel) Init() tea.Cmd {
	return nil
}

func (m setupProfilesModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if size, ok := msg.(tea.WindowSizeMsg); ok {
		m.width = size.Width
		m.height = size.Height
		return m, nil
	}
	key, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}
	switch key.Type {
	case tea.KeyCtrlC, tea.KeyEsc:
		m.result.Cancelled = true
		return m, tea.Quit
	case tea.KeyEnter:
		return m.handleEnter()
	case tea.KeyBackspace:
		if m.mode != setupProfilesModeBrowse && len(m.input) > 0 {
			m.input = m.input[:len(m.input)-1]
		}
		return m, nil
	case tea.KeySpace:
		if m.mode == setupProfilesModeChannels {
			channel := setupProfilesChannelChoices[m.channelIndex]
			m.channelDraft[channel] = !m.channelDraft[channel]
		}
		return m, nil
	case tea.KeyUp:
		if m.mode == setupProfilesModeChannels && m.channelIndex > 0 {
			m.channelIndex--
		} else if m.mode == setupProfilesModeBrowse && m.selected > 0 {
			m.selected--
		}
		return m, nil
	case tea.KeyDown:
		if m.mode == setupProfilesModeChannels && m.channelIndex < len(setupProfilesChannelChoices)-1 {
			m.channelIndex++
		} else if m.mode == setupProfilesModeBrowse && m.selected < len(m.state.Profiles)-1 {
			m.selected++
		}
		return m, nil
	case tea.KeyRunes:
		return m.handleRunes(key.Runes)
	}
	return m, nil
}

func (m setupProfilesModel) handleRunes(runes []rune) (tea.Model, tea.Cmd) {
	if m.mode == setupProfilesModeChannels {
		if len(runes) != 1 {
			return m, nil
		}
		switch runes[0] {
		case 'j':
			if m.channelIndex < len(setupProfilesChannelChoices)-1 {
				m.channelIndex++
			}
		case 'k':
			if m.channelIndex > 0 {
				m.channelIndex--
			}
		case 'q':
			m.mode = setupProfilesModeBrowse
		}
		return m, nil
	}
	if m.mode != setupProfilesModeBrowse {
		m.input += string(runes)
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
		m.input = ""
	case 'r':
		if m.state.ControlCenter {
			m.mode = setupProfilesModeDisplayName
			m.input = strings.TrimSpace(m.currentProfile().DisplayName)
		}
	case 'w':
		m.mode = setupProfilesModeWorkspaces
		m.input = strings.Join(m.currentProfile().Workspaces, ",")
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
			m.input = ""
		}
	case 't':
		if m.state.ControlCenter {
			m.mode = setupProfilesModeChannelCredential
			m.input = ""
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
		if m.result.Selected == "" {
			m.result.Selected = m.currentProfile().Name
		}
		return m, tea.Quit
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
		m.result.Workspaces = parseSetupWorkspaceList(value)
		m.state.Profiles[m.selected].Workspaces = append([]string(nil), m.result.Workspaces...)
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
	m.input = ""
	return m, nil
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
		fmt.Fprintf(&b, "Root: %s\n", profile.Root)
	}
	fmt.Fprintf(&b, "Workspaces: %s\n", setupProfilesListOrEmpty(profile.Workspaces))
	fmt.Fprintf(&b, "Channels: %s\n", setupProfilesListOrEmpty(profile.Channels))
	fmt.Fprintln(&b)
	fmt.Fprintln(&b, "Details")
	fmt.Fprintln(&b, "  add creates a new ~/.gormes/profiles/<name> home")
	fmt.Fprintln(&b, "  workspaces and channels save into the selected profile config.toml")
	fmt.Fprintln(&b, "  set active updates the sticky active_profile marker")
	fmt.Fprintln(&b)
	fmt.Fprintln(&b, "Commands")
	fmt.Fprintln(&b, "j/k or Up/Down move profile")
	fmt.Fprintln(&b, "n add profile")
	fmt.Fprintln(&b, "w edit workspaces")
	fmt.Fprintln(&b, "c edit channels")
	fmt.Fprintln(&b, "a set active")
	fmt.Fprintln(&b, "s save")
	fmt.Fprintln(&b, "Enter save selected profile")
	fmt.Fprintln(&b, "q cancel")
	switch m.mode {
	case setupProfilesModeAddProfile:
		fmt.Fprintf(&b, "\nNew profile: %s", m.input)
	case setupProfilesModeDisplayName:
		fmt.Fprintf(&b, "\nDisplay name: %s", m.input)
	case setupProfilesModeWorkspaces:
		fmt.Fprintf(&b, "\nWorkspace directories: %s", m.input)
	case setupProfilesModeProviderCredential:
		fmt.Fprintf(&b, "\nProvider credential provider:credential_id: %s", m.input)
	case setupProfilesModeChannelCredential:
		fmt.Fprintf(&b, "\nChannel credential channel:credential_id: %s", m.input)
	case setupProfilesModeChannels:
		fmt.Fprintln(&b, "\nChannels")
		for i, channel := range setupProfilesChannelChoices {
			cursor := " "
			if i == m.channelIndex {
				cursor = ">"
			}
			marker := "[ ]"
			if m.channelDraft[channel] {
				marker = "[x]"
			}
			fmt.Fprintf(&b, "%s %s %s\n", cursor, marker, channel)
		}
		fmt.Fprintln(&b, "Space toggle  j/k or Up/Down move  Enter done  q back")
	}
	return setupProfilesWrapView(b.String(), m.viewWidth(), m.viewHeight())
}

func (m setupProfilesModel) controlCenterView() string {
	if m.state.MigrationAvailable {
		return m.controlCenterMigrationView()
	}
	var b strings.Builder
	fmt.Fprintln(&b, "Profile Control Center")
	fmt.Fprintln(&b)
	fmt.Fprintln(&b, "Profile services")
	for i, profile := range m.state.Profiles {
		prefix := " "
		if i == m.selected {
			prefix = ">"
		}
		fmt.Fprintf(&b, "%s %s — %s\n", prefix, profile.Name, setupProfilesDisplayName(profile))
	}
	profile := m.currentProfile()
	fmt.Fprintln(&b)
	fmt.Fprintln(&b, "Selected profile")
	fmt.Fprintf(&b, "ID: %s\n", profile.Name)
	fmt.Fprintf(&b, "Display name: %s\n", setupProfilesDisplayName(profile))
	fmt.Fprintf(&b, "Workspaces: %s\n", setupProfilesListOrEmpty(profile.Workspaces))
	fmt.Fprintf(&b, "Channels: %s\n", setupProfilesChannelsListOrEmpty(profile.ChannelDetails, profile.Channels))
	fmt.Fprintf(&b, "Providers: %s\n", setupProfilesProvidersListOrEmpty(profile.Providers))
	for _, provider := range profile.Providers {
		if len(provider.Models) > 0 {
			fmt.Fprintf(&b, "provider models %s: %s\n", provider.ID, strings.Join(provider.Models, ", "))
		}
	}
	fmt.Fprintln(&b)
	fmt.Fprintln(&b, "Details")
	fmt.Fprintln(&b, "  draft changes stay in memory until Apply")
	fmt.Fprintln(&b, "  Apply writes one root config.toml transaction")
	fmt.Fprintln(&b, "  Discard writes nothing")
	if m.state.MigrationAvailable {
		fmt.Fprintln(&b)
		fmt.Fprintln(&b, "Migration preview")
		for _, line := range m.state.MigrationPreviewLines {
			fmt.Fprintf(&b, "  - %s\n", line)
		}
		if m.result.MigrateLegacyConfig {
			fmt.Fprintln(&b, "legacy migration staged for Apply")
		}
	}
	fmt.Fprintln(&b)
	fmt.Fprintln(&b, "Commands")
	fmt.Fprintln(&b, "j/k or Up/Down move profile")
	fmt.Fprintln(&b, "n add profile as id|display name")
	fmt.Fprintln(&b, "r rename display name")
	fmt.Fprintln(&b, "w edit workspaces")
	fmt.Fprintln(&b, "c edit channels")
	fmt.Fprintln(&b, "p assign provider credential/model")
	fmt.Fprintln(&b, "t assign channel credential/policy")
	if m.state.MigrationAvailable {
		fmt.Fprintln(&b, "m stage legacy migration")
	}
	fmt.Fprintln(&b, "s apply draft")
	fmt.Fprintln(&b, "d discard draft")
	fmt.Fprintln(&b, "q cancel")
	switch m.mode {
	case setupProfilesModeAddProfile:
		fmt.Fprintf(&b, "\nNew profile id|display name: %s", m.input)
	case setupProfilesModeDisplayName:
		fmt.Fprintf(&b, "\nDisplay name: %s", m.input)
	case setupProfilesModeWorkspaces:
		fmt.Fprintf(&b, "\nWorkspace directories: %s", m.input)
	case setupProfilesModeProviderCredential:
		fmt.Fprintf(&b, "\nProvider credential/model provider|credential_id|default_model|allowed_models: %s", m.input)
	case setupProfilesModeChannelCredential:
		fmt.Fprintf(&b, "\nChannel credential/policy channel|credential_id|allowed_chats|allowed_users|require_mention|tool_progress: %s", m.input)
	case setupProfilesModeChannels:
		fmt.Fprintln(&b, "\nChannels")
		for i, channel := range setupProfilesChannelChoices {
			cursor := " "
			if i == m.channelIndex {
				cursor = ">"
			}
			marker := "[ ]"
			if m.channelDraft[channel] {
				marker = "[x]"
			}
			fmt.Fprintf(&b, "%s %s %s\n", cursor, marker, channel)
		}
		fmt.Fprintln(&b, "Space toggle  j/k or Up/Down move  Enter done  q back")
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
	fmt.Fprintln(&b, "n add profile as id|display name")
	fmt.Fprintln(&b, "r rename display name")
	fmt.Fprintln(&b, "w edit workspaces")
	fmt.Fprintln(&b, "c edit channels")
	fmt.Fprintln(&b, "p assign provider credential/model")
	fmt.Fprintln(&b, "t assign channel credential/policy")
	fmt.Fprintln(&b, "m stage legacy migration")
	fmt.Fprintln(&b, "s apply draft")
	fmt.Fprintln(&b, "d discard draft")
	fmt.Fprintln(&b, "q cancel")
	return setupProfilesWrapView(b.String(), m.viewWidth(), m.viewHeight())
}

func setupProfilesDisplayName(profile setupProfileView) string {
	name := strings.TrimSpace(profile.DisplayName)
	if name != "" {
		return name
	}
	if strings.TrimSpace(profile.Name) == "" {
		return "(unnamed)"
	}
	return "(unnamed)"
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
		return setupProfileView{Name: "default", Active: true}
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
	return strings.Join(out, "\n")
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
		if strings.Contains(lines[i], "Workspace") || strings.Contains(lines[i], "New profile:") {
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
		if strings.HasPrefix(strings.TrimSpace(line), ">") {
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
