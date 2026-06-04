package main

import (
	"io"

	"github.com/spf13/cobra"

	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/gormescli"
)

func newUninstallCommand() *cobra.Command {
	return gormescli.NewUninstallCommand(func() gormescli.BuildProvenance {
		provenance := newBuildProvenance()
		return gormescli.BuildProvenance{Version: provenance.Version, GitCommit: provenance.GitCommit}
	})
}

type uninstallOptions = gormescli.UninstallOptions

type artifactGroup = gormescli.UninstallArtifactGroup

type artifactMover struct {
	label string
	move  func(string) error
}

func runUninstall(cmd *cobra.Command, opts uninstallOptions) error {
	return gormescli.RunUninstall(cmd, opts)
}

func collectArtifacts(home string) []artifactGroup { return gormescli.CollectUninstallArtifacts(home) }

func collectPublishedBinaryPaths(home string) []string {
	return gormescli.CollectPublishedBinaryPaths(home)
}

func publishedBinaryCandidates() []string { return gormescli.PublishedBinaryCandidates() }

func legacyXDGGormesDir() string { return gormescli.LegacyXDGUninstallGormesDir() }

func sortedExisting(paths ...string) []string { return gormescli.SortedExisting(paths...) }

func removeGroup(groups []artifactGroup, name string) []artifactGroup {
	return gormescli.RemoveUninstallGroup(groups, name)
}

func printDryRun(out io.Writer, groups []artifactGroup) error {
	return gormescli.PrintUninstallDryRun(out, groups)
}

func printDryRunJSON(out io.Writer, groups []artifactGroup) error {
	return gormescli.PrintUninstallDryRunJSON(out, groups)
}

func executeUninstall(out, errOut io.Writer, groups []artifactGroup) error {
	return gormescli.ExecuteUninstall(out, errOut, groups)
}

func executeUninstallJSON(out, errOut io.Writer, groups []artifactGroup) error {
	return gormescli.ExecuteUninstallJSON(out, errOut, groups)
}

func pickArtifactMover() artifactMover {
	mover := gormescli.PickUninstallArtifactMover()
	return artifactMover{label: mover.Label, move: mover.Move}
}
