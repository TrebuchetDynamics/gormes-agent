package identity

import (
	"errors"
	"fmt"
	"strings"
)

// DynamicAgentIdentityEvidence is the stable launch-evidence label emitted by
// the dynamic-agent identity resolver.
type DynamicAgentIdentityEvidence string

const (
	DynamicAgentIdentityResolved        DynamicAgentIdentityEvidence = "dynamic_agent_identity_resolved"
	DynamicAgentIdentityChildUnresolved DynamicAgentIdentityEvidence = "child_identity_unresolved"
)

// ErrChildIdentityUnresolved is returned when a delegated child identity is
// missing or too ambiguous to launch.
var ErrChildIdentityUnresolved = errors.New("subagent: child identity unresolved")

type dynamicAgentIdentityError struct {
	evidence DynamicAgentIdentityEvidence
	message  string
	err      error
}

func (e *dynamicAgentIdentityError) Error() string {
	if e == nil {
		return "subagent: dynamic identity error"
	}
	if e.message == "" {
		return e.err.Error()
	}
	return fmt.Sprintf("%s: %s", e.err, e.message)
}

func (e *dynamicAgentIdentityError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.err
}

// DynamicAgentIdentityErrorEvidence returns the stable evidence label carried
// by dynamic-agent identity errors.
func DynamicAgentIdentityErrorEvidence(err error) DynamicAgentIdentityEvidence {
	var identityErr *dynamicAgentIdentityError
	if errors.As(err, &identityErr) {
		return identityErr.evidence
	}
	return ""
}

// ParentAgentIdentity is lineage metadata. It is never rendered as the active
// child persona.
type ParentAgentIdentity struct {
	AgentID string
	Role    string
}

// ChildAgentIdentity is the resolved dynamic-agent identity selected for a
// child launch.
type ChildAgentIdentity struct {
	AgentID string
	Name    string
	Persona string
}

// ProjectContextScope identifies the AGENTS.md/project context attached to a
// parent or child workspace.
type ProjectContextScope struct {
	Workspace string
	AGENTS    string
}

// ToolPolicy captures the visible allow/deny/path-glob policy for a child
// launch. Empty means no explicit policy on that side of the launch.
type ToolPolicy struct {
	AllowedToolsets []string
	DeniedTools     []string
	PathGlobs       []string
}

// DynamicAgentIdentityRequest is a pure launch-plan input. ParentTurnData is
// accepted so callers can pass the live turn map for regression tests; the
// resolver treats it as read-only.
type DynamicAgentIdentityRequest struct {
	Parent ParentAgentIdentity
	Child  ChildAgentIdentity

	ParentProject ProjectContextScope
	ChildProject  ProjectContextScope

	ParentTurnData map[string]any

	ParentToolPolicy        ToolPolicy
	ChildToolPolicy         ToolPolicy
	InheritParentToolPolicy bool

	ParentMemoryFacts []string
	ChildMemoryFacts  []string
}

type ParentLineageEvidence struct {
	ParentAgentID string
	ParentRole    string
}

type ProjectScopeEvidence struct {
	ActiveWorkspace string
	ActiveAGENTS    string
	ParentWorkspace string
	ParentAGENTS    string
}

type ToolPolicyEvidence struct {
	Effective           ToolPolicy
	InheritedFromParent bool
	Evidence            string
}

type MemoryScopeEvidence struct {
	AgentID          string
	ParentAgentID    string
	SearchNamespaces []string
	VisibleFacts     []string
}

// DynamicAgentIdentityPlan is the safe child launch identity and scope
// evidence. ActiveIdentity is child-owned persona text; parent identity remains
// in ParentLineage only.
type DynamicAgentIdentityPlan struct {
	Evidence       DynamicAgentIdentityEvidence
	Active         ChildAgentIdentity
	ActiveIdentity string
	ParentLineage  ParentLineageEvidence
	ProjectContext ProjectScopeEvidence
	ToolPolicy     ToolPolicyEvidence
	MemoryScope    MemoryScopeEvidence
}

// ResolveDynamicAgentIdentity builds the child launch plan without mutating
// the parent turn. It intentionally performs no process/provider IO.
func ResolveDynamicAgentIdentity(req DynamicAgentIdentityRequest) (DynamicAgentIdentityPlan, error) {
	child := normalizeChildIdentity(req.Child)
	if child.AgentID == "" {
		return DynamicAgentIdentityPlan{}, &dynamicAgentIdentityError{
			evidence: DynamicAgentIdentityChildUnresolved,
			message:  "child agent id is required",
			err:      ErrChildIdentityUnresolved,
		}
	}
	if child.Name == "" && child.Persona == "" {
		return DynamicAgentIdentityPlan{}, &dynamicAgentIdentityError{
			evidence: DynamicAgentIdentityChildUnresolved,
			message:  "child agent name or persona is required",
			err:      ErrChildIdentityUnresolved,
		}
	}

	parent := ParentLineageEvidence{
		ParentAgentID: strings.TrimSpace(req.Parent.AgentID),
		ParentRole:    strings.TrimSpace(req.Parent.Role),
	}
	return DynamicAgentIdentityPlan{
		Evidence:       DynamicAgentIdentityResolved,
		Active:         child,
		ActiveIdentity: activeChildIdentityText(child),
		ParentLineage:  parent,
		ProjectContext: ProjectScopeEvidence{
			ActiveWorkspace: strings.TrimSpace(req.ChildProject.Workspace),
			ActiveAGENTS:    strings.TrimSpace(req.ChildProject.AGENTS),
			ParentWorkspace: strings.TrimSpace(req.ParentProject.Workspace),
			ParentAGENTS:    strings.TrimSpace(req.ParentProject.AGENTS),
		},
		ToolPolicy:  resolveDynamicAgentToolPolicy(req),
		MemoryScope: resolveDynamicAgentMemoryScope(child.AgentID, parent.ParentAgentID, req.ChildMemoryFacts),
	}, nil
}

func normalizeChildIdentity(child ChildAgentIdentity) ChildAgentIdentity {
	return ChildAgentIdentity{
		AgentID: strings.TrimSpace(child.AgentID),
		Name:    strings.TrimSpace(child.Name),
		Persona: strings.TrimSpace(child.Persona),
	}
}

func activeChildIdentityText(child ChildAgentIdentity) string {
	if child.Persona != "" {
		return child.Persona
	}
	if child.Name != "" {
		return "Dynamic child agent: " + child.Name
	}
	return "Dynamic child agent: " + child.AgentID
}

func resolveDynamicAgentToolPolicy(req DynamicAgentIdentityRequest) ToolPolicyEvidence {
	if req.InheritParentToolPolicy {
		return ToolPolicyEvidence{
			Effective:           mergeToolPolicies(req.ParentToolPolicy, req.ChildToolPolicy),
			InheritedFromParent: true,
			Evidence:            "explicit_parent_tool_policy_inheritance",
		}
	}
	return ToolPolicyEvidence{
		Effective:           normalizeToolPolicy(req.ChildToolPolicy),
		InheritedFromParent: false,
		Evidence:            "child_tool_policy_only",
	}
}

func mergeToolPolicies(parent, child ToolPolicy) ToolPolicy {
	parent = normalizeToolPolicy(parent)
	child = normalizeToolPolicy(child)
	return ToolPolicy{
		AllowedToolsets: appendUniqueStrings(parent.AllowedToolsets, child.AllowedToolsets...),
		DeniedTools:     appendUniqueStrings(parent.DeniedTools, child.DeniedTools...),
		PathGlobs:       appendUniqueStrings(parent.PathGlobs, child.PathGlobs...),
	}
}

func normalizeToolPolicy(policy ToolPolicy) ToolPolicy {
	return ToolPolicy{
		AllowedToolsets: uniqueTrimmedStrings(policy.AllowedToolsets),
		DeniedTools:     uniqueTrimmedStrings(policy.DeniedTools),
		PathGlobs:       uniqueTrimmedStrings(policy.PathGlobs),
	}
}

func resolveDynamicAgentMemoryScope(childID, parentID string, childFacts []string) MemoryScopeEvidence {
	var namespaces []string
	if childID != "" {
		namespaces = append(namespaces, "agent:"+childID)
	}
	if parentID != "" {
		namespaces = append(namespaces, "lineage:"+parentID)
	}
	return MemoryScopeEvidence{
		AgentID:          childID,
		ParentAgentID:    parentID,
		SearchNamespaces: namespaces,
		VisibleFacts:     uniqueTrimmedStrings(childFacts),
	}
}

func uniqueTrimmedStrings(values []string) []string {
	var out []string
	seen := map[string]struct{}{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}

func appendUniqueStrings(base []string, values ...string) []string {
	out := append([]string(nil), base...)
	seen := map[string]struct{}{}
	for _, value := range out {
		seen[value] = struct{}{}
	}
	for _, value := range values {
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}
