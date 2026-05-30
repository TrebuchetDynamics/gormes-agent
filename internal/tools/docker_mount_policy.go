package tools

import "github.com/TrebuchetDynamics/gormes-agent/internal/tools/docker"

type MountPolicy = docker.MountPolicy
type MountEntry = docker.MountEntry

func DefaultMountPolicy(hostPaths []string) MountPolicy {
	return docker.DefaultMountPolicy(hostPaths)
}
