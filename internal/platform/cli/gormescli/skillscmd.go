package gormescli

import (
	"context"
	"io"

	appskillscmd "github.com/TrebuchetDynamics/gormes-agent/internal/app/skillscmd"
	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	skillruntime "github.com/TrebuchetDynamics/gormes-agent/internal/extensibility/skills"
	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli"
)

type SkillsBuildProvenance = appskillscmd.BuildProvenance
type SkillsProfileSyncSeams = appskillscmd.ProfileSyncSeams
type SkillsSyncOptions = appskillscmd.SyncOptions

func ListInstalledSkills(opts skillruntime.ListOptions, disabled map[string]struct{}) []skillruntime.SkillRow {
	return appskillscmd.ListInstalledSkills(opts, disabled)
}

func SkillsURLInstallDeps() cli.SkillsURLInstallDeps {
	return appskillscmd.URLInstallDeps()
}

func SkillsCommandOptionsForConfig(cfg config.Config) gateway.SkillsCommandOptions {
	return appskillscmd.CommandOptionsForConfig(cfg)
}

func RunSkillsProfileSync(ctx context.Context, out io.Writer, seams SkillsProfileSyncSeams, opts SkillsSyncOptions) error {
	return appskillscmd.RunProfileSync(ctx, out, seams, opts)
}

func DefaultSkillProfileRoots() ([]skillruntime.SkillProfileRoot, error) {
	return appskillscmd.DefaultProfileRoots()
}
