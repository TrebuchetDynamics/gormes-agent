package gateway

import (
	"os"
	"path/filepath"
	"testing"
)

func writeActiveSkill(t *testing.T, name, description, body string) string {
	t.Helper()
	root := t.TempDir()
	dir := filepath.Join(root, "active", name)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	raw := "---\nname: " + name + "\ndescription: " + description + "\n---\n\n" + body + "\n"
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(raw), 0o644); err != nil {
		t.Fatalf("WriteFile SKILL.md: %v", err)
	}
	return root
}
