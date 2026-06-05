package gormescli

import (
	"io"
	"runtime/debug"

	appversion "github.com/TrebuchetDynamics/gormes-agent/internal/app/version"
)

type VersionInfo = appversion.Info
type VersionReportJSON = appversion.ReportJSON
type VersionBuildProvenance = appversion.BuildProvenance

func NewVersionBuildProvenance(info VersionInfo) VersionBuildProvenance {
	return appversion.NewBuildProvenance(info)
}

func RunVersion(out io.Writer, info VersionInfo, asJSON bool) error {
	return appversion.Run(out, info, asJSON)
}

func ParseGitDirty(value string) bool {
	return appversion.ParseGitDirty(value)
}

func ResolveGitCommit(injected string) string {
	return appversion.ResolveGitCommit(injected)
}

func ResolveGitCommitFrom(injected string, settings []debug.BuildSetting) string {
	return appversion.ResolveGitCommitFrom(injected, settings)
}

func ResolveGitDirty(injected string) bool {
	return appversion.ResolveGitDirty(injected)
}

func ResolveGitDirtyFrom(injected string, settings []debug.BuildSetting) bool {
	return appversion.ResolveGitDirtyFrom(injected, settings)
}

func ResolveBuildDate(injected string) string {
	return appversion.ResolveBuildDate(injected)
}

func ResolveBuildDateFrom(injected string, settings []debug.BuildSetting) string {
	return appversion.ResolveBuildDateFrom(injected, settings)
}
