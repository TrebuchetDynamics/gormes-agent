package profileapp

import (
	"github.com/spf13/cobra"

	appprofile "github.com/TrebuchetDynamics/gormes-agent/internal/app/profile"
	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/gormescli"
	profilemodule "github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/gormescli/modules/profiles"
)

type CommandSeams = appprofile.Seams

type CommandOptions = appprofile.Options

type ControlCenterModelOptions = profilemodule.ControlCenterModelOptions

type ControlCenterModel = profilemodule.ControlCenterModel

type ControlCenterWorkspace = profilemodule.ControlCenterWorkspace

type ControlCenterDraft = profilemodule.ControlCenterDraft

type ControlCenterDraftChange = profilemodule.ControlCenterDraftChange

func NewCommand(build func() gormescli.BuildProvenance) *cobra.Command {
	return appprofile.NewCommand(build)
}

func NewCommandWithSeams(seams CommandSeams, opts CommandOptions) *cobra.Command {
	return appprofile.NewCommandWithSeams(seams, opts)
}

func DefaultSeams() CommandSeams {
	return appprofile.DefaultSeams()
}

func DefaultListKnownProfiles() ([]string, error) {
	return appprofile.DefaultListKnownProfiles()
}

func ProfileSeedSeamsFromProfileSeams(seams CommandSeams, build func() gormescli.BuildProvenance) gormescli.ProfileSeedSeams {
	return appprofile.ProfileSeedSeamsFromProfileSeams(seams, build)
}

func SetupSections() []gormescli.SetupSection {
	return profilemodule.SetupSections()
}

func BuildControlCenterModel(cfg config.Config, opts ControlCenterModelOptions) ControlCenterModel {
	return profilemodule.BuildControlCenterModel(cfg, opts)
}

func NewControlCenterDraft(cfg config.Config) ControlCenterDraft {
	return profilemodule.NewControlCenterDraft(cfg)
}

func RenderControlCenterDraftPreview(changes []ControlCenterDraftChange) []string {
	return profilemodule.RenderControlCenterDraftPreview(changes)
}
