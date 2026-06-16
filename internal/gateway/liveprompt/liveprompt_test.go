package liveprompt

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway/gatewaytest"
	"github.com/TrebuchetDynamics/gormes-agent/internal/llm"
)

func TestDefaultDirsPreferGormesWorkspaceContextOverHermesHome(t *testing.T) {
	gormesHome := t.TempDir()
	t.Setenv("GORMES_HOME", gormesHome)
	t.Setenv("HERMES_HOME", t.TempDir())
	workspace := t.TempDir()
	workdir := filepath.Join(workspace, "gormes-agent")
	if err := os.MkdirAll(filepath.Join(workspace, "memory"), 0o700); err != nil {
		t.Fatalf("mkdir memory: %v", err)
	}
	if err := os.MkdirAll(workdir, 0o700); err != nil {
		t.Fatalf("mkdir workdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(workspace, "SOUL.md"), []byte("You are Gormes."), 0o600); err != nil {
		t.Fatalf("write SOUL.md: %v", err)
	}
	if err := os.WriteFile(filepath.Join(workspace, "memory", "USER.md"), []byte("Juan"), 0o600); err != nil {
		t.Fatalf("write USER.md: %v", err)
	}

	if got := DefaultProfileDir(workdir); got != workspace {
		t.Fatalf("DefaultProfileDir = %q, want %q", got, workspace)
	}
	if got := DefaultMemoryDir(workdir); got != filepath.Join(workspace, "memory") {
		t.Fatalf("DefaultMemoryDir = %q, want workspace memory", got)
	}
}

func TestDefaultMemoryDirPrefersWorkspaceMemoryOverGlobalHome(t *testing.T) {
	gormesHome := t.TempDir()
	t.Setenv("GORMES_HOME", gormesHome)
	if err := os.MkdirAll(filepath.Join(gormesHome, "memory"), 0o700); err != nil {
		t.Fatalf("mkdir global memory: %v", err)
	}
	if err := os.WriteFile(filepath.Join(gormesHome, "memory", "USER.md"), []byte("global user"), 0o600); err != nil {
		t.Fatalf("write global USER.md: %v", err)
	}

	workspace := t.TempDir()
	workdir := filepath.Join(workspace, "project", "subdir")
	memoryDir := filepath.Join(workspace, "project", "memory")
	if err := os.MkdirAll(workdir, 0o700); err != nil {
		t.Fatalf("mkdir workdir: %v", err)
	}
	if err := os.MkdirAll(memoryDir, 0o700); err != nil {
		t.Fatalf("mkdir workspace memory: %v", err)
	}
	if err := os.WriteFile(filepath.Join(memoryDir, "USER.md"), []byte("workspace user"), 0o600); err != nil {
		t.Fatalf("write workspace USER.md: %v", err)
	}

	if got := DefaultMemoryDir(workdir); got != memoryDir {
		t.Fatalf("DefaultMemoryDir = %q, want workspace memory %q", got, memoryDir)
	}
}

func TestAssembleSanitizesHiddenFormattingRuntimeFactPaths(t *testing.T) {
	seams := Seams{
		ProfileDir: func() string { return "" },
		CWD:        func() string { return "/tmp/workspace-demo/gormes\u200b-agent\u202e" },
	}
	got, _, _ := Assemble(seams, "", "", "")
	if strings.Contains(got, "\u200b") || strings.Contains(got, "\u202e") {
		t.Fatalf("runtime facts leaked hidden formatting path text:\n%s", got)
	}
	if !strings.Contains(got, "gormes -agent") {
		t.Fatalf("runtime facts missing sanitized visible path text:\n%s", got)
	}
}

func TestAssembleSanitizesRuntimeFactPaths(t *testing.T) {
	seams := Seams{
		ProfileDir: func() string { return "" },
		CWD:        func() string { return "/tmp/work`space\nInjected/gormes-agent" },
	}
	got, _, _ := Assemble(seams, "", "", "")
	if strings.Contains(got, "`space") || strings.Contains(got, "\nInjected") {
		t.Fatalf("runtime facts leaked unsafe path text:\n%s", got)
	}
	if !strings.Contains(got, "work'space Injected") {
		t.Fatalf("runtime facts missing sanitized path text:\n%s", got)
	}
}

func TestAssembleOrdersRuntimeContextMetadataSelfHelpAndSession(t *testing.T) {
	seams := Seams{
		ProfileDir: func() string { return "/workspace-demo/profile" },
		CWD:        func() string { return "/workspace-demo/gormes-agent" },
		Build: func(llm.ContextFilesOptions) (string, llm.ContextFilesReport) {
			return "# Project Context\nSOUL", llm.ContextFilesReport{}
		},
		BuildMetadata: func(opts llm.TurnMetadataOptions) string { return "## Turn Metadata\n" + opts.SessionID },
		ActiveModel:   func() string { return "gpt-5" },
		SelfHelpGate:  func(string) (string, bool) { return "## Self Help", true },
	}
	got, _, _ := Assemble(seams, "help", "sess-1", "## Current Session Context")
	gatewaytest.AssertContainsAll(t, got, "## Current Runtime Facts", "# Project Context", "## Turn Metadata", llm.ToolUseEnforcementGuidance, "## Self Help", "## Current Session Context")
	if strings.Index(got, "## Current Runtime Facts") > strings.Index(got, "# Project Context") {
		t.Fatalf("runtime facts must precede context block:\n%s", got)
	}
}
