package main

import (
	"encoding/json"
	"fmt"

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

// versionReportJSON is the wire shape for `gormes version --json`.
// Field order matches the rest of the --json arc: identity fields lead.
type versionReportJSON struct {
	Version   string `json:"version"`
	DateAlias string `json:"date_alias"`
	GitCommit string `json:"git_commit"`
}

var versionCmd = &cobra.Command{
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
			}, "", "  ")
			if err != nil {
				return err
			}
			fmt.Fprintln(out, string(body))
			return nil
		}
		fmt.Fprintln(out, "gormes", Version)
		return nil
	},
}

func init() {
	versionCmd.Flags().Bool("json", false, "emit a machine-readable {version, date_alias} JSON record (suitable for fleet automation)")
}
