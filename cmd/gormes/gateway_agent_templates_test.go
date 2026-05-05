package main

import (
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/agenttemplate"
	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/hermes"
)

func TestGatewayAgentTemplateTargetPrefersConfiguredTerminalWorkspace(t *testing.T) {
	root := t.TempDir()
	workspace := filepath.Join(root, "workspace-gormes")
	agentWorkspace := filepath.Join(root, "agent-workspace")
	gormesHome := filepath.Join(root, "home")
	t.Setenv("GORMES_HOME", gormesHome)

	got := gatewayAgentTemplateTarget(config.Config{
		Terminal: config.TerminalCfg{CWD: workspace},
		Agents: config.AgentsCfg{List: []config.AgentCfg{{
			ID:        config.DefaultAgentID,
			Workspace: agentWorkspace,
			Default:   true,
		}}},
	})
	if got != workspace {
		t.Fatalf("gatewayAgentTemplateTarget = %q, want terminal cwd %q", got, workspace)
	}
}

func TestGatewayAgentTemplateTargetFallsBackToDefaultAgentWorkspace(t *testing.T) {
	root := t.TempDir()
	workspace := filepath.Join(root, "workspace-main")
	t.Setenv("GORMES_HOME", filepath.Join(root, "home"))

	got := gatewayAgentTemplateTarget(config.Config{
		Agents: config.AgentsCfg{List: []config.AgentCfg{{
			ID:        config.DefaultAgentID,
			Workspace: workspace,
			Default:   true,
		}}},
	})
	if got != workspace {
		t.Fatalf("gatewayAgentTemplateTarget = %q, want default agent workspace %q", got, workspace)
	}
}

func TestEnsureGatewayAgentTemplatesCreatesMissingRuntimeContextAndPromptLoadsIt(t *testing.T) {
	root := t.TempDir()
	workspace := filepath.Join(root, "workspace-gormes")
	gormesHome := filepath.Join(root, "gormes-home")
	t.Setenv("GORMES_HOME", gormesHome)

	result, err := ensureGatewayAgentTemplates(config.Config{
		Terminal: config.TerminalCfg{CWD: workspace},
	}, discardLogger())
	if err != nil {
		t.Fatalf("ensureGatewayAgentTemplates: %v", err)
	}
	actions := gatewayTemplateActionsByPath(result)
	for _, path := range []string{
		"SOUL.md",
		"AGENTS.md",
		"IDENTITY.md",
		"TOOLS.md",
		filepath.Join("memory", "USER.md"),
		filepath.Join("memory", "MEMORY.md"),
	} {
		if actions[path] != agenttemplate.ActionCreate {
			t.Fatalf("%s action = %q, want create; all actions=%v", path, actions[path], actions)
		}
		if _, err := os.Stat(filepath.Join(workspace, path)); err != nil {
			t.Fatalf("template %s was not created in workspace: %v", path, err)
		}
	}
	if _, err := os.Stat(filepath.Join(gormesHome, "SOUL.md")); !os.IsNotExist(err) {
		t.Fatalf("runtime seeding should target the configured workspace, not GORMES_HOME; stat err=%v", err)
	}

	contextBlock, _ := hermes.BuildContextFilesPrompt(hermes.ContextFilesOptions{
		ProfileDir: workspace,
		CWD:        workspace,
	})
	for _, want := range []string{
		"# Gormes Agent Persona",
		"## AGENTS.md",
		"## IDENTITY.md",
		"## TOOLS.md",
	} {
		if !strings.Contains(contextBlock, want) {
			t.Fatalf("seeded context block missing %q:\n%s", want, contextBlock)
		}
	}
	durableBlock, _ := hermes.BuildDurableUserContextPrompt(hermes.DurableUserContextOptions{
		MemoryDir: filepath.Join(workspace, "memory"),
	})
	for _, want := range []string{"# User", "# Memory"} {
		if !strings.Contains(durableBlock, want) {
			t.Fatalf("seeded durable block missing %q:\n%s", want, durableBlock)
		}
	}
}

func TestEnsureGatewayAgentTemplatesPreservesExistingFiles(t *testing.T) {
	workspace := t.TempDir()
	if err := os.WriteFile(filepath.Join(workspace, "SOUL.md"), []byte("custom soul\n"), 0o600); err != nil {
		t.Fatalf("seed custom SOUL.md: %v", err)
	}

	result, err := ensureGatewayAgentTemplates(config.Config{
		Terminal: config.TerminalCfg{CWD: workspace},
	}, discardLogger())
	if err != nil {
		t.Fatalf("ensureGatewayAgentTemplates: %v", err)
	}
	if got := gatewayTemplateActionsByPath(result)["SOUL.md"]; got != agenttemplate.ActionSkip {
		t.Fatalf("SOUL.md action = %q, want skip", got)
	}
	body, err := os.ReadFile(filepath.Join(workspace, "SOUL.md"))
	if err != nil {
		t.Fatalf("read SOUL.md: %v", err)
	}
	if string(body) != "custom soul\n" {
		t.Fatalf("existing SOUL.md was overwritten:\n%s", body)
	}
}

func gatewayTemplateActionsByPath(result agenttemplate.WriteResult) map[string]agenttemplate.Action {
	out := make(map[string]agenttemplate.Action, len(result.Files))
	for _, file := range result.Files {
		out[file.Path] = file.Action
	}
	return out
}

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}
