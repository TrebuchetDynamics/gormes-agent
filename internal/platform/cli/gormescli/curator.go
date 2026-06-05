package gormescli

import (
	"time"

	"github.com/spf13/cobra"

	appcurator "github.com/TrebuchetDynamics/gormes-agent/internal/app/curator"
	"github.com/TrebuchetDynamics/gormes-agent/internal/extensibility/skills"
)

type CuratorBuildProvenance = appcurator.BuildProvenance

type CuratorCommandDeps struct {
	SkillsRoot func() string
	Now        func() time.Time
	Reviewer   skills.CuratorReviewer
}

func NewCuratorCommandWithDeps(deps CuratorCommandDeps, build func() BuildProvenance) *cobra.Command {
	appcurator.BuildProvenanceFunc = func() appcurator.BuildProvenance {
		if build == nil {
			return appcurator.BuildProvenance{}
		}
		provenance := build()
		return appcurator.BuildProvenance{Version: provenance.Version, GitCommit: provenance.GitCommit}
	}
	return appcurator.NewCommandWithDeps(appCuratorDeps(deps))
}

func ResolveCuratorSkillsRoot(deps CuratorCommandDeps) string {
	return appcurator.ResolveSkillsRoot(appCuratorDeps(deps))
}

func FormatCuratorTimestamp(ts *time.Time, deps CuratorCommandDeps) string {
	return appcurator.FormatTimestamp(ts, appCuratorDeps(deps))
}

func appCuratorDeps(deps CuratorCommandDeps) appcurator.CommandDeps {
	return appcurator.CommandDeps{
		SkillsRoot: deps.SkillsRoot,
		Now:        deps.Now,
		Reviewer:   deps.Reviewer,
	}
}
