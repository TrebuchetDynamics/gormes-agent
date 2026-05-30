package mirror

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestDefaultMirrorConfigUsesGormesHome(t *testing.T) {
	root := t.TempDir()
	gormesHome := filepath.Join(root, "gormes")
	hermesHome := filepath.Join(root, "hermes")
	t.Setenv("GORMES_HOME", gormesHome)
	t.Setenv("HERMES_HOME", hermesHome)

	cfg := DefaultMirrorConfig()

	want := filepath.Join(gormesHome, "memory", "USER.md")
	if cfg.Path != want {
		t.Fatalf("Path = %q, want %q", cfg.Path, want)
	}
	if strings.HasPrefix(cfg.Path, hermesHome+string(filepath.Separator)) {
		t.Fatalf("Path = %q, want not under poisoned Hermes home %q", cfg.Path, hermesHome)
	}
}
