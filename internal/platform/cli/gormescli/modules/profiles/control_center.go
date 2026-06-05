package profiles

import (
	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/gormescli/modules/profiles/controlcenter"
)

// Control Center aliases keep the profiles package as the public CLI module
// boundary while the pure profile-control model, text rendering, and draft
// mechanics live in the focused controlcenter package.
type (
	ControlCenterProfileGroup = controlcenter.ControlCenterProfileGroup
	ControlCenterLaneStatus   = controlcenter.ControlCenterLaneStatus
	ControlCenterReadiness    = controlcenter.ControlCenterReadiness
	ControlCenterIssueCode    = controlcenter.ControlCenterIssueCode
	ControlCenterActionCode   = controlcenter.ControlCenterActionCode

	ControlCenterModelOptions = controlcenter.ControlCenterModelOptions
	ControlCenterModel        = controlcenter.ControlCenterModel
	ControlCenterProfile      = controlcenter.ControlCenterProfile
	ControlCenterLane         = controlcenter.ControlCenterLane
	ControlCenterWorkspace    = controlcenter.ControlCenterWorkspace
	ControlCenterSurface      = controlcenter.ControlCenterSurface
	ControlCenterIssue        = controlcenter.ControlCenterIssue
	ControlCenterAction       = controlcenter.ControlCenterAction

	ControlCenterTUIScreenOptions = controlcenter.ControlCenterTUIScreenOptions
	ControlCenterTUIScreen        = controlcenter.ControlCenterTUIScreen
	ControlCenterTUIRow           = controlcenter.ControlCenterTUIRow
	ControlCenterDraft            = controlcenter.ControlCenterDraft
	ControlCenterDraftChange      = controlcenter.ControlCenterDraftChange
)

const (
	ControlCenterProfileGroupEnabled  = controlcenter.ControlCenterProfileGroupEnabled
	ControlCenterProfileGroupDisabled = controlcenter.ControlCenterProfileGroupDisabled

	ControlCenterLaneReady     = controlcenter.ControlCenterLaneReady
	ControlCenterLaneAttention = controlcenter.ControlCenterLaneAttention
	ControlCenterLaneDisabled  = controlcenter.ControlCenterLaneDisabled
	ControlCenterLaneUnknown   = controlcenter.ControlCenterLaneUnknown

	ControlCenterReadinessReady             = controlcenter.ControlCenterReadinessReady
	ControlCenterReadinessDisabled          = controlcenter.ControlCenterReadinessDisabled
	ControlCenterReadinessMissingCredential = controlcenter.ControlCenterReadinessMissingCredential

	ControlCenterIssueNameNeeded                = controlcenter.ControlCenterIssueNameNeeded
	ControlCenterIssueWorkspaceMissing          = controlcenter.ControlCenterIssueWorkspaceMissing
	ControlCenterIssueProviderCredentialMissing = controlcenter.ControlCenterIssueProviderCredentialMissing
	ControlCenterIssueChannelCredentialMissing  = controlcenter.ControlCenterIssueChannelCredentialMissing
	ControlCenterIssueCredentialShared          = controlcenter.ControlCenterIssueCredentialShared
	ControlCenterIssueLegacyConfigDetected      = controlcenter.ControlCenterIssueLegacyConfigDetected
	ControlCenterIssueMigrationAvailable        = controlcenter.ControlCenterIssueMigrationAvailable

	ControlCenterActionCreateProfile       = controlcenter.ControlCenterActionCreateProfile
	ControlCenterActionEditProfile         = controlcenter.ControlCenterActionEditProfile
	ControlCenterActionAddProvider         = controlcenter.ControlCenterActionAddProvider
	ControlCenterActionAddChannel          = controlcenter.ControlCenterActionAddChannel
	ControlCenterActionEnableProfile       = controlcenter.ControlCenterActionEnableProfile
	ControlCenterActionDisableProfile      = controlcenter.ControlCenterActionDisableProfile
	ControlCenterActionMigrateLegacyConfig = controlcenter.ControlCenterActionMigrateLegacyConfig
	ControlCenterActionApplyDraft          = controlcenter.ControlCenterActionApplyDraft
	ControlCenterActionDiscardDraft        = controlcenter.ControlCenterActionDiscardDraft
)

func BuildControlCenterModel(cfg config.Config, opts ControlCenterModelOptions) ControlCenterModel {
	return controlcenter.BuildControlCenterModel(cfg, opts)
}

func ControlCenterActionCatalog() []ControlCenterAction {
	return controlcenter.ControlCenterActionCatalog()
}

func BuildControlCenterTUIScreen(model ControlCenterModel, opts ControlCenterTUIScreenOptions) ControlCenterTUIScreen {
	return controlcenter.BuildControlCenterTUIScreen(model, opts)
}

func NewControlCenterDraft(cfg config.Config) ControlCenterDraft {
	return controlcenter.NewControlCenterDraft(cfg)
}

func RenderControlCenterDraftPreview(changes []ControlCenterDraftChange) []string {
	return controlcenter.RenderControlCenterDraftPreview(changes)
}

func controlCenterAction(code ControlCenterActionCode) ControlCenterAction {
	for _, action := range ControlCenterActionCatalog() {
		if action.Code == code {
			return action
		}
	}
	return ControlCenterAction{Code: code, Label: "unsupported action", Available: false}
}
