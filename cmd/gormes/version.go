package main

import (
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

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print gormes version",
	Run: func(_ *cobra.Command, _ []string) {
		fmt.Println("gormes", Version)
	},
}
