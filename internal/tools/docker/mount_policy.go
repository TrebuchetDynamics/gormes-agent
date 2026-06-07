package docker

import (
	"github.com/TrebuchetDynamics/gormes-agent/internal/tools/docker/contract"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tools/docker/policy"
)

// MountPolicy defines allowed and blocked host paths for Docker mounts.
type MountPolicy = policy.MountPolicy

// MountEntry describes a resolved mount from host to container.
type MountEntry = contract.MountEntry

// DefaultMountPolicy returns a policy that blocks dangerous system paths
// and Docker socket access while allowing configured host paths.
func DefaultMountPolicy(hostPaths []string) MountPolicy {
	return policy.DefaultMountPolicy(hostPaths)
}
