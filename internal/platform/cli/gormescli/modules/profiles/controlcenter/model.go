package controlcenter

import (
	"fmt"
	"sort"
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
)

type ControlCenterProfileGroup string

const (
	ControlCenterProfileGroupEnabled  ControlCenterProfileGroup = "enabled"
	ControlCenterProfileGroupDisabled ControlCenterProfileGroup = "disabled"
)

type ControlCenterLaneStatus string

const (
	ControlCenterLaneReady     ControlCenterLaneStatus = "ready"
	ControlCenterLaneAttention ControlCenterLaneStatus = "attention"
	ControlCenterLaneDisabled  ControlCenterLaneStatus = "disabled"
	ControlCenterLaneUnknown   ControlCenterLaneStatus = "unknown"
)

type ControlCenterReadiness string

const (
	ControlCenterReadinessReady             ControlCenterReadiness = "ready"
	ControlCenterReadinessDisabled          ControlCenterReadiness = "disabled"
	ControlCenterReadinessMissingCredential ControlCenterReadiness = "missing_credential"
)

type ControlCenterIssueCode string

const (
	ControlCenterIssueNameNeeded                ControlCenterIssueCode = "name_needed"
	ControlCenterIssueWorkspaceMissing          ControlCenterIssueCode = "workspace_missing"
	ControlCenterIssueProviderCredentialMissing ControlCenterIssueCode = "provider_credential_missing"
	ControlCenterIssueChannelCredentialMissing  ControlCenterIssueCode = "channel_credential_missing"
	ControlCenterIssueCredentialShared          ControlCenterIssueCode = "credential_shared"
	ControlCenterIssueLegacyConfigDetected      ControlCenterIssueCode = "legacy_config_detected"
	ControlCenterIssueMigrationAvailable        ControlCenterIssueCode = "migration_available"
)

type ControlCenterActionCode string

const (
	ControlCenterActionCreateProfile       ControlCenterActionCode = "create_profile"
	ControlCenterActionEditProfile         ControlCenterActionCode = "edit_profile"
	ControlCenterActionAddProvider         ControlCenterActionCode = "add_provider"
	ControlCenterActionAddChannel          ControlCenterActionCode = "add_channel"
	ControlCenterActionEnableProfile       ControlCenterActionCode = "enable_profile"
	ControlCenterActionDisableProfile      ControlCenterActionCode = "disable_profile"
	ControlCenterActionMigrateLegacyConfig ControlCenterActionCode = "migrate_legacy_profile_config"
)

type ControlCenterModelOptions struct {
	WorkspaceExists          func(path string) bool
	LegacyMigrationAvailable bool
}

type ControlCenterModel struct {
	Profiles []ControlCenterProfile
	Issues   []ControlCenterIssue
	Actions  []ControlCenterAction
}

type ControlCenterProfile struct {
	ID          string
	DisplayName string
	Enabled     bool
	Group       ControlCenterProfileGroup
	Runtime     ControlCenterLane
	Readiness   ControlCenterLane
	Activity    ControlCenterLane
	Workspaces  []ControlCenterWorkspace
	Providers   []ControlCenterSurface
	Channels    []ControlCenterSurface
	Actions     []ControlCenterAction
}

type ControlCenterLane struct {
	Status ControlCenterLaneStatus
	Issues []ControlCenterIssue
}

type ControlCenterWorkspace struct {
	Path   string
	Exists bool
}

type ControlCenterSurface struct {
	Kind         string
	ID           string
	Enabled      bool
	CredentialID string
	OwnerProfile string
	Shared       bool
	Readiness    ControlCenterReadiness
	Evidence     []string
}

type ControlCenterIssue struct {
	Code         ControlCenterIssueCode
	Severity     string
	ProfileID    string
	Subject      string
	CredentialID string
	OwnerProfile string
	Message      string
}

type ControlCenterAction struct {
	Code      ControlCenterActionCode
	Label     string
	Available bool
}

type credentialUse struct {
	profileID string
	kind      string
	surfaceID string
}

// BuildControlCenterModel projects profile config v2 into a pure operator read
// model. It never starts gateways, probes providers, resolves channel tokens,
// or reads secret values; callers inject any filesystem existence knowledge
// they want represented in the readiness lane.
func BuildControlCenterModel(cfg config.Config, opts ControlCenterModelOptions) ControlCenterModel {
	model := ControlCenterModel{Actions: []ControlCenterAction{controlCenterAction(ControlCenterActionCreateProfile)}}
	if !cfg.ProfileConfigV2Available() {
		model.Issues = append(model.Issues, ControlCenterIssue{
			Code:     ControlCenterIssueLegacyConfigDetected,
			Severity: "warning",
			Message:  "profile config v2 is not available; legacy config migration is required before profile services are current",
		})
	}
	if opts.LegacyMigrationAvailable || !cfg.ProfileConfigV2Available() {
		model.Issues = append(model.Issues, ControlCenterIssue{
			Code:     ControlCenterIssueMigrationAvailable,
			Severity: "info",
			Message:  "legacy profile config migration can be previewed and applied",
		})
		model.Actions = append(model.Actions, controlCenterAction(ControlCenterActionMigrateLegacyConfig))
	}
	if !cfg.ProfileConfigV2Available() {
		return model
	}

	uses := collectControlCenterCredentialUses(cfg)
	ids := sortedControlCenterProfileIDs(cfg.Profiles)
	model.Profiles = make([]ControlCenterProfile, 0, len(ids))
	for _, id := range ids {
		profile := cfg.Profiles[id]
		view := ControlCenterProfile{
			ID:          id,
			DisplayName: strings.TrimSpace(profile.Name),
			Enabled:     profile.Enabled,
			Group:       controlCenterProfileGroup(profile.Enabled),
			Runtime:     controlCenterRuntimeLane(profile.Enabled),
			Activity:    ControlCenterLane{Status: ControlCenterLaneUnknown},
			Workspaces:  controlCenterWorkspaces(profile.Workspaces, opts.WorkspaceExists),
			Providers:   controlCenterProviderSurfaces(id, profile.Providers, cfg.Credentials, uses),
			Channels:    controlCenterChannelSurfaces(id, profile.Channels, cfg.Credentials, uses),
			Actions:     controlCenterProfileActions(profile.Enabled),
		}
		view.Readiness = controlCenterReadinessLane(view)
		model.Profiles = append(model.Profiles, view)
	}
	return model
}

func collectControlCenterCredentialUses(cfg config.Config) map[string][]credentialUse {
	uses := map[string][]credentialUse{}
	for profileID, profile := range cfg.Profiles {
		for providerID, provider := range profile.Providers {
			credentialID := strings.TrimSpace(provider.Credential)
			if credentialID == "" {
				continue
			}
			uses[credentialID] = append(uses[credentialID], credentialUse{profileID: profileID, kind: "provider", surfaceID: providerID})
		}
		for channelID, channel := range profile.Channels {
			credentialID := strings.TrimSpace(channel.Credential)
			if credentialID == "" {
				continue
			}
			uses[credentialID] = append(uses[credentialID], credentialUse{profileID: profileID, kind: "channel", surfaceID: channelID})
		}
	}
	return uses
}

func sortedControlCenterProfileIDs(profiles map[string]config.ProfileCfg) []string {
	enabled := make([]string, 0, len(profiles))
	disabled := make([]string, 0, len(profiles))
	for id, profile := range profiles {
		if profile.Enabled {
			enabled = append(enabled, id)
		} else {
			disabled = append(disabled, id)
		}
	}
	sort.Strings(enabled)
	sort.Strings(disabled)
	return append(enabled, disabled...)
}

func controlCenterProfileGroup(enabled bool) ControlCenterProfileGroup {
	if enabled {
		return ControlCenterProfileGroupEnabled
	}
	return ControlCenterProfileGroupDisabled
}

func controlCenterRuntimeLane(enabled bool) ControlCenterLane {
	if !enabled {
		return ControlCenterLane{Status: ControlCenterLaneDisabled}
	}
	return ControlCenterLane{Status: ControlCenterLaneReady}
}

func controlCenterWorkspaces(paths []string, exists func(string) bool) []ControlCenterWorkspace {
	out := make([]ControlCenterWorkspace, 0, len(paths))
	seen := map[string]bool{}
	for _, path := range paths {
		path = strings.TrimSpace(path)
		if path == "" || seen[path] {
			continue
		}
		seen[path] = true
		workspace := ControlCenterWorkspace{Path: path, Exists: true}
		if exists != nil {
			workspace.Exists = exists(path)
		}
		out = append(out, workspace)
	}
	return out
}

func controlCenterProviderSurfaces(profileID string, providers map[string]config.ProfileProviderCfg, credentials map[string]config.CredentialCfg, uses map[string][]credentialUse) []ControlCenterSurface {
	ids := sortedStringKeys(providers)
	out := make([]ControlCenterSurface, 0, len(ids))
	for _, id := range ids {
		provider := providers[id]
		out = append(out, controlCenterSurface(profileID, "provider", id, provider.Enabled, provider.Credential, credentials, uses))
	}
	return out
}

func controlCenterChannelSurfaces(profileID string, channels map[string]config.ProfileChannelCfg, credentials map[string]config.CredentialCfg, uses map[string][]credentialUse) []ControlCenterSurface {
	ids := sortedStringKeys(channels)
	out := make([]ControlCenterSurface, 0, len(ids))
	for _, id := range ids {
		channel := channels[id]
		out = append(out, controlCenterSurface(profileID, "channel", id, channel.Enabled, channel.Credential, credentials, uses))
	}
	return out
}

func controlCenterSurface(profileID, kind, id string, enabled bool, credentialID string, credentials map[string]config.CredentialCfg, uses map[string][]credentialUse) ControlCenterSurface {
	credentialID = strings.TrimSpace(credentialID)
	surface := ControlCenterSurface{Kind: kind, ID: id, Enabled: enabled, CredentialID: credentialID, Readiness: ControlCenterReadinessReady}
	if !enabled {
		surface.Readiness = ControlCenterReadinessDisabled
		return surface
	}
	credential, ok := credentials[credentialID]
	if credentialID == "" || !ok || !credentialMatchesSurface(credential, kind, id) {
		surface.Readiness = ControlCenterReadinessMissingCredential
		surface.Evidence = append(surface.Evidence, kind+"_credential_missing")
		return surface
	}
	surface.OwnerProfile = credential.OwnerProfile
	if len(uses[credentialID]) > 1 {
		surface.Shared = true
		surface.Evidence = append(surface.Evidence, "credential_shared_from:"+credential.OwnerProfile)
	}
	return surface
}

func credentialMatchesSurface(credential config.CredentialCfg, kind, id string) bool {
	if credential.Kind != "" && credential.Kind != kind {
		return false
	}
	switch kind {
	case "provider":
		return credential.Provider == "" || credential.Provider == id
	case "channel":
		return credential.Channel == "" || credential.Channel == id
	default:
		return false
	}
}

func controlCenterReadinessLane(profile ControlCenterProfile) ControlCenterLane {
	if !profile.Enabled {
		return ControlCenterLane{Status: ControlCenterLaneDisabled}
	}
	issues := make([]ControlCenterIssue, 0)
	if profile.DisplayName == "" {
		issues = append(issues, ControlCenterIssue{
			Code:      ControlCenterIssueNameNeeded,
			Severity:  "warning",
			ProfileID: profile.ID,
			Subject:   profile.ID,
			Message:   "profile display name is blank",
		})
	}
	for _, workspace := range profile.Workspaces {
		if !workspace.Exists {
			issues = append(issues, ControlCenterIssue{
				Code:      ControlCenterIssueWorkspaceMissing,
				Severity:  "warning",
				ProfileID: profile.ID,
				Subject:   workspace.Path,
				Message:   "workspace path is not present",
			})
		}
	}
	for _, provider := range profile.Providers {
		issues = append(issues, controlCenterSurfaceIssues(profile.ID, provider, ControlCenterIssueProviderCredentialMissing)...)
	}
	for _, channel := range profile.Channels {
		issues = append(issues, controlCenterSurfaceIssues(profile.ID, channel, ControlCenterIssueChannelCredentialMissing)...)
	}
	status := ControlCenterLaneReady
	for _, issue := range issues {
		switch issue.Code {
		case ControlCenterIssueCredentialShared:
			continue
		default:
			status = ControlCenterLaneAttention
		}
	}
	return ControlCenterLane{Status: status, Issues: issues}
}

func controlCenterSurfaceIssues(profileID string, surface ControlCenterSurface, missingCode ControlCenterIssueCode) []ControlCenterIssue {
	issues := make([]ControlCenterIssue, 0, 2)
	if surface.Readiness == ControlCenterReadinessMissingCredential {
		issues = append(issues, ControlCenterIssue{
			Code:         missingCode,
			Severity:     "warning",
			ProfileID:    profileID,
			Subject:      surface.ID,
			CredentialID: surface.CredentialID,
			Message:      fmt.Sprintf("%s %s has no usable credential", surface.Kind, surface.ID),
		})
	}
	if surface.Shared {
		issues = append(issues, ControlCenterIssue{
			Code:         ControlCenterIssueCredentialShared,
			Severity:     "info",
			ProfileID:    profileID,
			Subject:      surface.ID,
			CredentialID: surface.CredentialID,
			OwnerProfile: surface.OwnerProfile,
			Message:      fmt.Sprintf("credential %s is shared from owner_profile=%s", surface.CredentialID, surface.OwnerProfile),
		})
	}
	return issues
}

func controlCenterProfileActions(enabled bool) []ControlCenterAction {
	actions := []ControlCenterAction{
		controlCenterAction(ControlCenterActionEditProfile),
		controlCenterAction(ControlCenterActionAddProvider),
		controlCenterAction(ControlCenterActionAddChannel),
	}
	if enabled {
		actions = append(actions, controlCenterAction(ControlCenterActionDisableProfile))
	} else {
		actions = append(actions, controlCenterAction(ControlCenterActionEnableProfile))
	}
	return actions
}

func ControlCenterActionCatalog() []ControlCenterAction {
	codes := []ControlCenterActionCode{
		ControlCenterActionCreateProfile,
		ControlCenterActionEditProfile,
		ControlCenterActionAddProvider,
		ControlCenterActionAddChannel,
		ControlCenterActionEnableProfile,
		ControlCenterActionDisableProfile,
		ControlCenterActionMigrateLegacyConfig,
		ControlCenterActionApplyDraft,
		ControlCenterActionDiscardDraft,
	}
	catalog := make([]ControlCenterAction, 0, len(codes))
	for _, code := range codes {
		catalog = append(catalog, controlCenterAction(code))
	}
	return catalog
}

func controlCenterAction(code ControlCenterActionCode) ControlCenterAction {
	label, ok := controlCenterActionLabels()[code]
	if !ok {
		return ControlCenterAction{Code: code, Label: "unsupported action", Available: false}
	}
	return ControlCenterAction{Code: code, Label: label, Available: true}
}

func controlCenterActionLabels() map[ControlCenterActionCode]string {
	return map[ControlCenterActionCode]string{
		ControlCenterActionCreateProfile:       "create profile",
		ControlCenterActionEditProfile:         "edit profile",
		ControlCenterActionAddProvider:         "add provider",
		ControlCenterActionAddChannel:          "add channel",
		ControlCenterActionEnableProfile:       "enable profile",
		ControlCenterActionDisableProfile:      "disable profile",
		ControlCenterActionMigrateLegacyConfig: "migrate legacy profile config",
		ControlCenterActionApplyDraft:          "apply",
		ControlCenterActionDiscardDraft:        "discard",
	}
}

func sortedStringKeys[T any](values map[string]T) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func (m ControlCenterModel) String() string {
	var b strings.Builder
	for _, issue := range m.Issues {
		fmt.Fprintf(&b, "%s %s %s\n", issue.Code, issue.Subject, issue.Message)
	}
	for _, profile := range m.Profiles {
		fmt.Fprintf(&b, "%s %s %s\n", profile.Group, profile.ID, profile.DisplayName)
		for _, issue := range profile.Readiness.Issues {
			fmt.Fprintf(&b, "  %s %s %s owner_profile=%s credential=%s\n", issue.Code, issue.Subject, issue.Message, issue.OwnerProfile, issue.CredentialID)
		}
	}
	return b.String()
}
