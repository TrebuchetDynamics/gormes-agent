package main

import (
	"encoding/json"
	"fmt"
	"runtime"
	"strings"

	"github.com/spf13/cobra"
)

// Version marks the current operator-facing release line.
//
// Gormes adopts the Hermes-style dual taxonomy: the canonical semver tag (the
// value below) is paired with a date alias of the form vYYYY.M.D in release
// notes, release.json, and the GitHub release title. The git tag remains
// v<Version> until the release workflow learns to extract version from this
// file independently of the tag string.
var Version = "0.1.07"

// VersionDateAlias is the Hermes-style vYYYY.M.D paired alias for the
// current release. Bumped together with Version on every release. Fleet
// automation comparing Gormes deployments against Hermes upstream
// baselines (whose own version IS the date) consumes this through
// `gormes version --json`.
var VersionDateAlias = "v2026.5.7"

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

// versionReportJSON is the wire shape for `gormes version --json`.
// Field order matches the rest of the --json arc: identity fields lead.
type versionReportJSON struct {
	Version   string `json:"version"`
	DateAlias string `json:"date_alias"`
	GitCommit string `json:"git_commit"`
	GitDirty  bool   `json:"git_dirty"`
	GoVersion string `json:"go_version"`
	OS        string `json:"os"`
	Arch      string `json:"arch"`
}

// buildProvenanceJSON is the `{version, git_commit}` block prepended to
// every `--json` document that reports captured runtime/operator state
// (update, doctor, status, restore, auth-status, secrets, gateway-status).
// Same wire shape across commands keeps fleet automation parsing one
// type. `version --json` carries a richer surface (`versionReportJSON`)
// because that command IS the binary-identity contract; everything else
// just needs the minimum to attribute a snapshot to a binary.
type buildProvenanceJSON struct {
	Version   string `json:"version"`
	GitCommit string `json:"git_commit"`
}

// newBuildProvenance returns the build-provenance block populated from
// the package-level Version + GitCommit ldflags-injected values.
func newBuildProvenance() buildProvenanceJSON {
	return buildProvenanceJSON{Version: Version, GitCommit: GitCommit}
}

// parseGitDirty interprets the build-time GitDirty string as a bool.
// "true" (case-insensitive) and "1" mean dirty; everything else is
// clean. Robust to ldflags injection variants.
func parseGitDirty(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "true", "1", "yes", "y":
		return true
	}
	return false
}

// newVersionCommand returns a fresh `gormes version` cobra.Command.
// Constructor pattern (rather than a package-level var with init-time
// flag registration) avoids cross-test flag-value contamination on the
// shared cobra FlagSet — each newRootCommand() builds an independent
// instance.
func newVersionCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "version",
		Short: "Print gormes version",
		RunE: func(cmd *cobra.Command, _ []string) error {
			out := cmd.OutOrStdout()
			asJSON, _ := cmd.Flags().GetBool("json")
			if asJSON {
				body, err := json.MarshalIndent(versionReportJSON{
					Version:   Version,
					DateAlias: VersionDateAlias,
					GitCommit: GitCommit,
					GitDirty:  parseGitDirty(GitDirty),
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
			if parseGitDirty(GitDirty) {
				fmt.Fprintln(out, "gormes", Version, "(dirty)")
			} else {
				fmt.Fprintln(out, "gormes", Version)
			}
			return nil
		},
	}
	cmd.Flags().Bool("json", false, "emit a machine-readable {version, date_alias, git_commit} JSON record (suitable for fleet automation)")
	return cmd
}
