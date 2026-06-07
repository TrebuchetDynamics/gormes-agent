package identity

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"

	goncho "github.com/TrebuchetDynamics/goncho/dynamicagents"
	"github.com/TrebuchetDynamics/gormes-agent/internal/core/agent"
	"github.com/TrebuchetDynamics/gormes-agent/internal/memory"
)

func TestDynamicAgentIdentity_ChildPersonaOverridesRootRole(t *testing.T) {
	child := createDynamicIdentityTestAgent(t, "Release Triage", "You are the release triage child agent.")
	plan, err := ResolveDynamicAgentIdentity(DynamicAgentIdentityRequest{
		Parent: ParentAgentIdentity{
			AgentID: "root-agent",
			Role:    "You are the root operator agent.",
		},
		Child: ChildAgentIdentity{
			AgentID: child.ID,
			Name:    child.Name,
			Persona: child.Persona,
		},
	})
	if err != nil {
		t.Fatalf("ResolveDynamicAgentIdentity: %v", err)
	}

	if plan.Active.AgentID != child.ID {
		t.Fatalf("active agent id = %q, want %q", plan.Active.AgentID, child.ID)
	}
	if plan.ActiveIdentity != child.Persona {
		t.Fatalf("active identity = %q, want child persona %q", plan.ActiveIdentity, child.Persona)
	}
	if strings.Contains(plan.ActiveIdentity, "root operator") {
		t.Fatalf("active identity leaked parent role: %q", plan.ActiveIdentity)
	}
	if plan.ParentLineage.ParentAgentID != "root-agent" || plan.ParentLineage.ParentRole != "You are the root operator agent." {
		t.Fatalf("parent lineage = %+v, want root metadata preserved separately", plan.ParentLineage)
	}
	if plan.Evidence != DynamicAgentIdentityResolved {
		t.Fatalf("evidence = %q, want %q", plan.Evidence, DynamicAgentIdentityResolved)
	}
}

func TestDynamicAgentIdentity_AGENTSScopesDoNotBleedAcrossChild(t *testing.T) {
	child := createDynamicIdentityTestAgent(t, "Docs Child", "You work only inside the docs workspace.")
	parentTurn := &agent.MiddlewareContext{
		ThreadID: "parent-thread",
		Data: map[string]any{
			"agents_context": "parent AGENTS.md: root rules",
			"workspace":      "/repo/root",
		},
	}

	plan, err := ResolveDynamicAgentIdentity(DynamicAgentIdentityRequest{
		Parent: ParentAgentIdentity{AgentID: "root-agent", Role: "root role"},
		Child:  ChildAgentIdentity{AgentID: child.ID, Name: child.Name, Persona: child.Persona},
		ParentProject: ProjectContextScope{
			Workspace: "/repo/root",
			AGENTS:    "parent AGENTS.md: root rules",
		},
		ChildProject: ProjectContextScope{
			Workspace: "/repo/docs",
			AGENTS:    "child AGENTS.md: docs rules",
		},
		ParentTurnData: parentTurn.Data,
	})
	if err != nil {
		t.Fatalf("ResolveDynamicAgentIdentity: %v", err)
	}

	if plan.ProjectContext.ActiveWorkspace != "/repo/docs" || plan.ProjectContext.ActiveAGENTS != "child AGENTS.md: docs rules" {
		t.Fatalf("active project context = %+v, want child workspace AGENTS scope", plan.ProjectContext)
	}
	if strings.Contains(plan.ProjectContext.ActiveAGENTS, "root rules") {
		t.Fatalf("active AGENTS context leaked parent rules: %q", plan.ProjectContext.ActiveAGENTS)
	}
	if parentTurn.Data["agents_context"] != "parent AGENTS.md: root rules" || parentTurn.Data["workspace"] != "/repo/root" {
		t.Fatalf("parent turn data mutated: %+v", parentTurn.Data)
	}
}

func TestDynamicAgentIdentity_ToolPolicyInheritedOnlyWhenExplicit(t *testing.T) {
	base := DynamicAgentIdentityRequest{
		Parent: ParentAgentIdentity{AgentID: "root-agent", Role: "root"},
		Child:  ChildAgentIdentity{AgentID: "reviewer", Name: "Reviewer", Persona: "Review only."},
		ParentToolPolicy: ToolPolicy{
			AllowedToolsets: []string{"memory"},
			DeniedTools:     []string{"terminal"},
			PathGlobs:       []string{"src/**"},
		},
		ChildToolPolicy: ToolPolicy{
			AllowedToolsets: []string{"skills"},
			DeniedTools:     []string{"write_file"},
		},
	}

	childOnly, err := ResolveDynamicAgentIdentity(base)
	if err != nil {
		t.Fatalf("ResolveDynamicAgentIdentity child-only: %v", err)
	}
	if childOnly.ToolPolicy.InheritedFromParent {
		t.Fatalf("InheritedFromParent = true without explicit opt-in")
	}
	if !reflect.DeepEqual(childOnly.ToolPolicy.Effective.AllowedToolsets, []string{"skills"}) {
		t.Fatalf("child-only allowed toolsets = %+v, want [skills]", childOnly.ToolPolicy.Effective.AllowedToolsets)
	}
	if containsString(childOnly.ToolPolicy.Effective.DeniedTools, "terminal") {
		t.Fatalf("child-only policy inherited parent denied tool terminal: %+v", childOnly.ToolPolicy.Effective)
	}

	base.InheritParentToolPolicy = true
	inherited, err := ResolveDynamicAgentIdentity(base)
	if err != nil {
		t.Fatalf("ResolveDynamicAgentIdentity inherited: %v", err)
	}
	if !inherited.ToolPolicy.InheritedFromParent {
		t.Fatalf("InheritedFromParent = false with explicit opt-in")
	}
	for _, want := range []string{"memory", "skills"} {
		if !containsString(inherited.ToolPolicy.Effective.AllowedToolsets, want) {
			t.Fatalf("inherited allowed toolsets missing %q: %+v", want, inherited.ToolPolicy.Effective.AllowedToolsets)
		}
	}
	for _, want := range []string{"terminal", "write_file"} {
		if !containsString(inherited.ToolPolicy.Effective.DeniedTools, want) {
			t.Fatalf("inherited denied tools missing %q: %+v", want, inherited.ToolPolicy.Effective.DeniedTools)
		}
	}
	if !containsString(inherited.ToolPolicy.Effective.PathGlobs, "src/**") {
		t.Fatalf("inherited path globs = %+v, want src/**", inherited.ToolPolicy.Effective.PathGlobs)
	}
	if !strings.Contains(inherited.ToolPolicy.Evidence, "explicit") {
		t.Fatalf("tool-policy evidence = %q, want explicit inheritance evidence", inherited.ToolPolicy.Evidence)
	}
}

func TestDynamicAgentIdentity_InheritedToolPolicyDeduplicatesAndTrims(t *testing.T) {
	plan, err := ResolveDynamicAgentIdentity(DynamicAgentIdentityRequest{
		Parent: ParentAgentIdentity{AgentID: "root-agent", Role: "root"},
		Child:  ChildAgentIdentity{AgentID: "reviewer", Name: "Reviewer"},
		ParentToolPolicy: ToolPolicy{
			AllowedToolsets: []string{" memory ", "skills"},
			DeniedTools:     []string{" terminal ", "terminal"},
			PathGlobs:       []string{" src/** "},
		},
		ChildToolPolicy: ToolPolicy{
			AllowedToolsets: []string{"skills", " browser "},
			DeniedTools:     []string{"write_file", " terminal "},
			PathGlobs:       []string{"src/**", " docs/** "},
		},
		InheritParentToolPolicy: true,
	})
	if err != nil {
		t.Fatalf("ResolveDynamicAgentIdentity: %v", err)
	}
	if !reflect.DeepEqual(plan.ToolPolicy.Effective.AllowedToolsets, []string{"memory", "skills", "browser"}) {
		t.Fatalf("AllowedToolsets = %+v, want [memory skills browser]", plan.ToolPolicy.Effective.AllowedToolsets)
	}
	if !reflect.DeepEqual(plan.ToolPolicy.Effective.DeniedTools, []string{"terminal", "write_file"}) {
		t.Fatalf("DeniedTools = %+v, want [terminal write_file]", plan.ToolPolicy.Effective.DeniedTools)
	}
	if !reflect.DeepEqual(plan.ToolPolicy.Effective.PathGlobs, []string{"src/**", "docs/**"}) {
		t.Fatalf("PathGlobs = %+v, want [src/** docs/**]", plan.ToolPolicy.Effective.PathGlobs)
	}
}

func TestDynamicAgentIdentity_MemoryScopeIsChildAware(t *testing.T) {
	plan, err := ResolveDynamicAgentIdentity(DynamicAgentIdentityRequest{
		Parent: ParentAgentIdentity{AgentID: "root-agent", Role: "root"},
		Child:  ChildAgentIdentity{AgentID: "research-child", Name: "Research Child", Persona: "Research."},
		ParentMemoryFacts: []string{
			"root-only deployment secret",
		},
		ChildMemoryFacts: []string{
			"child only watches MCP protocol notes",
		},
	})
	if err != nil {
		t.Fatalf("ResolveDynamicAgentIdentity: %v", err)
	}

	if plan.MemoryScope.AgentID != "research-child" || plan.MemoryScope.ParentAgentID != "root-agent" {
		t.Fatalf("memory scope lineage = %+v, want child and parent IDs", plan.MemoryScope)
	}
	for _, want := range []string{"agent:research-child", "lineage:root-agent"} {
		if !containsString(plan.MemoryScope.SearchNamespaces, want) {
			t.Fatalf("search namespaces missing %q: %+v", want, plan.MemoryScope.SearchNamespaces)
		}
	}
	if !reflect.DeepEqual(plan.MemoryScope.VisibleFacts, []string{"child only watches MCP protocol notes"}) {
		t.Fatalf("visible memory facts = %+v, want only child facts", plan.MemoryScope.VisibleFacts)
	}
	for _, fact := range plan.MemoryScope.VisibleFacts {
		if strings.Contains(fact, "root-only") {
			t.Fatalf("child memory scope merged unrelated root fact: %+v", plan.MemoryScope.VisibleFacts)
		}
	}
}

func TestDynamicAgentIdentity_UnresolvedChildRefusesLaunch(t *testing.T) {
	parentTurn := &agent.MiddlewareContext{
		Data: map[string]any{"agents_context": "parent context remains"},
	}
	_, err := ResolveDynamicAgentIdentity(DynamicAgentIdentityRequest{
		Parent:         ParentAgentIdentity{AgentID: "root-agent", Role: "root"},
		ParentTurnData: parentTurn.Data,
	})
	if !errors.Is(err, ErrChildIdentityUnresolved) {
		t.Fatalf("ResolveDynamicAgentIdentity err = %v, want ErrChildIdentityUnresolved", err)
	}
	if DynamicAgentIdentityErrorEvidence(err) != DynamicAgentIdentityChildUnresolved {
		t.Fatalf("error evidence = %q, want %q", DynamicAgentIdentityErrorEvidence(err), DynamicAgentIdentityChildUnresolved)
	}
	if parentTurn.Data["agents_context"] != "parent context remains" {
		t.Fatalf("parent turn data mutated after refused launch: %+v", parentTurn.Data)
	}
}

func createDynamicIdentityTestAgent(t *testing.T, name, persona string) goncho.AgentRecord {
	t.Helper()
	store, err := memory.OpenSqlite(t.TempDir()+"/dynamic-identity.db", 0, nil)
	if err != nil {
		t.Fatalf("OpenSqlite: %v", err)
	}
	t.Cleanup(func() { _ = store.Close(context.Background()) })

	reg, err := goncho.NewDynamicAgentRegistry(store.DB())
	if err != nil {
		t.Fatalf("NewDynamicAgentRegistry: %v", err)
	}
	rec, err := reg.Create(context.Background(), goncho.CreateAgentOptions{Name: name, Persona: persona})
	if err != nil {
		t.Fatalf("Create dynamic agent: %v", err)
	}
	return rec
}

func containsString(values []string, want string) bool {
	for _, got := range values {
		if got == want {
			return true
		}
	}
	return false
}
