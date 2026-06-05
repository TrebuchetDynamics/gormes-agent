package gormescli

import (
	"io"

	"github.com/spf13/cobra"
)

func newUninstallCommand() *cobra.Command {
	return NewUninstallCommand(func() BuildProvenance {
		return BuildProvenance{Version: Version, GitCommit: "test-git"}
	})
}

type uninstallOptions = UninstallOptions

type artifactGroup = UninstallArtifactGroup

type artifactMover struct {
	label string
	move  func(string) error
}

func runUninstall(cmd *cobra.Command, opts uninstallOptions) error { return RunUninstall(cmd, opts) }

func collectArtifacts(home string) []artifactGroup { return CollectUninstallArtifacts(home) }

func collectPublishedBinaryPaths(home string) []string { return CollectPublishedBinaryPaths(home) }

func publishedBinaryCandidates() []string { return PublishedBinaryCandidates() }

func legacyXDGGormesDir() string { return LegacyXDGUninstallGormesDir() }

func sortedExisting(paths ...string) []string { return SortedExisting(paths...) }

func removeGroup(groups []artifactGroup, name string) []artifactGroup {
	return RemoveUninstallGroup(groups, name)
}

func printDryRun(out io.Writer, groups []artifactGroup) error {
	return PrintUninstallDryRun(out, groups)
}

func printDryRunJSON(out io.Writer, groups []artifactGroup) error {
	return PrintUninstallDryRunJSON(out, groups)
}

func executeUninstall(out, errOut io.Writer, groups []artifactGroup) error {
	return ExecuteUninstall(out, errOut, groups)
}

func executeUninstallJSON(out, errOut io.Writer, groups []artifactGroup) error {
	return ExecuteUninstallJSON(out, errOut, groups)
}

func pickArtifactMover() artifactMover {
	mover := PickUninstallArtifactMover()
	return artifactMover{label: mover.Label, move: mover.Move}
}
