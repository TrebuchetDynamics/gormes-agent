package integration_test

import (
	. "github.com/TrebuchetDynamics/gormes-agent/internal/config"

	"os"
	"path/filepath"
	"testing"
)

func TestLoad_AgentsDefaultMain(t *testing.T) {
	home := filepath.Join(t.TempDir(), "gormes")
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("GORMES_HOME", home)

	cfg, err := Load(nil)
	if err != nil {
		t.Fatal(err)
	}
	if got := cfg.Agents.DefaultAgentID(); got != "main" {
		t.Fatalf("DefaultAgentID = %q, want main", got)
	}
	main, ok := cfg.Agents.AgentByID("main")
	if !ok {
		t.Fatal("default main agent missing")
	}
	if main.Workspace != filepath.Join(home, "workspace") || main.AgentDir != filepath.Join(home, "agents", "main", "agent") {
		t.Fatalf("main agent = %+v, want defaults under GORMES_HOME", main)
	}
}

func TestLoad_AgentsDefaultsOnlyOverrideMain(t *testing.T) {
	cfgHome := t.TempDir()
	home := filepath.Join(cfgHome, "gormes")
	t.Setenv("XDG_CONFIG_HOME", cfgHome)
	t.Setenv("GORMES_HOME", home)
	if err := os.MkdirAll(home, 0o755); err != nil {
		t.Fatalf("mkdir home: %v", err)
	}
	if err := os.WriteFile(filepath.Join(home, "config.toml"), []byte(`
[agents.defaults]
workspace = "/srv/gormes/workspaces/default"
agent_dir = "/srv/gormes/agents/default/agent"
skills = ["base"]
`), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := Load(nil)
	if err != nil {
		t.Fatal(err)
	}
	main, ok := cfg.Agents.AgentByID("main")
	if !ok {
		t.Fatal("default main agent missing")
	}
	if main.Workspace != "/srv/gormes/workspaces/default" || main.AgentDir != "/srv/gormes/agents/default/agent" || len(main.Skills) != 1 || main.Skills[0] != "base" {
		t.Fatalf("main agent = %+v, want defaults-only override", main)
	}
}

func TestLoad_AgentsAndBindingsFromTOML(t *testing.T) {
	cfgHome := t.TempDir()
	home := filepath.Join(cfgHome, "gormes")
	t.Setenv("XDG_CONFIG_HOME", cfgHome)
	t.Setenv("GORMES_HOME", home)
	if err := os.MkdirAll(home, 0o755); err != nil {
		t.Fatalf("mkdir home: %v", err)
	}
	if err := os.WriteFile(filepath.Join(home, "config.toml"), []byte(`
[agents.defaults]
workspace = "/srv/gormes/workspaces/main"
agent_dir = "/srv/gormes/agents/default/agent"
skills = ["base"]
workspaces = [" /srv/gormes/workspaces/project-a ", "", "/srv/gormes/workspaces/project-b"]

[[agents.list]]
id = "coding"
name = "Coding"
workspace = "/srv/gormes/workspaces/coding"
agent_dir = "/srv/gormes/agents/coding/agent"
default = true
model = "gpt-5.5"
skills = ["go", "web"]

[agents.list.tools]
allow = ["read", "write", "exec"]
deny = ["browser"]

[[agents.list]]
id = "family"
name = "Family"
workspace = "/srv/gormes/workspaces/family"

[[bindings]]
agent_id = "coding"
[bindings.match]
channel = "telegram"
account_id = "coding"
[bindings.match.peer]
kind = "direct"
id = "42"

[[bindings]]
agent_id = "family"
[bindings.match]
channel = "whatsapp"
[bindings.match.peer]
kind = "group"
id = "120363999@g.us"
`), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := Load(nil)
	if err != nil {
		t.Fatal(err)
	}
	if got := cfg.Agents.DefaultAgentID(); got != "coding" {
		t.Fatalf("DefaultAgentID = %q, want coding", got)
	}
	if got := cfg.Agents.Defaults.Workspaces; len(got) != 2 || got[0] != "/srv/gormes/workspaces/project-a" || got[1] != "/srv/gormes/workspaces/project-b" {
		t.Fatalf("workspaces = %#v, want cleaned list", got)
	}
	coding, ok := cfg.Agents.AgentByID("coding")
	if !ok {
		t.Fatal("coding agent missing")
	}
	if coding.Model != "gpt-5.5" || coding.Workspace != "/srv/gormes/workspaces/coding" || coding.AgentDir != "/srv/gormes/agents/coding/agent" || len(coding.Tools.Allow) != 3 || coding.Tools.Deny[0] != "browser" {
		t.Fatalf("coding agent = %+v", coding)
	}
	family, ok := cfg.Agents.AgentByID("family")
	if !ok {
		t.Fatal("family agent missing")
	}
	if family.AgentDir != filepath.Join(home, "agents", "family", "agent") {
		t.Fatalf("family agent_dir = %q, want default per-agent dir", family.AgentDir)
	}
	if len(cfg.Bindings) != 2 || cfg.Bindings[0].AgentID != "coding" || cfg.Bindings[0].Match.Peer.ID != "42" {
		t.Fatalf("bindings = %+v", cfg.Bindings)
	}
}
