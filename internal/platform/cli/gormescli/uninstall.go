package gormescli

import (
	"io"

	"github.com/spf13/cobra"

	appuninstall "github.com/TrebuchetDynamics/gormes-agent/internal/app/uninstall"
)

type UninstallBuildProvenance = appuninstall.BuildProvenance

type UninstallOptions = appuninstall.Options

type UninstallArtifactGroup = appuninstall.ArtifactGroup

type UninstallArtifactMover = appuninstall.ArtifactMover

func NewUninstallCommand(build func() BuildProvenance) *cobra.Command {
	appuninstall.BuildProvenanceFunc = func() appuninstall.BuildProvenance {
		if build == nil {
			return appuninstall.BuildProvenance{}
		}
		provenance := build()
		return appuninstall.BuildProvenance{Version: provenance.Version, GitCommit: provenance.GitCommit}
	}
	return appuninstall.NewCommand()
}

func RunUninstall(cmd *cobra.Command, opts UninstallOptions) error {
	return appuninstall.Run(cmd, opts)
}

func PrintUninstallDryRun(out io.Writer, groups []UninstallArtifactGroup) error {
	return appuninstall.PrintDryRun(out, groups)
}

func PrintUninstallDryRunJSON(out io.Writer, groups []UninstallArtifactGroup) error {
	return appuninstall.PrintDryRunJSON(out, groups)
}

func ExecuteUninstall(out, errOut io.Writer, groups []UninstallArtifactGroup) error {
	return appuninstall.Execute(out, errOut, groups)
}

func ExecuteUninstallJSON(out, errOut io.Writer, groups []UninstallArtifactGroup) error {
	return appuninstall.ExecuteJSON(out, errOut, groups)
}

func CollectUninstallArtifacts(home string) []UninstallArtifactGroup {
	return appuninstall.CollectArtifacts(home)
}

func PickUninstallArtifactMover() UninstallArtifactMover { return appuninstall.PickArtifactMover() }

func LegacyXDGUninstallGormesDir() string { return appuninstall.LegacyXDGGormesDir() }

func CollectPublishedBinaryPaths(home string) []string {
	return appuninstall.CollectPublishedBinaryPaths(home)
}

func PublishedBinaryCandidates() []string {
	return appuninstall.PublishedBinaryCandidates()
}

func SortedExisting(paths ...string) []string {
	return appuninstall.SortedExisting(paths...)
}

func RemoveUninstallGroup(groups []UninstallArtifactGroup, name string) []UninstallArtifactGroup {
	return appuninstall.RemoveGroup(groups, name)
}
