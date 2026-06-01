package gormeshome

import (
	"os"
	"path/filepath"
	"testing"
)

type Home struct {
	Root       string
	GormesHome string
	HermesHome string
	CodexHome  string
}

func New(t testing.TB) Home {
	t.Helper()
	root := t.TempDir()
	h := Home{
		Root:       root,
		GormesHome: filepath.Join(root, "gormes-home"),
		HermesHome: filepath.Join(root, "hermes-home"),
		CodexHome:  filepath.Join(root, "codex-home"),
	}
	for _, dir := range []string{h.GormesHome, h.HermesHome, h.CodexHome} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", dir, err)
		}
	}
	t.Setenv("GORMES_HOME", h.GormesHome)
	t.Setenv("HERMES_HOME", h.HermesHome)
	t.Setenv("CODEX_HOME", h.CodexHome)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(root, "xdg-config"))
	t.Setenv("XDG_DATA_HOME", filepath.Join(root, "xdg-data"))
	return h
}

func (h Home) WriteConfig(t testing.TB, body string) string {
	t.Helper()
	path := filepath.Join(h.GormesHome, "config.toml")
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	return path
}

func (h Home) WriteEnv(t testing.TB, body string) string {
	t.Helper()
	path := filepath.Join(h.GormesHome, ".env")
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatalf("write env: %v", err)
	}
	return path
}
