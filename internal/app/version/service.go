package version

import (
	"encoding/json"
	"fmt"
	"io"
	"runtime"
	"runtime/debug"
	"strings"
)

// Info is the build identity surfaced by `gormes version`.
type Info struct {
	Version   string
	DateAlias string
	GitCommit string
	GitDirty  string
	BuildDate string
}

// ReportJSON is the machine-readable `gormes version --json` payload.
type ReportJSON struct {
	Version   string `json:"version"`
	DateAlias string `json:"date_alias"`
	GitCommit string `json:"git_commit"`
	GitDirty  bool   `json:"git_dirty"`
	BuildDate string `json:"build_date"`
	GoVersion string `json:"go_version"`
	OS        string `json:"os"`
	Arch      string `json:"arch"`
}

// BuildProvenance is the shared `{version, git_commit}` block used by command JSON reports.
type BuildProvenance struct {
	Version   string `json:"version"`
	GitCommit string `json:"git_commit"`
}

// NewBuildProvenance builds the shared command JSON provenance block.
func NewBuildProvenance(info Info) BuildProvenance {
	return BuildProvenance{Version: info.Version, GitCommit: ResolveGitCommit(info.GitCommit)}
}

// Run writes the version report in human or JSON form.
func Run(out io.Writer, info Info, asJSON bool) error {
	if asJSON {
		body, err := json.MarshalIndent(ReportJSON{
			Version:   info.Version,
			DateAlias: info.DateAlias,
			GitCommit: ResolveGitCommit(info.GitCommit),
			GitDirty:  ResolveGitDirty(info.GitDirty),
			BuildDate: ResolveBuildDate(info.BuildDate),
			GoVersion: runtime.Version(),
			OS:        runtime.GOOS,
			Arch:      runtime.GOARCH,
		}, "", "  ")
		if err != nil {
			return err
		}
		fmt.Fprintln(out, string(body))
		return nil
	}
	if ResolveGitDirty(info.GitDirty) {
		fmt.Fprintln(out, "gormes", info.Version, "(dirty)")
	} else {
		fmt.Fprintln(out, "gormes", info.Version)
	}
	return nil
}

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

// ResolveGitCommit resolves a source commit from ldflags first, then Go VCS build info.
func ResolveGitCommit(injected string) string {
	return ResolveGitCommitFrom(injected, buildInfoSettings())
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

// ResolveGitDirty resolves dirty state from ldflags first, then Go VCS build info.
func ResolveGitDirty(injected string) bool {
	return ResolveGitDirtyFrom(injected, buildInfoSettings())
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

// ResolveBuildDate resolves build date from ldflags first, then Go VCS build time.
func ResolveBuildDate(injected string) string {
	return ResolveBuildDateFrom(injected, buildInfoSettings())
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

func buildInfoSettings() []debug.BuildSetting {
	if info, ok := debug.ReadBuildInfo(); ok {
		return info.Settings
	}
	return nil
}
