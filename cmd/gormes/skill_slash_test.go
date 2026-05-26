package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
)

func TestTUIWiresSkillSlashReload(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "active", "fresh-skill")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	raw := "---\nname: fresh-skill\ndescription: Fresh skill\n---\n\nFresh body.\n"
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(raw), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	fn := tuiSkillSlashReloadFunc(config.Config{Skills: config.SkillsCfg{Root: root, MaxDocumentBytes: 8 * 1024, SelectionCap: 5}})
	result, err := fn(context.Background())
	if err != nil {
		t.Fatalf("reload func error = %v", err)
	}
	if len(result.Commands) != 1 || result.Commands[0].Command != "/fresh-skill" {
		t.Fatalf("commands = %#v, want fresh skill", result.Commands)
	}
	if !strings.Contains(result.Output, "1 skill(s) available") {
		t.Fatalf("output = %q, want skill count", result.Output)
	}
}
