package skillscmd

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"unicode/utf8"
)

func TestHandleSkillsInspectBodyPreviewKeepsUTF8Boundary(t *testing.T) {
	root := t.TempDir()
	body := strings.Repeat("a", 1999) + "🙂tail"
	writeSkillCommandSkill(t, root, "active/utf8-skill", "utf8-skill", "UTF-8 skill", body)

	out := HandleSkillsCommandWithOptions(context.Background(), "/skills inspect utf8-skill", SkillsCommandOptions{SkillsRoot: root})

	if !utf8.ValidString(out) {
		t.Fatalf("inspect output is not valid UTF-8 near body boundary: %q", out)
	}
	if !strings.Contains(out, strings.Repeat("a", 1999)+"🙂") {
		t.Fatalf("inspect output dropped complete boundary rune; output suffix=%q", out[len(out)-min(len(out), 120):])
	}
	if strings.Contains(out, string(utf8.RuneError)+"tail") {
		t.Fatalf("inspect output contains replacement rune from split UTF-8 body: %q", out)
	}
}

func writeSkillCommandSkill(t *testing.T, root, rel, name, description, body string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(rel), "SKILL.md")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir skill: %v", err)
	}
	raw := "---\nname: " + name + "\ndescription: " + description + "\n---\n\n" + body
	if err := os.WriteFile(path, []byte(raw), 0o644); err != nil {
		t.Fatalf("write skill: %v", err)
	}
}
