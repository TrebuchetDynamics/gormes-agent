package docker

import (
	"path/filepath"
	"strings"
)

const defaultContainerWorkspace = "/workspace"

// MountPolicy defines allowed and blocked host paths for Docker mounts.
type MountPolicy struct {
	AllowedHostPaths []string
	BlockedPrefixes  []string
}

// MountEntry describes a resolved mount from host to container.
type MountEntry struct {
	HostPath      string
	ContainerPath string
	ReadOnly      bool
}

// DefaultMountPolicy returns a policy that blocks dangerous system paths
// and Docker socket access while allowing configured host paths.
func DefaultMountPolicy(hostPaths []string) MountPolicy {
	return MountPolicy{
		AllowedHostPaths: hostPaths,
		BlockedPrefixes: []string{
			"/etc/",
			"/proc/",
			"/sys/",
			"/var/run/docker.sock",
		},
	}
}

// IsBlocked reports whether a host path matches any blocked prefix.
func (p MountPolicy) IsBlocked(hostPath string) bool {
	clean := filepath.Clean(hostPath)
	for _, prefix := range p.BlockedPrefixes {
		if strings.HasPrefix(clean, prefix) || clean == strings.TrimSuffix(prefix, "/") {
			return true
		}
	}
	return false
}

// IsAllowed reports whether a host path is both in the allowlist and not blocked.
func (p MountPolicy) IsAllowed(hostPath string) bool {
	if p.IsBlocked(hostPath) {
		return false
	}
	clean := filepath.Clean(hostPath)
	for _, allowed := range p.AllowedHostPaths {
		if clean == filepath.Clean(allowed) {
			return true
		}
	}
	return false
}

// AllowedMounts returns the resolved mount entries for allowed host paths.
// Workspace paths (typically the working directory) are mapped read-write;
// other allowed paths are mapped read-only.
func (p MountPolicy) AllowedMounts(workspacePath, containerWorkspace string) []MountEntry {
	var mounts []MountEntry
	workspace := mountWorkspace{hostPath: filepath.Clean(workspacePath), containerPath: containerWorkspacePath(containerWorkspace)}
	for _, hostPath := range p.AllowedHostPaths {
		if entry, ok := p.mountCandidate(hostPath, workspace); ok {
			mounts = append(mounts, entry)
		}
	}
	return mounts
}

type mountWorkspace struct {
	hostPath      string
	containerPath string
}

func (p MountPolicy) mountCandidate(hostPath string, workspace mountWorkspace) (MountEntry, bool) {
	if p.IsBlocked(hostPath) {
		return MountEntry{}, false
	}
	cleanHost := filepath.Clean(hostPath)
	readOnly := true
	containerPath := cleanHost
	if cleanHost == workspace.hostPath {
		readOnly = false
		containerPath = workspace.containerPath
	}
	return MountEntry{
		HostPath:      cleanHost,
		ContainerPath: containerPath,
		ReadOnly:      readOnly,
	}, true
}

func containerWorkspacePath(containerWorkspace string) string {
	if containerWorkspace == "" {
		return defaultContainerWorkspace
	}
	return containerWorkspace
}
