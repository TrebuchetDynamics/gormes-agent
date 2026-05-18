package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/pelletier/go-toml/v2"
	"github.com/spf13/cobra"
)

type setupProfileView struct {
	Name       string
	Root       string
	Active     bool
	Workspaces []string
	Channels   []string
}

type setupProfilesTUIState struct {
	Active   string
	Profiles []setupProfileView
}

type setupProfilesTUIResult struct {
	Cancelled     bool
	CreateName    string
	Selected      string
	SetActive     bool
	WorkspacesSet bool
	Workspaces    []string
	ChannelsSet   bool
	Channels      []string
}

type setupProfilesMode string

const (
	setupProfilesModeBrowse     setupProfilesMode = "browse"
	setupProfilesModeAddProfile setupProfilesMode = "add_profile"
	setupProfilesModeWorkspaces setupProfilesMode = "workspaces"
	setupProfilesModeChannels   setupProfilesMode = "channels"
)

type setupProfilesModel struct {
	state        setupProfilesTUIState
	selected     int
	mode         setupProfilesMode
	input        string
	channelDraft map[string]bool
	channelIndex int
	result       setupProfilesTUIResult
	err          error
}

var runSetupProfilesTUI = runSetupProfilesTUIDefault

var setupProfilesChannelChoices = []string{"telegram", "whatsapp", "discord", "slack"}

func maybeRunSetupProfilesTUI(cmd *cobra.Command, pseams profileCommandSeams, known []string, active string) (bool, error) {
	stdin, ok := cmd.InOrStdin().(*os.File)
	if !ok || !setupInputIsTerminal(stdin) {
		return false, nil
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
		channelDraft: make(map[string]bool),
	}
}

func (m setupProfilesModel) Init() tea.Cmd {
	return nil
}

func (m setupProfilesModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
	case 'n':
		m.mode = setupProfilesModeAddProfile
		m.input = ""
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
	case 'a':
		profile := m.currentProfile()
		for i := range m.state.Profiles {
			m.state.Profiles[i].Active = i == m.selected
		}
		m.state.Active = profile.Name
		m.result.Selected = profile.Name
		m.result.SetActive = true
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
	case setupProfilesModeAddProfile:
		if value != "" {
			view := setupProfileView{Name: value, Root: value}
			m.state.Profiles = append(m.state.Profiles, view)
			m.selected = len(m.state.Profiles) - 1
			m.result.CreateName = value
			m.result.Selected = value
		}
	case setupProfilesModeWorkspaces:
		m.result.Selected = m.currentProfile().Name
		m.result.WorkspacesSet = true
		m.result.Workspaces = parseSetupWorkspaceList(value)
		m.state.Profiles[m.selected].Workspaces = append([]string(nil), m.result.Workspaces...)
	case setupProfilesModeChannels:
		m.result.Selected = m.currentProfile().Name
		m.result.ChannelsSet = true
		for _, channel := range setupProfilesChannelChoices {
			if m.channelDraft[channel] {
				m.result.Channels = append(m.result.Channels, channel)
			}
		}
		m.state.Profiles[m.selected].Channels = append([]string(nil), m.result.Channels...)
	}
	m.mode = setupProfilesModeBrowse
	m.input = ""
	return m, nil
}

func (m setupProfilesModel) View() string {
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
	fmt.Fprintln(&b, "n add profile")
	fmt.Fprintln(&b, "w edit workspaces")
	fmt.Fprintln(&b, "c edit channels")
	fmt.Fprintln(&b, "a set active")
	fmt.Fprintln(&b, "s save")
	fmt.Fprintln(&b, "q cancel")
	switch m.mode {
	case setupProfilesModeAddProfile:
		fmt.Fprintf(&b, "\nNew profile: %s", m.input)
	case setupProfilesModeWorkspaces:
		fmt.Fprintf(&b, "\nWorkspace directories: %s", m.input)
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
		fmt.Fprintln(&b, "Space toggle  Up/Down move  Enter done")
	}
	return b.String()
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

func setupProfilesListOrEmpty(values []string) string {
	if len(values) == 0 {
		return "(none)"
	}
	return strings.Join(values, ", ")
}
