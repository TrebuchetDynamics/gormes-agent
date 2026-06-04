package main

import (
	"runtime/debug"

	"github.com/spf13/cobra"

	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/gormescli"
)

// Version marks the current operator-facing release line.
//
// Gormes adopts the Hermes-style dual taxonomy: the canonical semver tag (the
// value below) is paired with a date alias of the form vYYYY.M.D in release
// notes, release.json, and the GitHub release title. The git tag remains
// v<Version> until the release workflow learns to extract version from this
// file independently of the tag string.
var Version = "0.2.23"

// VersionDateAlias is the Hermes-style vYYYY.M.D paired alias for the
// current release. Bumped together with Version on every release. Fleet
// automation (whose own version IS the date) consumes this through
// `gormes version --json`.
var VersionDateAlias = "v2026.5.25"

// GitCommit is the source SHA the binary was built from. Defaults to
// "unknown" in dev/source builds; release CI is expected to inject the
// real value via `-ldflags="-X main.GitCommit=<sha>"`. Fleet automation
// verifying binaries against a specific commit reads this field
// through `gormes version --json`.
var GitCommit = "unknown"

// GitDirty marks whether the source tree had uncommitted changes when
// the binary was built. Stored as a string so the value is settable
// via ldflags; parsed to bool for JSON output. Accepts "true"/"1" as
// dirty, anything else (including the default empty/false) as clean.
// Release CI is expected to inject the actual flag — dev/source
// builds default to clean.
var GitDirty = "false"

// BuildDate is the UTC timestamp when the binary was built. Defaults to
// "unknown" in dev/source builds; release CI injects the real value via
// `-ldflags="-X main.BuildDate=<RFC3339 UTC>"`.
var BuildDate = "unknown"

// buildProvenanceJSON is the `{version, git_commit}` block prepended to
// every `--json` document that reports captured runtime/operator state.
type buildProvenanceJSON = gormescli.VersionBuildProvenance

func versionInfo() gormescli.VersionInfo {
	return gormescli.VersionInfo{
		Version:   Version,
		DateAlias: VersionDateAlias,
		GitCommit: GitCommit,
		GitDirty:  GitDirty,
		BuildDate: BuildDate,
	}
}

func newBuildProvenance() buildProvenanceJSON {
	return gormescli.NewVersionBuildProvenance(versionInfo())
}

func parseGitDirty(value string) bool {
	return gormescli.ParseGitDirty(value)
}

func resolveGitCommit() string {
	return gormescli.ResolveGitCommit(GitCommit)
}

func resolveGitCommitFrom(injected string, settings []debug.BuildSetting) string {
	return gormescli.ResolveGitCommitFrom(injected, settings)
}

func resolveGitDirty() bool {
	return gormescli.ResolveGitDirty(GitDirty)
}

func resolveGitDirtyFrom(injected string, settings []debug.BuildSetting) bool {
	return gormescli.ResolveGitDirtyFrom(injected, settings)
}

func resolveBuildDate() string {
	return gormescli.ResolveBuildDate(BuildDate)
}

func resolveBuildDateFrom(injected string, settings []debug.BuildSetting) string {
	return gormescli.ResolveBuildDateFrom(injected, settings)
}

func newVersionCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "version",
		Short: "Print gormes version",
		RunE: func(cmd *cobra.Command, _ []string) error {
			asJSON, _ := cmd.Flags().GetBool("json")
			return gormescli.RunVersion(cmd.OutOrStdout(), versionInfo(), asJSON)
		},
	}
	cmd.Flags().Bool("json", false, "emit a machine-readable {version, date_alias, git_commit, build_date} JSON record (suitable for fleet automation)")
	return cmd
}
