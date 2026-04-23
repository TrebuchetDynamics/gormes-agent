package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/gormes/internal/config"
)

func TestConfiguredSkillsRuntimeAutoDiscoversInstalledSkills(t *testing.T) {
	dataHome := t.TempDir()
	configHome := t.TempDir()
	t.Setenv("XDG_DATA_HOME", dataHome)
	t.Setenv("XDG_CONFIG_HOME", configHome)

	skillsRoot := filepath.Join(dataHome, "gormes", "skills")
	if err := os.MkdirAll(filepath.Join(skillsRoot, "active", "review", "careful-review"), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	raw := "---\nname: careful-review\ndescription: Review carefully\n---\n\nFollow the review checklist."
	if err := os.WriteFile(filepath.Join(skillsRoot, "active", "review", "careful-review", "SKILL.md"), []byte(raw), 0o644); err != nil {
		t.Fatalf("WriteFile(SKILL.md): %v", err)
	}

	cfg, err := config.Load(nil)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	runtime := configuredSkillsRuntime(cfg)
	if runtime == nil {
		t.Fatal("configuredSkillsRuntime() = nil, want runtime")
	}

	block, names, err := runtime.BuildSkillBlock(context.Background(), "review this carefully")
	if err != nil {
		t.Fatalf("BuildSkillBlock() error = %v", err)
	}
	if len(names) != 1 || names[0] != "careful-review" {
		t.Fatalf("names = %#v, want [careful-review]", names)
	}
	if block == "" {
		t.Fatal("BuildSkillBlock() returned empty block, want selected skill block")
	}
	if _, err := os.Stat(filepath.Join(dataHome, "gormes", ".skills_prompt_snapshot.json")); err != nil {
		t.Fatalf("prompt snapshot missing: %v", err)
	}
}
