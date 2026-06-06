package gormescli

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/exp/teatest"
)

func TestSetupProfilesTUIRendersManagerSurface(t *testing.T) {
	m := newSetupProfilesModel(setupProfilesTUIState{
		Active: "main",
		Profiles: []setupProfileView{
			{
				Name:       "main",
				Root:       "/home/operator/.gormes",
				Active:     true,
				Workspaces: []string{"/srv/alpha", "/srv/beta"},
				Channels:   []string{"telegram"},
			},
		},
	})

	view := m.View()
	if strings.Contains(view, "/home/operator") {
		t.Fatalf("profile TUI view leaked raw operator profile root:\n%s", view)
	}
	if !strings.Contains(view, "Root: .../.gormes") {
		t.Fatalf("profile TUI view must show a redacted profile root:\n%s", view)
	}
	for _, want := range []string{
		"Gormes profile setup",
		"Profiles",
		"Selected profile",
		"main",
		"/srv/alpha (primary)",
		"telegram",
		"n add profile",
		"w edit workspaces",
		"c edit channels",
		"a set active",
		"s save",
	} {
		if !strings.Contains(view, want) {
			t.Fatalf("profile TUI view missing %q:\n%s", want, view)
		}
	}
}

func TestSetupProfilesControlCenterShowsDefaultDisplayNameAndHelp(t *testing.T) {
	m := newSetupProfilesModel(setupProfilesTUIState{
		ControlCenter: true,
		Profiles: []setupProfileView{{
			Name:       "main",
			Workspaces: []string{"/srv/gormes", "/srv/salma"},
			Channels:   []string{"telegram", "whatsapp"},
		}},
	})
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 50})
	m = updated.(setupProfilesModel)

	view := m.View()
	for _, want := range []string{
		"Profile Control Center",
		"main — Gormes",
		"Display name: Gormes",
		"Workspaces: 2 total, primary: /srv/gormes",
		"Setup progress:",
		"Profiles are agents",
		"Display name",
		"Telegram/WhatsApp/Navivox",
	} {
		if !strings.Contains(view, want) {
			t.Fatalf("control center view missing %q:\n%s", want, view)
		}
	}
	if strings.Contains(view, "Profile services") {
		t.Fatalf("control center view used confusing Profile services wording:\n%s", view)
	}
}

func TestSetupProfilesTextInputUsesBubblesEditing(t *testing.T) {
	m := newSetupProfilesModel(setupProfilesTUIState{Profiles: []setupProfileView{{Name: "main"}}})
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	m = updated.(setupProfilesModel)
	for _, r := range "profile" {
		updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		m = updated.(setupProfilesModel)
	}
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeySpace})
	m = updated.(setupProfilesModel)
	for _, r := range "name" {
		updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		m = updated.(setupProfilesModel)
	}

	if m.input != "profile name" || m.inputField.Value() != "profile name" {
		t.Fatalf("setup textinput value = input %q field %q, want profile name", m.input, m.inputField.Value())
	}
}

func TestSetupProfilesControlCenterInputUsesSharedChrome(t *testing.T) {
	m := newSetupProfilesModel(setupProfilesTUIState{
		ControlCenter: true,
		Profiles: []setupProfileView{{
			Name:        "main",
			DisplayName: "Gormes",
		}},
	})
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 90, Height: 28})
	m = updated.(setupProfilesModel)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	m = updated.(setupProfilesModel)
	m.input = "Research Agent"

	view := m.View()
	for _, want := range []string{"Display name", "friendly name shown in channel routing", "Enter save", "Esc back", "Research Agent"} {
		if !strings.Contains(view, want) {
			t.Fatalf("control center input view missing %q:\n%s", want, view)
		}
	}
	for _, line := range strings.Split(view, "\n") {
		if got := lipgloss.Width(line); got > 90 {
			t.Fatalf("control center input line width %d exceeds 90:\n%q\n\n%s", got, line, view)
		}
	}
}

func TestSetupProfilesControlCenterInputModeIsCompact(t *testing.T) {
	m := newSetupProfilesModel(setupProfilesTUIState{
		ControlCenter: true,
		Profiles: []setupProfileView{{
			Name:        "main",
			DisplayName: "Gormes",
			Workspaces:  []string{"/srv/gormes"},
			Channels:    []string{"telegram"},
		}},
	})
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 56, Height: 18})
	m = updated.(setupProfilesModel)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	m = updated.(setupProfilesModel)
	m = m.setInput("mineru")

	view := m.View()
	if strings.Contains(view, "Tip: r rename") || strings.Contains(view, "[choose]") || strings.Contains(view, "Workspaces: 1 primary") {
		t.Fatalf("input mode should not render browse command/details chrome:\n%s", view)
	}
	for _, want := range []string{"Profile Control Center", "Draft input", "New profile", "mineru"} {
		if !strings.Contains(view, want) {
			t.Fatalf("input mode missing %q:\n%s", want, view)
		}
	}
	for _, line := range strings.Split(view, "\n") {
		if got := lipgloss.Width(line); got > 56 {
			t.Fatalf("input mode line width %d exceeds 56:\n%q\n\n%s", got, line, view)
		}
	}
}

func TestSetupProfilesControlCenterMainScreenFitsWithoutOmission(t *testing.T) {
	m := newSetupProfilesModel(setupProfilesTUIState{
		ControlCenter: true,
		Profiles: []setupProfileView{{
			Name:        "main",
			DisplayName: "Gormes",
			Channels:    []string{"telegram"},
			Workspaces:  []string{"/srv/gormes"},
		}},
	})
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = updated.(setupProfilesModel)

	view := m.View()
	if strings.Contains(view, "omitted; resize") || strings.Contains(view, "highlighted action") {
		t.Fatalf("control center main screen is still cramped/truncated:\n%s", view)
	}
	for _, line := range strings.Split(view, "\n") {
		if got := lipgloss.Width(line); got > 80 {
			t.Fatalf("control center line width %d exceeds 80:\n%q\n\n%s", got, line, view)
		}
	}
	for _, want := range []string{
		"Profile Control Center",
		"Profiles are agents",
		"Agent: main — Gormes",
		"Workspaces: 1 primary: /srv/gormes",
		"Actions: ←/→ select, Enter run, ↑/↓ profile",
		"[choose]",
	} {
		if !strings.Contains(view, want) {
			t.Fatalf("control center compact view missing %q:\n%s", want, view)
		}
	}
}

func TestSetupProfilesTUICommandActionsAreArrowSelectable(t *testing.T) {
	m := newSetupProfilesModel(setupProfilesTUIState{
		ControlCenter: true,
		Active:        "main",
		Profiles: []setupProfileView{{
			Name:        "main",
			DisplayName: "Gormes",
			Workspaces:  []string{"/srv/gormes"},
		}},
	})

	view := m.View()
	for _, want := range []string{
		"[choose]",
		"Actions: ←/→ select, Enter run, ↑/↓ profile",
		"r rename",
	} {
		if !strings.Contains(view, want) {
			t.Fatalf("profile TUI command view missing %q:\n%s", want, view)
		}
	}

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRight})
	m = updated.(setupProfilesModel)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRight})
	m = updated.(setupProfilesModel)
	view = m.View()
	if !strings.Contains(view, "[r rename]") {
		t.Fatalf("Right arrow did not highlight display-name action:\n%s", view)
	}

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(setupProfilesModel)
	if cmd != nil {
		t.Fatalf("Enter on display-name action returned command %T, want in-TUI editor", cmd)
	}
	if m.mode != setupProfilesModeDisplayName {
		t.Fatalf("Enter on highlighted display-name action mode = %q, want display_name", m.mode)
	}
	if m.input != "Gormes" {
		t.Fatalf("display-name editor input = %q, want current display name", m.input)
	}
}

func TestSetupProfilesChannelEditorIncludesNavivoxRoutingHelp(t *testing.T) {
	m := newSetupProfilesModel(setupProfilesTUIState{
		ControlCenter: true,
		Profiles: []setupProfileView{{
			Name:     "main",
			Channels: []string{"telegram"},
		}},
	})
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 140, Height: 80})
	m = updated.(setupProfilesModel)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}})
	m = updated.(setupProfilesModel)

	view := m.View()
	for _, want := range []string{
		"Channels attach this profile agent to Gormes messaging channels",
		"Navivox routes through the Gormes Navivox channel, not directly to Goncho",
		"navivox",
	} {
		if !strings.Contains(view, want) {
			t.Fatalf("channel editor missing %q:\n%s", want, view)
		}
	}

	for setupProfilesChannelChoices[m.channelIndex] != "navivox" {
		updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
		m = updated.(setupProfilesModel)
	}
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeySpace})
	m = updated.(setupProfilesModel)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(setupProfilesModel)
	if !m.result.ChannelsSet || !containsString(m.result.Channels, "telegram") || !containsString(m.result.Channels, "navivox") {
		t.Fatalf("channels result = %+v, want telegram and navivox", m.result)
	}
}

func TestSetupProfilesWorkspaceEditorManagesListAndPrimary(t *testing.T) {
	m := newSetupProfilesModel(setupProfilesTUIState{
		ControlCenter: true,
		Profiles: []setupProfileView{{
			Name:       "main",
			Workspaces: []string{"/srv/guides", "/srv/salma"},
		}},
	})
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 140, Height: 80})
	m = updated.(setupProfilesModel)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'w'}})
	m = updated.(setupProfilesModel)
	view := m.View()
	for _, want := range []string{"guides — /srv/guides (primary)", "salma — /srv/salma", "x Remove", "p Set primary", "a Add path"} {
		if !strings.Contains(view, want) {
			t.Fatalf("workspace editor missing %q:\n%s", want, view)
		}
	}

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = updated.(setupProfilesModel)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'p'}})
	m = updated.(setupProfilesModel)
	if got := m.workspaceDraft; len(got) != 2 || got[0] != "/srv/salma" || got[1] != "/srv/guides" {
		t.Fatalf("after set primary workspaceDraft = %#v", got)
	}
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = updated.(setupProfilesModel)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	m = updated.(setupProfilesModel)
	if got := m.workspaceDraft; len(got) != 1 || got[0] != "/srv/salma" {
		t.Fatalf("after remove workspaceDraft = %#v", got)
	}
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	m = updated.(setupProfilesModel)
	if m.mode != setupProfilesModeWorkspacePath {
		t.Fatalf("add path mode = %q, want workspace_path", m.mode)
	}
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("/srv/monorepo")})
	m = updated.(setupProfilesModel)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(setupProfilesModel)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(setupProfilesModel)
	if got, want := m.result.Workspaces, []string{"/srv/salma", "/srv/monorepo"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("saved workspaces = %#v, want %#v", got, want)
	}
}

func TestSetupProfilesWorkspaceEditorCanBrowseAndSelectFolder(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "guides"), 0o755); err != nil {
		t.Fatalf("mkdir guides: %v", err)
	}
	if err := os.Mkdir(filepath.Join(root, "salma"), 0o755); err != nil {
		t.Fatalf("mkdir salma: %v", err)
	}
	m := newSetupProfilesModel(setupProfilesTUIState{
		ControlCenter: true,
		Profiles: []setupProfileView{{
			Name:       "main",
			Workspaces: []string{root},
		}},
	})
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 140, Height: 80})
	m = updated.(setupProfilesModel)

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'w'}})
	m = updated.(setupProfilesModel)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'f'}})
	m = updated.(setupProfilesModel)
	if m.mode != setupProfilesModeWorkspaceBrowser {
		t.Fatalf("f in workspace editor mode = %q, want workspace_browser", m.mode)
	}
	view := m.View()
	for _, want := range []string{"Workspace folder browser", "Space to select", "guides/", "salma/"} {
		if !strings.Contains(view, want) {
			t.Fatalf("workspace browser missing %q:\n%s", want, view)
		}
	}

	if len(m.workspaceBrowserEntries) == 0 || m.workspaceBrowserEntries[m.workspaceBrowserIndex] != "guides" {
		t.Fatalf("workspace browser entries = %#v index=%d, want guides first", m.workspaceBrowserEntries, m.workspaceBrowserIndex)
	}
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeySpace})
	m = updated.(setupProfilesModel)
	if m.mode != setupProfilesModeWorkspaces {
		t.Fatalf("Space select mode = %q, want workspaces", m.mode)
	}
	if !containsString(m.workspaceDraft, filepath.Join(root, "guides")) {
		t.Fatalf("workspace draft = %#v, want selected guides path", m.workspaceDraft)
	}

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(setupProfilesModel)
	if !m.result.WorkspacesSet || len(m.result.Workspaces) == 0 || m.result.Workspaces[len(m.result.Workspaces)-1] != filepath.Join(root, "guides") {
		t.Fatalf("result workspaces = %#v set=%v, want selected guides", m.result.Workspaces, m.result.WorkspacesSet)
	}
}

func TestSetupProfilesTUIBrowseAdvertisesAndSupportsEnterSave(t *testing.T) {
	m := newSetupProfilesModel(setupProfilesTUIState{
		Active: "main",
		Profiles: []setupProfileView{
			{Name: "main", Root: "/home/operator/.gormes", Active: true},
			{Name: "work", Root: "/home/operator/.gormes/profiles/work"},
		},
	})
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = updated.(setupProfilesModel)

	view := m.View()
	for _, want := range []string{
		"Up/Down move profile",
		"Enter save selected profile",
	} {
		if !strings.Contains(view, want) {
			t.Errorf("profile TUI browse view missing %q:\n%s", want, view)
		}
	}

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(setupProfilesModel)
	if m.result.Selected != "work" {
		t.Errorf("Enter selected profile = %q, want work", m.result.Selected)
	}
	if m.result.Cancelled {
		t.Error("Enter save marked result canceled")
	}
	if cmd == nil {
		t.Fatal("Enter in browse mode returned nil command, want tea.Quit")
	}
	if _, ok := cmd().(tea.QuitMsg); !ok {
		t.Fatalf("Enter in browse mode command = %T, want tea.QuitMsg", cmd())
	}
}

func TestSetupProfilesTUIFramesPadRowsToClearStaleProfileCells(t *testing.T) {
	m := newSetupProfilesModel(setupProfilesTUIState{
		Active: "main",
		Profiles: []setupProfileView{
			{Name: "main", Root: "/home/operator/.gormes", Active: true},
			{Name: "mineru", Root: "/home/operator/.gormes/profiles/mineru"},
		},
	})
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 30, Height: 40})
	m = updated.(setupProfilesModel)
	if first := m.View(); !strings.Contains(first, "Name: main") {
		t.Fatalf("initial frame missing main profile name:\n%s", first)
	}

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = updated.(setupProfilesModel)
	view := m.View()
	if !strings.Contains(view, "Name: mineru") {
		t.Fatalf("selected frame missing mineru profile name:\n%s", view)
	}
	for _, line := range strings.Split(view, "\n") {
		if got := lipgloss.Width(line); got != 30 {
			t.Fatalf("profile TUI line width = %d, want full terminal width 30 to clear stale cells:\n%q\n\nfull output:\n%s", got, line, view)
		}
	}
}

func TestSetupProfilesTUIBrowseSupportsVimNavigation(t *testing.T) {
	m := newSetupProfilesModel(setupProfilesTUIState{
		Active: "main",
		Profiles: []setupProfileView{
			{Name: "main", Root: "/home/operator/.gormes", Active: true},
			{Name: "work", Root: "/home/operator/.gormes/profiles/work"},
			{Name: "ops", Root: "/home/operator/.gormes/profiles/ops"},
		},
	})

	view := m.View()
	if !strings.Contains(view, "j/k or Up/Down move profile") {
		t.Fatalf("profile TUI browse view missing vim navigation hint:\n%s", view)
	}

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	m = updated.(setupProfilesModel)
	if got := m.currentProfile().Name; got != "work" {
		t.Fatalf("j selected profile = %q, want work", got)
	}

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	m = updated.(setupProfilesModel)
	if got := m.currentProfile().Name; got != "ops" {
		t.Fatalf("second j selected profile = %q, want ops", got)
	}

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	m = updated.(setupProfilesModel)
	if got := m.currentProfile().Name; got != "ops" {
		t.Fatalf("j at bottom selected profile = %q, want ops", got)
	}

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	m = updated.(setupProfilesModel)
	if got := m.currentProfile().Name; got != "work" {
		t.Fatalf("k selected profile = %q, want work", got)
	}
}

func TestSetupProfilesTUIChannelModeAdvertisesExistingShortcuts(t *testing.T) {
	m := newSetupProfilesModel(setupProfilesTUIState{
		Active: "main",
		Profiles: []setupProfileView{{
			Name:     "main",
			Root:     "/home/operator/.gormes",
			Active:   true,
			Channels: []string{"telegram"},
		}},
	})
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}})
	m = updated.(setupProfilesModel)

	view := m.View()
	for _, want := range []string{
		"j/k or Up/Down move",
		"q back",
	} {
		if !strings.Contains(view, want) {
			t.Errorf("profile TUI channel view missing %q:\n%s", want, view)
		}
	}

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	m = updated.(setupProfilesModel)
	if got := setupProfilesChannelChoices[m.channelIndex]; got != "whatsapp" {
		t.Fatalf("j selected channel = %q, want whatsapp", got)
	}

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	m = updated.(setupProfilesModel)
	if m.mode != setupProfilesModeBrowse {
		t.Fatalf("q in channel mode = %q, want browse", m.mode)
	}
	if m.result.Cancelled {
		t.Fatal("q back from channel mode should not cancel setup")
	}
}

func TestSetupProfilesTUIHardeningBoundsLongOperatorContent(t *testing.T) {
	long := strings.Repeat("x", 180)
	m := newSetupProfilesModel(setupProfilesTUIState{
		Active: "default-" + long,
		Profiles: []setupProfileView{
			{
				Name:       "default-" + long,
				Root:       "/home/operator/.gormes/profiles/main-" + long,
				Active:     true,
				Workspaces: []string{"/srv/alpha-" + long, "/srv/beta-" + long},
				Channels:   []string{"telegram", "discord"},
			},
		},
	})

	for _, size := range []struct{ width, height int }{{20, 10}, {40, 12}, {80, 24}} {
		t.Run(strings.Join([]string{strconv.Itoa(size.width), strconv.Itoa(size.height)}, "x"), func(t *testing.T) {
			updated, _ := m.Update(tea.WindowSizeMsg{Width: size.width, Height: size.height})
			m := updated.(setupProfilesModel)
			m = m.openWorkspaceEditor()
			m.workspaceDraft = append(m.workspaceDraft, "/tmp/new-workspace-"+long)

			view := m.View()
			if strings.TrimSpace(view) == "" {
				t.Fatal("setup profiles view is blank")
			}
			for _, line := range strings.Split(view, "\n") {
				if got := lipgloss.Width(line); got > size.width {
					t.Fatalf("setup profiles line width %d exceeds terminal width %d:\n%q\n\nfull output:\n%s", got, size.width, line, view)
				}
			}
			collapsed := strings.Join(strings.Fields(view), " ")
			for _, want := range []string{"Gormes profile setup", "omitted", "resize", "Workspace editor"} {
				if !strings.Contains(collapsed, want) {
					t.Fatalf("setup profiles view missing %q:\n%s", want, view)
				}
			}
		})
	}
}

func TestSetupProfilesTUIHardeningBoundsShortTerminalHeight(t *testing.T) {
	long := strings.Repeat("x", 240)
	m := newSetupProfilesModel(setupProfilesTUIState{
		Active: "main",
		Profiles: []setupProfileView{
			{Name: "main", Root: "/home/operator/.gormes", Active: true, Workspaces: []string{"/srv/" + long}},
			{Name: "very-long-profile-" + long, Root: "/tmp/" + long},
		},
	})
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 24, Height: 8})
	m = updated.(setupProfilesModel)
	m = m.openWorkspaceEditor()
	m.workspaceDraft = append(m.workspaceDraft, "/tmp/new-workspace-"+long)

	view := m.View()
	lines := strings.Split(view, "\n")
	if len(lines) > 8 {
		t.Fatalf("setup profiles view height = %d, want <= 8:\n%s", len(lines), view)
	}
	for _, line := range lines {
		if got := lipgloss.Width(line); got > 24 {
			t.Fatalf("setup profiles line width %d exceeds 24:\n%q\n\nfull output:\n%s", got, line, view)
		}
	}
	collapsed := strings.Join(strings.Fields(view), " ")
	for _, want := range []string{"Gormes profile setup", "omitted", "resize", "Workspace editor"} {
		if !strings.Contains(collapsed, want) {
			t.Fatalf("setup profiles short view missing %q:\n%s", want, view)
		}
	}
}

func TestSetupProfilesTUIHardeningBoundsShortChannelPicker(t *testing.T) {
	long := strings.Repeat("x", 180)
	m := newSetupProfilesModel(setupProfilesTUIState{
		Active: "main",
		Profiles: []setupProfileView{{
			Name:       "main",
			Root:       "/home/operator/.gormes/profiles/main-" + long,
			Active:     true,
			Workspaces: []string{"/srv/" + long},
			Channels:   []string{"telegram"},
		}},
	})
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 28, Height: 6})
	m = updated.(setupProfilesModel)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}})
	m = updated.(setupProfilesModel)

	view := m.View()
	lines := strings.Split(view, "\n")
	if len(lines) > 6 {
		t.Fatalf("setup profiles channel view height = %d, want <= 6:\n%s", len(lines), view)
	}
	for _, line := range lines {
		if got := lipgloss.Width(line); got > 28 {
			t.Fatalf("setup profiles channel line width %d exceeds 28:\n%q\n\nfull output:\n%s", got, line, view)
		}
	}
	collapsed := strings.Join(strings.Fields(view), " ")
	for _, want := range []string{"Gormes profile setup", "Channels", "telegram", "Space toggle", "resize"} {
		if !strings.Contains(collapsed, want) {
			t.Fatalf("setup profiles channel view missing %q:\n%s", want, view)
		}
	}
}

func TestSetupProfilesTUIHardeningBoundsShortAddProfile(t *testing.T) {
	long := strings.Repeat("x", 180)
	m := newSetupProfilesModel(setupProfilesTUIState{
		Active: "main",
		Profiles: []setupProfileView{{
			Name:       "main",
			Root:       "/home/operator/.gormes/profiles/main-" + long,
			Active:     true,
			Workspaces: []string{"/srv/" + long},
			Channels:   []string{"telegram"},
		}},
	})
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 28, Height: 6})
	m = updated.(setupProfilesModel)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	m = updated.(setupProfilesModel)
	m.input = "new-profile-" + long

	view := m.View()
	lines := strings.Split(view, "\n")
	if len(lines) > 6 {
		t.Fatalf("setup profiles add view height = %d, want <= 6:\n%s", len(lines), view)
	}
	for _, line := range lines {
		if got := lipgloss.Width(line); got > 28 {
			t.Fatalf("setup profiles add line width %d exceeds 28:\n%q\n\nfull output:\n%s", got, line, view)
		}
	}
	collapsed := strings.Join(strings.Fields(view), " ")
	for _, want := range []string{"Gormes profile setup", "New profile", "new-profile", "resize"} {
		if !strings.Contains(collapsed, want) {
			t.Fatalf("setup profiles add view missing %q:\n%s", want, view)
		}
	}
}

func TestSetupProfilesTUIAddsEditsAndReturnsSelection(t *testing.T) {
	m := newSetupProfilesModel(setupProfilesTUIState{
		Active: "main",
		Profiles: []setupProfileView{
			{Name: "main", Root: "/home/operator/.gormes", Active: true},
		},
	})
	tm := teatest.NewTestModel(t, m, teatest.WithInitialTermSize(100, 30))

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	tm.Type("work")
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'w'}})
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	tm.Type("/srv/work")
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}})
	tm.Send(tea.KeyMsg{Type: tea.KeySpace})
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})

	final := tm.FinalModel(t, teatest.WithFinalTimeout(2*time.Second)).(setupProfilesModel)
	if final.err != nil {
		t.Fatalf("profile TUI error = %v", final.err)
	}
	got := final.result
	if got.Cancelled {
		t.Fatal("result unexpectedly canceled")
	}
	if got.CreateName != "work" || got.Selected != "work" || !got.SetActive {
		t.Fatalf("result identity = %+v, want create/select/activate work", got)
	}
	if !got.WorkspacesSet || len(got.Workspaces) != 1 || got.Workspaces[0] != "/srv/work" {
		t.Fatalf("workspace result = %+v, want /srv/work", got)
	}
	if !got.ChannelsSet || len(got.Channels) != 1 || got.Channels[0] != "telegram" {
		t.Fatalf("channel result = %+v, want telegram", got)
	}
}

func TestSetupProfilesUsesRichTUIWhenTTY(t *testing.T) {
	home := t.TempDir()
	t.Setenv("GORMES_HOME", home)
	fake := &setupCommandFakeSeams{isTTY: true}

	stdin, err := os.Open(os.DevNull)
	if err != nil {
		t.Fatalf("open devnull: %v", err)
	}
	defer stdin.Close()

	oldInputIsTerminal := setupProfilesInputIsTerminal
	oldRunner := runSetupProfilesTUI
	setupProfilesInputIsTerminal = func(file *os.File) bool { return file == stdin }
	runSetupProfilesTUI = func(_ context.Context, gotStdin *os.File, _ io.Writer, state setupProfilesTUIState) (setupProfilesTUIResult, error) {
		if gotStdin != stdin {
			t.Fatalf("TUI stdin = %v, want injected stdin", gotStdin)
		}
		if len(state.Profiles) == 0 || state.Profiles[0].Name != "main" || !state.Profiles[0].Active {
			t.Fatalf("TUI state = %+v, want active main profile", state)
		}
		return setupProfilesTUIResult{
			CreateName:    "work",
			Selected:      "work",
			SetActive:     true,
			Workspaces:    []string{"/srv/work"},
			WorkspacesSet: true,
			Channels:      []string{"telegram"},
			ChannelsSet:   true,
		}, nil
	}
	t.Cleanup(func() {
		setupProfilesInputIsTerminal = oldInputIsTerminal
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
	if strings.Contains(out, "Create a new profile?") {
		t.Fatalf("TTY setup profiles must use the rich TUI, not text prompts:\n%s", out)
	}
	if !strings.Contains(out, "Created profile \"work\"") || !strings.Contains(out, "Active profile set to \"work\"") {
		t.Fatalf("TUI result not applied:\n%s", out)
	}
	if _, statErr := os.Stat(filepath.Join(config.GormesHome(), "profiles", "work")); statErr != nil {
		t.Fatalf("profile dir missing after TUI apply: %v", statErr)
	}
	active, readErr := os.ReadFile(filepath.Join(config.GormesHome(), "active_profile"))
	if readErr != nil {
		t.Fatalf("read active profile: %v", readErr)
	}
	if strings.TrimSpace(string(active)) != "work" {
		t.Fatalf("active profile = %q, want work", strings.TrimSpace(string(active)))
	}
	cfg, loadErr := config.Load(nil)
	if loadErr != nil {
		t.Fatalf("config.Load: %v", loadErr)
	}
	if got := cfg.Agents.Defaults.Workspaces; len(got) != 0 {
		t.Fatalf("default config should not receive named profile workspaces, got %#v", got)
	}
	raw, readErr := os.ReadFile(filepath.Join(config.GormesHome(), "profiles", "work", "config.toml"))
	if readErr != nil {
		t.Fatalf("read named profile config: %v", readErr)
	}
	if body := string(raw); !strings.Contains(body, "/srv/work") || !strings.Contains(body, "telegram") {
		t.Fatalf("named profile config missing TUI edits:\n%s", body)
	}
}
