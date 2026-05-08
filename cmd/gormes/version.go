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

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print gormes version",
	RunE: func(cmd *cobra.Command, _ []string) error {
		out := cmd.OutOrStdout()
		asJSON, _ := cmd.Flags().GetBool("json")
		if asJSON {
			body, err := json.MarshalIndent(map[string]string{
				"version":    Version,
				"date_alias": VersionDateAlias,
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
