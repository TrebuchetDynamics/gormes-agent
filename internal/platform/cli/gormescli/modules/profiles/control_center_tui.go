package profiles

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/textvalue"
)

const (
	ControlCenterActionApplyDraft   ControlCenterActionCode = "apply"
	ControlCenterActionDiscardDraft ControlCenterActionCode = "discard"
)

var controlCenterProfileIDPattern = regexp.MustCompile(`^[a-z][a-z0-9_-]{0,63}$`)

type ControlCenterTUIScreenOptions struct {
	SelectedProfileID string
}

type ControlCenterTUIScreen struct {
	Title             string
	SelectedProfileID string
	Rows              []ControlCenterTUIRow
	Details           []string
	Issues            []ControlCenterIssue
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
	ProfileID    string
	CredentialID string
	Field        string
	Before       string
	After        string
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
	screen := ControlCenterTUIScreen{Title: "Profile Control Center", SelectedProfileID: selectedID, Issues: append([]ControlCenterIssue(nil), model.Issues...)}
	screen.Actions = append(screen.Actions, model.Actions...)
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
			screen.Details = controlCenterTUIDetails(profile)
			screen.Actions = append(screen.Actions, profile.Actions...)
		}
	}
	screen.Actions = append(screen.Actions, controlCenterAction(ControlCenterActionApplyDraft), controlCenterAction(ControlCenterActionDiscardDraft))
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
	writeControlCenterRowsForGroup(&b, screen.Rows, ControlCenterProfileGroupEnabled)
	writeControlCenterRowsForGroup(&b, screen.Rows, ControlCenterProfileGroupDisabled)
	if migration := controlCenterMigrationIssueText(screen.Issues); migration != "" {
		fmt.Fprintf(&b, "migration: %s\n", migration)
	}
	for _, detail := range screen.Details {
		fmt.Fprintln(&b, detail)
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

func controlCenterMigrationIssueText(issues []ControlCenterIssue) string {
	codes := make([]string, 0, len(issues))
	for _, issue := range issues {
		switch issue.Code {
		case ControlCenterIssueLegacyConfigDetected, ControlCenterIssueMigrationAvailable:
			codes = append(codes, string(issue.Code))
		}
	}
	if len(codes) == 0 {
		return ""
	}
	return strings.Join(codes, ", ")
}

func writeControlCenterRowsForGroup(b *strings.Builder, rows []ControlCenterTUIRow, group ControlCenterProfileGroup) {
	wroteHeader := false
	for _, row := range rows {
		if row.Group != group {
			continue
		}
		if !wroteHeader {
			fmt.Fprintf(b, "%s profiles:\n", group)
			wroteHeader = true
		}
		name := row.DisplayName
		if strings.TrimSpace(name) == "" {
			name = "(unnamed)"
		}
		marker := ""
		if row.Selected {
			marker = " selected"
		}
		fmt.Fprintf(b, "%s — %s — %s%s\n", row.ProfileID, name, row.Group, marker)
		fmt.Fprintf(b, "lanes: runtime=%s readiness=%s activity=%s\n", row.Runtime, row.Readiness, row.Activity)
	}
}

func controlCenterTUIDetails(profile ControlCenterProfile) []string {
	details := []string{fmt.Sprintf("details for profile: %s", profile.ID)}
	workspaces := make([]string, 0, len(profile.Workspaces))
	for _, workspace := range profile.Workspaces {
		workspaces = append(workspaces, workspace.Path)
	}
	details = append(details, fmt.Sprintf("workspaces: %s", controlCenterListOrNone(workspaces)))
	providerDetails := make([]string, 0, len(profile.Providers))
	for _, provider := range profile.Providers {
		providerDetails = append(providerDetails, controlCenterSurfaceDetail(provider))
	}
	details = append(details, fmt.Sprintf("providers: %s", controlCenterListOrNone(providerDetails)))
	channelDetails := make([]string, 0, len(profile.Channels))
	for _, channel := range profile.Channels {
		channelDetails = append(channelDetails, controlCenterSurfaceDetail(channel))
	}
	details = append(details, fmt.Sprintf("channels: %s", controlCenterListOrNone(channelDetails)))
	return details
}

func controlCenterSurfaceDetail(surface ControlCenterSurface) string {
	parts := []string{surface.ID}
	if surface.CredentialID != "" {
		parts = append(parts, "credential="+surface.CredentialID)
	}
	if surface.OwnerProfile != "" {
		parts = append(parts, "owner_profile="+surface.OwnerProfile)
	}
	parts = append(parts, "readiness="+string(surface.Readiness))
	return strings.Join(parts, " ")
}

func controlCenterListOrNone(values []string) string {
	if len(values) == 0 {
		return "(none)"
	}
	return strings.Join(values, "; ")
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

func (d *ControlCenterDraft) AddProfile(profileID, name string) error {
	profileID = strings.TrimSpace(profileID)
	if !controlCenterProfileIDPattern.MatchString(profileID) {
		return fmt.Errorf("profile control center draft: profile id %q is invalid", profileID)
	}
	if d.working.Profiles == nil {
		d.working.Profiles = map[string]config.ProfileCfg{}
	}
	if _, ok := d.working.Profiles[profileID]; ok {
		return fmt.Errorf("profile control center draft: profile %q already exists", profileID)
	}
	d.working.Profiles[profileID] = config.ProfileCfg{Enabled: true, Name: strings.TrimSpace(name)}
	return nil
}

func (d *ControlCenterDraft) SetProfileWorkspaces(profileID string, workspaces []string) error {
	profileID = strings.TrimSpace(profileID)
	profile, ok := d.working.Profiles[profileID]
	if profileID == "" || !ok {
		return fmt.Errorf("profile control center draft: profile %q not found", profileID)
	}
	profile.Workspaces = cleanControlCenterDraftStrings(workspaces)
	d.working.Profiles[profileID] = profile
	return nil
}

func (d *ControlCenterDraft) AssignProviderCredential(profileID, providerID, credentialID string) error {
	profileID = strings.TrimSpace(profileID)
	providerID = textvalue.LowerTrim(providerID)
	credentialID = strings.TrimSpace(credentialID)
	profile, ok := d.working.Profiles[profileID]
	if profileID == "" || !ok {
		return fmt.Errorf("profile control center draft: profile %q not found", profileID)
	}
	if providerID == "" || credentialID == "" {
		return fmt.Errorf("profile control center draft: provider and credential are required")
	}
	if profile.Providers == nil {
		profile.Providers = map[string]config.ProfileProviderCfg{}
	}
	provider := profile.Providers[providerID]
	provider.Enabled = true
	provider.Credential = credentialID
	profile.Providers[providerID] = provider
	d.working.Profiles[profileID] = profile
	return nil
}

func (d *ControlCenterDraft) SetProfileProviderModels(profileID, providerID, defaultModel string, allowedModels []string) error {
	profileID = strings.TrimSpace(profileID)
	providerID = textvalue.LowerTrim(providerID)
	profile, ok := d.working.Profiles[profileID]
	if profileID == "" || !ok {
		return fmt.Errorf("profile control center draft: profile %q not found", profileID)
	}
	if providerID == "" {
		return fmt.Errorf("profile control center draft: provider is required")
	}
	if profile.Providers == nil {
		profile.Providers = map[string]config.ProfileProviderCfg{}
	}
	provider := profile.Providers[providerID]
	provider.Enabled = true
	provider.DefaultModel = strings.TrimSpace(defaultModel)
	provider.AllowedModels = cleanControlCenterDraftStrings(allowedModels)
	profile.Providers[providerID] = provider
	d.working.Profiles[profileID] = profile
	return nil
}

func (d *ControlCenterDraft) AssignChannelCredential(profileID, channelID, credentialID string) error {
	profileID = strings.TrimSpace(profileID)
	channelID = textvalue.LowerTrim(channelID)
	credentialID = strings.TrimSpace(credentialID)
	profile, ok := d.working.Profiles[profileID]
	if profileID == "" || !ok {
		return fmt.Errorf("profile control center draft: profile %q not found", profileID)
	}
	if channelID == "" || credentialID == "" {
		return fmt.Errorf("profile control center draft: channel and credential are required")
	}
	if profile.Channels == nil {
		profile.Channels = map[string]config.ProfileChannelCfg{}
	}
	channel := profile.Channels[channelID]
	channel.Enabled = true
	channel.Credential = credentialID
	profile.Channels[channelID] = channel
	d.working.Profiles[profileID] = profile
	return nil
}

func (d *ControlCenterDraft) SetProfileChannelPolicy(profileID, channelID string, allowedChats, allowedUsers []string, requireMention bool, toolProgress string) error {
	profileID = strings.TrimSpace(profileID)
	channelID = textvalue.LowerTrim(channelID)
	profile, ok := d.working.Profiles[profileID]
	if profileID == "" || !ok {
		return fmt.Errorf("profile control center draft: profile %q not found", profileID)
	}
	if channelID == "" {
		return fmt.Errorf("profile control center draft: channel is required")
	}
	if profile.Channels == nil {
		profile.Channels = map[string]config.ProfileChannelCfg{}
	}
	channel := profile.Channels[channelID]
	channel.Enabled = true
	channel.AllowedChats = cleanControlCenterDraftStrings(allowedChats)
	channel.AllowedUsers = cleanControlCenterDraftStrings(allowedUsers)
	channel.RequireMention = requireMention
	channel.ToolProgress = textvalue.LowerTrim(toolProgress)
	profile.Channels[channelID] = channel
	d.working.Profiles[profileID] = profile
	return nil
}

func (d *ControlCenterDraft) SetProfileChannels(profileID string, channelIDs []string) error {
	profileID = strings.TrimSpace(profileID)
	profile, ok := d.working.Profiles[profileID]
	if profileID == "" || !ok {
		return fmt.Errorf("profile control center draft: profile %q not found", profileID)
	}
	channels := map[string]config.ProfileChannelCfg{}
	for _, channelID := range cleanControlCenterDraftStrings(channelIDs) {
		channelID = strings.ToLower(channelID)
		channel := profile.Channels[channelID]
		channel.Enabled = true
		channels[channelID] = channel
	}
	profile.Channels = channels
	d.working.Profiles[profileID] = profile
	return nil
}

func (d *ControlCenterDraft) SetCredential(credentialID string, credential config.CredentialCfg) error {
	credentialID = strings.TrimSpace(credentialID)
	if !controlCenterProfileIDPattern.MatchString(credentialID) {
		return fmt.Errorf("profile control center draft: credential id %q is invalid", credentialID)
	}
	credential.Kind = textvalue.LowerTrim(credential.Kind)
	credential.Provider = textvalue.LowerTrim(credential.Provider)
	credential.Channel = textvalue.LowerTrim(credential.Channel)
	credential.OwnerProfile = strings.TrimSpace(credential.OwnerProfile)
	if d.working.Credentials == nil {
		d.working.Credentials = map[string]config.CredentialCfg{}
	}
	d.working.Credentials[credentialID] = cloneControlCenterCredential(credential)
	return nil
}

func (d ControlCenterDraft) Preview() []ControlCenterDraftChange {
	return controlCenterDraftChanges(d.base, d.working)
}

func (d ControlCenterDraft) Apply() (config.Config, []ControlCenterDraftChange, error) {
	changes := d.Preview()
	return cloneControlCenterConfig(d.working), changes, nil
}

func (d ControlCenterDraft) Discard() config.Config {
	return cloneControlCenterConfig(d.base)
}

func RenderControlCenterDraftPreview(changes []ControlCenterDraftChange) []string {
	if len(changes) == 0 {
		return []string{"no profile draft changes"}
	}
	lines := make([]string, 0, len(changes))
	for _, change := range changes {
		subject := "profile " + change.ProfileID
		if change.CredentialID != "" {
			subject = "credential " + change.CredentialID
		}
		if change.Before == "" && change.Field == "created" {
			lines = append(lines, fmt.Sprintf("%s created: %s", subject, change.After))
			continue
		}
		lines = append(lines, fmt.Sprintf("%s %s: %s -> %s", subject, change.Field, renderControlCenterDraftValue(change.Field, change.Before), renderControlCenterDraftValue(change.Field, change.After)))
	}
	return lines
}

func renderControlCenterDraftValue(field, value string) string {
	if field == "name" || strings.HasSuffix(field, " credential") {
		return fmt.Sprintf("%q", value)
	}
	return value
}

func cleanControlCenterDraftStrings(values []string) []string {
	out := make([]string, 0, len(values))
	seen := map[string]bool{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		out = append(out, value)
	}
	return out
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
		before, beforeOK := base.Profiles[id]
		after, afterOK := working.Profiles[id]
		if !beforeOK && afterOK {
			changes = append(changes, ControlCenterDraftChange{ProfileID: id, Field: "created", After: fmt.Sprintf("enabled=%t name=%q", after.Enabled, after.Name)})
		}
		if before.Name != after.Name {
			changes = append(changes, ControlCenterDraftChange{ProfileID: id, Field: "name", Before: before.Name, After: after.Name})
		}
		if !equalControlCenterStrings(before.Workspaces, after.Workspaces) {
			changes = append(changes, ControlCenterDraftChange{ProfileID: id, Field: "workspaces", Before: formatControlCenterDraftList(before.Workspaces), After: formatControlCenterDraftList(after.Workspaces)})
		}
		changes = append(changes, controlCenterSurfaceDraftChanges(id, "provider", before.Providers, after.Providers)...)
		changes = append(changes, controlCenterSurfaceDraftChanges(id, "channel", before.Channels, after.Channels)...)
	}
	changes = append(changes, controlCenterCredentialDraftChanges(base.Credentials, working.Credentials)...)
	return changes
}

func controlCenterSurfaceDraftChanges[T interface {
	config.ProfileProviderCfg | config.ProfileChannelCfg
}](profileID, kind string, before, after map[string]T) []ControlCenterDraftChange {
	ids := sortedControlCenterDraftMapKeys(before, after)
	changes := make([]ControlCenterDraftChange, 0, len(ids))
	for _, id := range ids {
		beforeEnabled := controlCenterSurfaceDraftEnabled(before[id])
		afterEnabled := controlCenterSurfaceDraftEnabled(after[id])
		if beforeEnabled != afterEnabled {
			changes = append(changes, ControlCenterDraftChange{ProfileID: profileID, Field: kind + " " + id + " enabled", Before: fmt.Sprintf("%t", beforeEnabled), After: fmt.Sprintf("%t", afterEnabled)})
		}
		beforeCredential := controlCenterSurfaceDraftCredential(before[id])
		afterCredential := controlCenterSurfaceDraftCredential(after[id])
		if beforeCredential != afterCredential {
			changes = append(changes, ControlCenterDraftChange{ProfileID: profileID, Field: kind + " " + id + " credential", Before: beforeCredential, After: afterCredential})
		}
		if kind == "provider" {
			beforeModel := controlCenterProviderDraftDefaultModel(before[id])
			afterModel := controlCenterProviderDraftDefaultModel(after[id])
			if beforeModel != afterModel {
				changes = append(changes, ControlCenterDraftChange{ProfileID: profileID, Field: kind + " " + id + " default_model", Before: beforeModel, After: afterModel})
			}
			beforeAllowed := formatControlCenterDraftList(controlCenterProviderDraftAllowedModels(before[id]))
			afterAllowed := formatControlCenterDraftList(controlCenterProviderDraftAllowedModels(after[id]))
			if beforeAllowed != afterAllowed {
				changes = append(changes, ControlCenterDraftChange{ProfileID: profileID, Field: kind + " " + id + " allowed_models", Before: beforeAllowed, After: afterAllowed})
			}
		}
		if kind == "channel" {
			beforeChats := formatControlCenterDraftList(controlCenterChannelDraftAllowedChats(before[id]))
			afterChats := formatControlCenterDraftList(controlCenterChannelDraftAllowedChats(after[id]))
			if beforeChats != afterChats {
				changes = append(changes, ControlCenterDraftChange{ProfileID: profileID, Field: kind + " " + id + " allowed_chats", Before: beforeChats, After: afterChats})
			}
			beforeUsers := formatControlCenterDraftList(controlCenterChannelDraftAllowedUsers(before[id]))
			afterUsers := formatControlCenterDraftList(controlCenterChannelDraftAllowedUsers(after[id]))
			if beforeUsers != afterUsers {
				changes = append(changes, ControlCenterDraftChange{ProfileID: profileID, Field: kind + " " + id + " allowed_users", Before: beforeUsers, After: afterUsers})
			}
			beforeMention := controlCenterChannelDraftRequireMention(before[id])
			afterMention := controlCenterChannelDraftRequireMention(after[id])
			if beforeMention != afterMention {
				changes = append(changes, ControlCenterDraftChange{ProfileID: profileID, Field: kind + " " + id + " require_mention", Before: fmt.Sprintf("%t", beforeMention), After: fmt.Sprintf("%t", afterMention)})
			}
			beforeProgress := controlCenterChannelDraftToolProgress(before[id])
			afterProgress := controlCenterChannelDraftToolProgress(after[id])
			if beforeProgress != afterProgress {
				changes = append(changes, ControlCenterDraftChange{ProfileID: profileID, Field: kind + " " + id + " tool_progress", Before: beforeProgress, After: afterProgress})
			}
		}
	}
	return changes
}

func controlCenterSurfaceDraftEnabled(surface any) bool {
	switch typed := surface.(type) {
	case config.ProfileProviderCfg:
		return typed.Enabled
	case config.ProfileChannelCfg:
		return typed.Enabled
	default:
		return false
	}
}

func controlCenterSurfaceDraftCredential(surface any) string {
	switch typed := surface.(type) {
	case config.ProfileProviderCfg:
		return typed.Credential
	case config.ProfileChannelCfg:
		return typed.Credential
	default:
		return ""
	}
}

func controlCenterProviderDraftDefaultModel(surface any) string {
	provider, ok := surface.(config.ProfileProviderCfg)
	if !ok {
		return ""
	}
	return provider.DefaultModel
}

func controlCenterProviderDraftAllowedModels(surface any) []string {
	provider, ok := surface.(config.ProfileProviderCfg)
	if !ok {
		return nil
	}
	return provider.AllowedModels
}

func controlCenterChannelDraftAllowedChats(surface any) []string {
	channel, ok := surface.(config.ProfileChannelCfg)
	if !ok {
		return nil
	}
	return channel.AllowedChats
}

func controlCenterChannelDraftAllowedUsers(surface any) []string {
	channel, ok := surface.(config.ProfileChannelCfg)
	if !ok {
		return nil
	}
	return channel.AllowedUsers
}

func controlCenterChannelDraftRequireMention(surface any) bool {
	channel, ok := surface.(config.ProfileChannelCfg)
	return ok && channel.RequireMention
}

func controlCenterChannelDraftToolProgress(surface any) string {
	channel, ok := surface.(config.ProfileChannelCfg)
	if !ok {
		return ""
	}
	return channel.ToolProgress
}

func controlCenterCredentialDraftChanges(before, after map[string]config.CredentialCfg) []ControlCenterDraftChange {
	ids := sortedControlCenterDraftMapKeys(before, after)
	changes := make([]ControlCenterDraftChange, 0, len(ids))
	for _, id := range ids {
		beforeSecret := controlCenterSecretRefSummary(before[id].SecretRef)
		afterSecret := controlCenterSecretRefSummary(after[id].SecretRef)
		if beforeSecret != afterSecret {
			changes = append(changes, ControlCenterDraftChange{CredentialID: id, Field: "secret_ref", Before: beforeSecret, After: afterSecret})
		}
	}
	return changes
}

func controlCenterSecretRefSummary(ref *config.SecretRef) string {
	if ref == nil {
		return "none"
	}
	return "redacted_ref(" + string(ref.Source) + ")"
}

func sortedControlCenterDraftMapKeys[T any](left, right map[string]T) []string {
	seen := map[string]bool{}
	keys := make([]string, 0, len(left)+len(right))
	for key := range left {
		if !seen[key] {
			seen[key] = true
			keys = append(keys, key)
		}
	}
	for key := range right {
		if !seen[key] {
			seen[key] = true
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)
	return keys
}

func equalControlCenterStrings(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if left[i] != right[i] {
			return false
		}
	}
	return true
}

func formatControlCenterDraftList(values []string) string {
	if len(values) == 0 {
		return "[]"
	}
	return "[" + strings.Join(values, " ") + "]"
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
