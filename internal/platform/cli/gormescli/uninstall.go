package gormescli

import appuninstall "github.com/TrebuchetDynamics/gormes-agent/internal/app/uninstall"

type UninstallArtifactGroup = appuninstall.ArtifactGroup

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
