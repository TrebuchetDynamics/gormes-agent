package policy

import (
	"path/filepath"
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/tools/docker/contract"
)

const defaultContainerWorkspace = "/workspace"

// MountPolicy defines allowed and blocked host paths for Docker mounts.
type MountPolicy struct {
	AllowedHostPaths []string
	BlockedPrefixes  []string
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
func (p MountPolicy) AllowedMounts(workspacePath, containerWorkspace string) []contract.MountEntry {
	var mounts []contract.MountEntry
	seen := map[string]struct{}{}
	workspaceHost, _ := normalizeMountHostPath(workspacePath)
	workspace := mountWorkspace{hostPath: workspaceHost, containerPath: ContainerWorkspacePath(containerWorkspace)}
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
	entry   contract.MountEntry
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
		entry: contract.MountEntry{
			HostPath:      cleanHost,
			ContainerPath: containerPath,
			ReadOnly:      readOnly,
		},
		allowed: true,
	}
}

func ContainerWorkspacePath(containerWorkspace string) string {
	clean := filepath.Clean(strings.TrimSpace(containerWorkspace))
	if clean == "." || !strings.HasPrefix(clean, "/") {
		return defaultContainerWorkspace
	}
	return clean
}
