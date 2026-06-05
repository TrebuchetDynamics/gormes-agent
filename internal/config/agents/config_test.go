package agents

import (
	"path/filepath"
	"reflect"
	"testing"
)

func TestDefaultConfigCreatesMainAgentUnderHome(t *testing.T) {
	home := filepath.Join(t.TempDir(), "gormes")
	cfg := DefaultConfig(home)

	if got := cfg.DefaultAgentID(); got != DefaultAgentID {
		t.Fatalf("DefaultAgentID = %q, want %s", got, DefaultAgentID)
	}
	main, ok := cfg.AgentByID(DefaultAgentID)
	if !ok {
		t.Fatal("default main agent missing")
	}
	if main.Workspace != filepath.Join(home, "workspace") {
		t.Fatalf("main workspace = %q, want default workspace under home", main.Workspace)
	}
	if main.AgentDir != filepath.Join(home, "agents", "main", "agent") {
		t.Fatalf("main agent_dir = %q, want per-agent state under home", main.AgentDir)
	}
}

func TestNormalizeConfigDefaultsOnlyOverrideMain(t *testing.T) {
	home := filepath.Join(t.TempDir(), "gormes")
	cfg := AgentsCfg{Defaults: AgentDefaultsCfg{
		Workspace: " /srv/gormes/workspaces/default ",
		AgentDir:  " /srv/gormes/agents/default/agent ",
		Skills:    []string{" base ", ""},
	}}

	if err := NormalizeConfig(home, &cfg, nil); err != nil {
		t.Fatalf("NormalizeConfig: %v", err)
	}
	main, ok := cfg.AgentByID("main")
	if !ok {
		t.Fatal("default main agent missing")
	}
	if main.Workspace != "/srv/gormes/workspaces/default" || main.AgentDir != "/srv/gormes/agents/default/agent" {
		t.Fatalf("main agent = %+v, want defaults-only override", main)
	}
	if !reflect.DeepEqual(main.Skills, []string{"base"}) {
		t.Fatalf("main skills = %#v, want defaults-only skills", main.Skills)
	}
}

func TestNormalizeConfigCleansDefaultsWorkspaceList(t *testing.T) {
	cfg := AgentsCfg{Defaults: AgentDefaultsCfg{Workspaces: []string{" /srv/gormes/workspaces/project-a ", "", "/srv/gormes/workspaces/project-b"}}}

	if err := NormalizeConfig(t.TempDir(), &cfg, nil); err != nil {
		t.Fatalf("NormalizeConfig: %v", err)
	}
	want := []string{"/srv/gormes/workspaces/project-a", "/srv/gormes/workspaces/project-b"}
	if !reflect.DeepEqual(cfg.Defaults.Workspaces, want) {
		t.Fatalf("workspaces = %#v, want %#v", cfg.Defaults.Workspaces, want)
	}
}

func TestNormalizeConfigAgentsAndBindings(t *testing.T) {
	home := filepath.Join(t.TempDir(), "gormes")
	cfg := AgentsCfg{
		Defaults: AgentDefaultsCfg{Skills: []string{"base"}},
		List: []AgentCfg{
			{ID: " Coding ", Name: " Coding ", Workspace: "/srv/gormes/workspaces/coding", AgentDir: "/srv/gormes/agents/coding/agent", Default: true, Model: " gpt-5.5 ", Skills: []string{" go ", "web"}, Tools: AgentToolPolicy{Allow: []string{"read", "write", "exec"}, Deny: []string{"browser"}}},
			{ID: "family", Name: "Family", Workspace: "/srv/gormes/workspaces/family"},
		},
	}
	bindings := []AgentBindingCfg{
		{AgentID: "coding", Match: AgentBindingMatchCfg{Channel: "telegram", Peer: AgentPeerMatchCfg{Kind: "direct", ID: "42"}}},
		{AgentID: "family", Match: AgentBindingMatchCfg{Channel: "whatsapp", Peer: AgentPeerMatchCfg{Kind: "group", ID: "120363999@g.us"}}},
	}

	if err := NormalizeConfig(home, &cfg, bindings); err != nil {
		t.Fatalf("NormalizeConfig: %v", err)
	}
	if got := cfg.DefaultAgentID(); got != "coding" {
		t.Fatalf("DefaultAgentID = %q, want coding", got)
	}
	coding, ok := cfg.AgentByID("coding")
	if !ok {
		t.Fatal("coding agent missing")
	}
	if coding.Model != "gpt-5.5" || coding.Workspace != "/srv/gormes/workspaces/coding" || coding.AgentDir != "/srv/gormes/agents/coding/agent" {
		t.Fatalf("coding agent = %+v", coding)
	}
	family, ok := cfg.AgentByID("family")
	if !ok {
		t.Fatal("family agent missing")
	}
	if family.AgentDir != filepath.Join(home, "agents", "family", "agent") {
		t.Fatalf("family agent_dir = %q, want default per-agent dir", family.AgentDir)
	}

	normalized := NormalizeBindings(bindings)
	if normalized[0].AgentID != "coding" || normalized[0].Match.Channel != "telegram" || normalized[0].Match.Peer.ID != "42" {
		t.Fatalf("first binding = %+v", normalized[0])
	}
}
