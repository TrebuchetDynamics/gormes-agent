package gormeshome

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFixtureGormesHomeIsIsolatedAndWritable(t *testing.T) {
	h := New(t)
	if os.Getenv("GORMES_HOME") != h.GormesHome || os.Getenv("HERMES_HOME") != h.HermesHome || os.Getenv("CODEX_HOME") != h.CodexHome {
		t.Fatalf("home env not isolated: GORMES_HOME=%q HERMES_HOME=%q CODEX_HOME=%q fixture=%+v", os.Getenv("GORMES_HOME"), os.Getenv("HERMES_HOME"), os.Getenv("CODEX_HOME"), h)
	}
	configPath := h.WriteConfig(t, "[hermes]\nmodel = 'fake'\n")
	envPath := h.WriteEnv(t, "GORMES_API_KEY=sk-fixture\n")
	for _, path := range []string{configPath, envPath} {
		if filepath.Dir(path) != h.GormesHome {
			t.Fatalf("%s not under fixture GORMES_HOME %s", path, h.GormesHome)
		}
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("stat %s: %v", path, err)
		}
	}
}
