package main

import (
	"time"

	"github.com/spf13/cobra"

	"github.com/TrebuchetDynamics/gormes-agent/internal/extensibility/skills"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/gormescli"
)

type curatorCommandDeps struct {
	skillsRoot func() string
	now        func() time.Time
	reviewer   skills.CuratorReviewer
}

func newCuratorCommand() *cobra.Command {
	return newCuratorCommandWithDeps(curatorCommandDeps{})
}

func newCuratorCommandWithDeps(deps curatorCommandDeps) *cobra.Command {
	return gormescli.NewCuratorCommandWithDeps(gormesCuratorDeps(deps), func() gormescli.BuildProvenance {
		build := newBuildProvenance()
		return gormescli.BuildProvenance{Version: build.Version, GitCommit: build.GitCommit}
	})
}

func gormesCuratorDeps(deps curatorCommandDeps) gormescli.CuratorCommandDeps {
	return gormescli.CuratorCommandDeps{
		SkillsRoot: deps.skillsRoot,
		Now:        deps.now,
		Reviewer:   deps.reviewer,
	}
}

func resolveCuratorSkillsRoot(deps curatorCommandDeps) string {
	return gormescli.ResolveCuratorSkillsRoot(gormesCuratorDeps(deps))
}

func formatCuratorTimestamp(ts *time.Time, deps curatorCommandDeps) string {
	return gormescli.FormatCuratorTimestamp(ts, gormesCuratorDeps(deps))
}
