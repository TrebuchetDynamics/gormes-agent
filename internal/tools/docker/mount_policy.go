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
	clean, ok := normalizeMountHostPath(hostPath)
	if !ok {
		return false
	}
	for _, prefix := range p.BlockedPrefixes {
		if blockedPrefixMatches(clean, prefix) {
			return true
		}
	}
	return false
}

func normalizeMountHostPath(hostPath string) (string, bool) {
	clean := filepath.Clean(strings.TrimSpace(hostPath))
	if clean == "." || !filepath.IsAbs(clean) {
		return "", false
	}
	return clean, true
}

func blockedPrefixMatches(cleanHostPath, rawPrefix string) bool {
	cleanPrefix := filepath.Clean(strings.TrimSpace(rawPrefix))
	if cleanPrefix == "." {
		return false
	}
	return cleanHostPath == cleanPrefix || strings.HasPrefix(cleanHostPath, cleanPrefix+string(filepath.Separator))
}

// IsAllowed reports whether a host path is both in the allowlist and not blocked.
func (p MountPolicy) IsAllowed(hostPath string) bool {
	clean, ok := normalizeMountHostPath(hostPath)
	if !ok || p.IsBlocked(clean) {
		return false
	}
	for _, allowed := range p.AllowedHostPaths {
		allowedClean, ok := normalizeMountHostPath(allowed)
		if ok && clean == allowedClean {
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
	seen := map[string]struct{}{}
	workspaceHost, _ := normalizeMountHostPath(workspacePath)
	workspace := mountWorkspace{hostPath: workspaceHost, containerPath: containerWorkspacePath(containerWorkspace)}
	for _, hostPath := range p.AllowedHostPaths {
		candidate := p.classifyMountCandidate(hostPath, workspace)
		if !candidate.allowed {
			continue
		}
		if _, ok := seen[candidate.entry.HostPath]; ok {
			continue
		}
		seen[candidate.entry.HostPath] = struct{}{}
		mounts = append(mounts, candidate.entry)
	}
	return mounts
}

type mountWorkspace struct {
	hostPath      string
	containerPath string
}

type mountCandidate struct {
	entry   MountEntry
	allowed bool
	blocked bool
}

func (p MountPolicy) classifyMountCandidate(hostPath string, workspace mountWorkspace) mountCandidate {
	cleanHost, ok := normalizeMountHostPath(hostPath)
	if !ok {
		return mountCandidate{}
	}
	if p.IsBlocked(cleanHost) {
		return mountCandidate{blocked: true}
	}
	readOnly := true
	containerPath := cleanHost
	if cleanHost == workspace.hostPath {
		readOnly = false
		containerPath = workspace.containerPath
	}
	return mountCandidate{
		entry: MountEntry{
			HostPath:      cleanHost,
			ContainerPath: containerPath,
			ReadOnly:      readOnly,
		},
		allowed: true,
	}
}

func containerWorkspacePath(containerWorkspace string) string {
	if containerWorkspace == "" {
		return defaultContainerWorkspace
	}
	return containerWorkspace
}
