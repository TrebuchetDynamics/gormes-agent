package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
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

func TestOnboardExplainsRuntimeSkillsAndLearningState(t *testing.T) {
	home := t.TempDir()
	t.Setenv("GORMES_HOME", home)
	t.Setenv("GORMES_SKILLS_ROOT", "")
	t.Setenv("GORMES_BUNDLED_SKILLS_ROOT", "")

	cmd := newRootCommandWithRuntime(rootRuntime{})
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"onboard"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v stdout=%s stderr=%s", err, stdout.String(), stderr.String())
	}
	output := stdout.String()
	for _, want := range []string{
		"Gormes onboarding",
		filepath.Join(home, "skills"),
		"Runtime skills",
		"docs/development-skills",
		"manual/prompted",
		"gormes skills list",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("onboard output missing %q:\n%s", want, output)
		}
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
