package gormescli

import (
	"runtime/debug"

	appversion "github.com/TrebuchetDynamics/gormes-agent/internal/app/version"
)

func ParseGitDirty(value string) bool {
	return appversion.ParseGitDirty(value)
}

func ResolveGitCommitFrom(injected string, settings []debug.BuildSetting) string {
	return appversion.ResolveGitCommitFrom(injected, settings)
}

func ResolveGitDirtyFrom(injected string, settings []debug.BuildSetting) bool {
	return appversion.ResolveGitDirtyFrom(injected, settings)
}

func ResolveBuildDateFrom(injected string, settings []debug.BuildSetting) string {
	return appversion.ResolveBuildDateFrom(injected, settings)
}
