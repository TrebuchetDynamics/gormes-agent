package profiles

import (
	"fmt"
	"sort"
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
)

const ControlCenterActionApplyDraft ControlCenterActionCode = "apply"

type ControlCenterTUIScreenOptions struct {
	SelectedProfileID string
}

type ControlCenterTUIScreen struct {
	Title             string
	SelectedProfileID string
	Rows              []ControlCenterTUIRow
	Actions           []ControlCenterAction
	AccessibleText    string
}

type ControlCenterTUIRow struct {
	ProfileID   string
	DisplayName string
	Group       ControlCenterProfileGroup
	Runtime     ControlCenterLaneStatus
	Readiness   ControlCenterLaneStatus
	Activity    ControlCenterLaneStatus
	Selected    bool
}

type ControlCenterDraft struct {
	base    config.Config
	working config.Config
}

type ControlCenterDraftChange struct {
	ProfileID string
	Field     string
	Before    string
	After     string
}

// BuildControlCenterTUIScreen renders the pure control-center read model into
// stable text-first TUI rows. The returned AccessibleText is the contract for
// screen readers and tests; richer Bubble Tea chrome can decorate these rows
// without changing the action catalog or reading secrets.
func BuildControlCenterTUIScreen(model ControlCenterModel, opts ControlCenterTUIScreenOptions) ControlCenterTUIScreen {
	selectedID := strings.TrimSpace(opts.SelectedProfileID)
	if selectedID == "" || !controlCenterModelHasProfile(model, selectedID) {
		if len(model.Profiles) > 0 {
			selectedID = model.Profiles[0].ID
		}
	}
	screen := ControlCenterTUIScreen{Title: "Profile Control Center", SelectedProfileID: selectedID}
	screen.Rows = make([]ControlCenterTUIRow, 0, len(model.Profiles))
	for _, profile := range model.Profiles {
		screen.Rows = append(screen.Rows, ControlCenterTUIRow{
			ProfileID:   profile.ID,
			DisplayName: profile.DisplayName,
			Group:       profile.Group,
			Runtime:     profile.Runtime.Status,
			Readiness:   profile.Readiness.Status,
			Activity:    profile.Activity.Status,
			Selected:    profile.ID == selectedID,
		})
		if profile.ID == selectedID {
			screen.Actions = append(screen.Actions, profile.Actions...)
		}
	}
	screen.Actions = append(screen.Actions, controlCenterAction(ControlCenterActionApplyDraft))
	screen.AccessibleText = buildControlCenterAccessibleText(screen)
	return screen
}

func controlCenterModelHasProfile(model ControlCenterModel, id string) bool {
	for _, profile := range model.Profiles {
		if profile.ID == id {
			return true
		}
	}
	return false
}

func buildControlCenterAccessibleText(screen ControlCenterTUIScreen) string {
	var b strings.Builder
	fmt.Fprintln(&b, screen.Title)
	if screen.SelectedProfileID != "" {
		fmt.Fprintf(&b, "selected profile: %s\n", screen.SelectedProfileID)
	}
	for _, row := range screen.Rows {
		name := row.DisplayName
		if strings.TrimSpace(name) == "" {
			name = "(unnamed)"
		}
		marker := ""
		if row.Selected {
			marker = " selected"
		}
		fmt.Fprintf(&b, "%s — %s — %s%s\n", row.ProfileID, name, row.Group, marker)
		fmt.Fprintf(&b, "lanes: runtime=%s readiness=%s activity=%s\n", row.Runtime, row.Readiness, row.Activity)
	}
	if len(screen.Actions) > 0 {
		labels := make([]string, 0, len(screen.Actions))
		for _, action := range screen.Actions {
			if !action.Available {
				continue
			}
			labels = append(labels, action.Label)
		}
		fmt.Fprintf(&b, "actions: %s\n", strings.Join(labels, ", "))
	}
	return b.String()
}

// NewControlCenterDraft creates a staged edit buffer for config v2 profile
// changes. The input config is copied so navigation/editing cannot mutate the
// persisted model before an explicit Apply.
func NewControlCenterDraft(cfg config.Config) ControlCenterDraft {
	base := cloneControlCenterConfig(cfg)
	return ControlCenterDraft{base: base, working: cloneControlCenterConfig(base)}
}

func (d *ControlCenterDraft) SetProfileDisplayName(profileID, name string) error {
	profileID = strings.TrimSpace(profileID)
	profile, ok := d.working.Profiles[profileID]
	if profileID == "" || !ok {
		return fmt.Errorf("profile control center draft: profile %q not found", profileID)
	}
	profile.Name = strings.TrimSpace(name)
	d.working.Profiles[profileID] = profile
	return nil
}

func (d ControlCenterDraft) Preview() []ControlCenterDraftChange {
	return controlCenterDraftChanges(d.base, d.working)
}

func (d ControlCenterDraft) Apply() (config.Config, []ControlCenterDraftChange, error) {
	changes := d.Preview()
	return cloneControlCenterConfig(d.working), changes, nil
}

func controlCenterDraftChanges(base, working config.Config) []ControlCenterDraftChange {
	ids := make([]string, 0, len(working.Profiles))
	seen := map[string]bool{}
	for id := range working.Profiles {
		ids = append(ids, id)
		seen[id] = true
	}
	for id := range base.Profiles {
		if !seen[id] {
			ids = append(ids, id)
		}
	}
	sort.Strings(ids)

	changes := make([]ControlCenterDraftChange, 0)
	for _, id := range ids {
		before := base.Profiles[id].Name
		after := working.Profiles[id].Name
		if before != after {
			changes = append(changes, ControlCenterDraftChange{ProfileID: id, Field: "name", Before: before, After: after})
		}
	}
	return changes
}

func cloneControlCenterConfig(cfg config.Config) config.Config {
	out := cfg
	if cfg.Profiles != nil {
		out.Profiles = make(map[string]config.ProfileCfg, len(cfg.Profiles))
		for id, profile := range cfg.Profiles {
			out.Profiles[id] = cloneControlCenterProfile(profile)
		}
	}
	if cfg.Credentials != nil {
		out.Credentials = make(map[string]config.CredentialCfg, len(cfg.Credentials))
		for id, credential := range cfg.Credentials {
			out.Credentials[id] = cloneControlCenterCredential(credential)
		}
	}
	return out
}

func cloneControlCenterProfile(profile config.ProfileCfg) config.ProfileCfg {
	out := profile
	out.Workspaces = append([]string(nil), profile.Workspaces...)
	out.Tags = append([]string(nil), profile.Tags...)
	if profile.Settings != nil {
		out.Settings = make(map[string]any, len(profile.Settings))
		for key, value := range profile.Settings {
			out.Settings[key] = value
		}
	}
	if profile.Providers != nil {
		out.Providers = make(map[string]config.ProfileProviderCfg, len(profile.Providers))
		for id, provider := range profile.Providers {
			provider.AllowedModels = append([]string(nil), provider.AllowedModels...)
			out.Providers[id] = provider
		}
	}
	if profile.Channels != nil {
		out.Channels = make(map[string]config.ProfileChannelCfg, len(profile.Channels))
		for id, channel := range profile.Channels {
			channel.AllowedChats = append([]string(nil), channel.AllowedChats...)
			channel.AllowedUsers = append([]string(nil), channel.AllowedUsers...)
			channel.Servers = append([]string(nil), channel.Servers...)
			out.Channels[id] = channel
		}
	}
	return out
}

func cloneControlCenterCredential(credential config.CredentialCfg) config.CredentialCfg {
	out := credential
	if credential.SecretRef != nil {
		ref := *credential.SecretRef
		out.SecretRef = &ref
	}
	return out
}
