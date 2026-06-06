package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
)

func TestRootSkillsListUsesRuntimeSkillsRoot(t *testing.T) {
	home := t.TempDir()
	t.Setenv("GORMES_HOME", home)
	t.Setenv("GORMES_SKILLS_ROOT", "")
	t.Setenv("GORMES_BUNDLED_SKILLS_ROOT", "")
	writeRootCommandSkill(t, filepath.Join(home, "skills"), "runtime-skill")

	cmd := newRootCommandWithRuntime(rootRuntime{})
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"skills", "list", "--source", "local"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v stdout=%s stderr=%s", err, stdout.String(), stderr.String())
	}
	if !strings.Contains(stdout.String(), "runtime-skill") {
		t.Fatalf("skills list did not include runtime skill from GORMES_HOME skills root:\nstdout=%s\nstderr=%s", stdout.String(), stderr.String())
	}
}

func TestRootSkillsListUsesExternalDirsFromConfig(t *testing.T) {
	home := t.TempDir()
	external := filepath.Join(t.TempDir(), "team-skills")
	t.Setenv("GORMES_HOME", home)
	t.Setenv("GORMES_SKILLS_ROOT", "")
	t.Setenv("GORMES_BUNDLED_SKILLS_ROOT", "")
	writeExternalRootCommandSkill(t, external, "research", "external-skill")
	if err := config.WriteTOMLValue(config.ConfigPath(), "skills.external_dirs", external); err != nil {
		t.Fatalf("WriteTOMLValue skills.external_dirs: %v", err)
	}

	cmd := newRootCommandWithRuntime(rootRuntime{})
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"skills", "list", "--source", "external"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v stdout=%s stderr=%s", err, stdout.String(), stderr.String())
	}
	out := stdout.String()
	for _, want := range []string{"external-skill", "research", "external", "operator"} {
		if !strings.Contains(out, want) {
			t.Fatalf("skills list missing %q:\nstdout=%s\nstderr=%s", want, out, stderr.String())
		}
	}
	if strings.Contains(out, external) {
		t.Fatalf("skills list leaked external root path:\n%s", out)
	}
}

func writeRootCommandSkill(t *testing.T, root, name string) {
	t.Helper()
	dir := filepath.Join(root, "active", name)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("MkdirAll(%q): %v", dir, err)
	}
	raw := "---\nname: " + name + "\ndescription: Runtime skill used by root command tests\n---\n\nUse this skill from the runtime skill root."
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(raw), 0o644); err != nil {
		t.Fatalf("WriteFile(SKILL.md): %v", err)
	}
}

func writeExternalRootCommandSkill(t *testing.T, root, category, name string) {
	t.Helper()
	dir := filepath.Join(root, category, name)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("MkdirAll(%q): %v", dir, err)
	}
	raw := "---\nname: " + name + "\ndescription: External runtime skill used by root command tests\n---\n\nUse this external skill."
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(raw), 0o644); err != nil {
		t.Fatalf("WriteFile(SKILL.md): %v", err)
	}
}
