package subagent

import "github.com/TrebuchetDynamics/gormes-agent/internal/core/subagent/identity"

// DynamicAgentIdentityEvidence is the stable launch-evidence label emitted by
// the dynamic-agent identity resolver.
type DynamicAgentIdentityEvidence = identity.DynamicAgentIdentityEvidence

const (
	DynamicAgentIdentityResolved        = identity.DynamicAgentIdentityResolved
	DynamicAgentIdentityChildUnresolved = identity.DynamicAgentIdentityChildUnresolved
)

// ErrChildIdentityUnresolved is returned when a delegated child identity is
// missing or too ambiguous to launch.
var ErrChildIdentityUnresolved = identity.ErrChildIdentityUnresolved

// DynamicAgentIdentityErrorEvidence returns the stable evidence label carried
// by dynamic-agent identity errors.
func DynamicAgentIdentityErrorEvidence(err error) DynamicAgentIdentityEvidence {
	return identity.DynamicAgentIdentityErrorEvidence(err)
}

// ParentAgentIdentity is lineage metadata. It is never rendered as the active
// child persona.
type ParentAgentIdentity = identity.ParentAgentIdentity

// ChildAgentIdentity is the resolved dynamic-agent identity selected for a
// child launch.
type ChildAgentIdentity = identity.ChildAgentIdentity

// ProjectContextScope identifies the AGENTS.md/project context attached to a
// parent or child workspace.
type ProjectContextScope = identity.ProjectContextScope

// ToolPolicy captures the visible allow/deny/path-glob policy for a child
// launch. Empty means no explicit policy on that side of the launch.
type ToolPolicy = identity.ToolPolicy

// DynamicAgentIdentityRequest is a pure launch-plan input. ParentTurnData is
// accepted so callers can pass the live turn map for regression tests; the
// resolver treats it as read-only.
type DynamicAgentIdentityRequest = identity.DynamicAgentIdentityRequest

type ParentLineageEvidence = identity.ParentLineageEvidence

type ProjectScopeEvidence = identity.ProjectScopeEvidence

type ToolPolicyEvidence = identity.ToolPolicyEvidence

type MemoryScopeEvidence = identity.MemoryScopeEvidence

// DynamicAgentIdentityPlan is the safe child launch identity and scope
// evidence. ActiveIdentity is child-owned persona text; parent identity remains
// in ParentLineage only.
type DynamicAgentIdentityPlan = identity.DynamicAgentIdentityPlan

// ResolveDynamicAgentIdentity builds the child launch plan without mutating
// the parent turn. It intentionally performs no process/provider IO.
func ResolveDynamicAgentIdentity(req DynamicAgentIdentityRequest) (DynamicAgentIdentityPlan, error) {
	return identity.ResolveDynamicAgentIdentity(req)
}
