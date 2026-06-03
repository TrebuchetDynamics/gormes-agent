package version

import (
	"runtime/debug"
	"strings"
)

// ParseGitDirty interprets the build-time GitDirty string as a bool.
// "true" (case-insensitive) and "1" mean dirty; common yes/y variants are
// also accepted for robustness to ldflags injection variants.
func ParseGitDirty(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "true", "1", "yes", "y":
		return true
	}
	return false
}

// ResolveGitCommitFrom resolves a source commit from ldflags first, then Go VCS build info.
func ResolveGitCommitFrom(injected string, settings []debug.BuildSetting) string {
	if injected != "" && injected != "unknown" {
		return injected
	}
	for _, s := range settings {
		if s.Key == "vcs.revision" && s.Value != "" {
			if len(s.Value) >= 9 {
				return s.Value[:9]
			}
			return s.Value
		}
	}
	return "unknown"
}

// ResolveGitDirtyFrom resolves dirty state from ldflags first, then Go VCS build info.
func ResolveGitDirtyFrom(injected string, settings []debug.BuildSetting) bool {
	if ParseGitDirty(injected) {
		return true
	}
	for _, s := range settings {
		if s.Key == "vcs.modified" {
			return strings.EqualFold(s.Value, "true")
		}
	}
	return false
}

// ResolveBuildDateFrom resolves build date from ldflags first, then Go VCS build time.
func ResolveBuildDateFrom(injected string, settings []debug.BuildSetting) string {
	if injected != "" && injected != "unknown" {
		return injected
	}
	for _, s := range settings {
		if s.Key == "vcs.time" && s.Value != "" {
			return s.Value
		}
	}
	return "unknown"
}
