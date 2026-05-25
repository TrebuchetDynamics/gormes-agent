package cli

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCreateProfileCloneAllExcludesDefaultInfrastructure(t *testing.T) {
	xdg := t.TempDir()
	defaultRoot := filepath.Join(xdg, "gormes")
	mustWriteProfileFile(t, filepath.Join(defaultRoot, "config.toml"), "model = \"gpt-4\"\n")
	mustWriteProfileFile(t, filepath.Join(defaultRoot, ".env"), "TOKEN=placeholder\n")
	mustWriteProfileFile(t, filepath.Join(defaultRoot, "state.db"), "state")
	mustWriteProfileFile(t, filepath.Join(defaultRoot, "home", ".gitconfig"), "[user]\n\tname = Worker\n")
	mustWriteProfileFile(t, filepath.Join(defaultRoot, "sessions", "turn.json"), "{}")
	mustWriteProfileFile(t, filepath.Join(defaultRoot, "logs", "gateway.log"), "log")
	mustWriteProfileFile(t, filepath.Join(defaultRoot, "skills", "my-skill", "SKILL.md"), "skill")

	for _, path := range []string{
		"gormes-agent/.git/HEAD",
		".worktrees/some-tree/HEAD",
		"profiles/other/config.toml",
		"bin/gormes",
		"node_modules/pkg/index.js",
		"skills/my-skill/__pycache__/module.pyc",
		"skills/my-skill/module.pyc",
		"skills/my-skill/module.pyo",
		"skills/my-skill/data.sock",
		"skills/my-skill/data.tmp",
	} {
		mustWriteProfileFile(t, filepath.Join(defaultRoot, path), "skip")
	}

	result, err := CreateProfile(ProfileCreateOptions{
		Name:          "cloned",
		XDGConfigHome: xdg,
		CloneAll:      true,
	})
	if err != nil {
		t.Fatalf("CreateProfile clone_all: %v", err)
	}
	target := filepath.Join(xdg, "gormes", "profiles", "cloned")
	if result.Root != target || !result.CloneAll || result.Name != "cloned" {
		t.Fatalf("result = %+v, want name=cloned root=%s clone_all=true", result, target)
	}

	for _, path := range []string{
		"config.toml",
		".env",
		"state.db",
		"home/.gitconfig",
		"sessions/turn.json",
		"logs/gateway.log",
		"skills/my-skill/SKILL.md",
	} {
		if _, err := os.Stat(filepath.Join(target, path)); err != nil {
			t.Fatalf("expected copied profile data %s: %v", path, err)
		}
	}
	for _, path := range []string{
		"gormes-agent",
		".worktrees",
		"profiles",
		"bin",
		"node_modules",
		"skills/my-skill/__pycache__",
		"skills/my-skill/module.pyc",
		"skills/my-skill/module.pyo",
		"skills/my-skill/data.sock",
		"skills/my-skill/data.tmp",
	} {
		if _, err := os.Stat(filepath.Join(target, path)); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("expected excluded %s, stat err=%v", path, err)
		}
	}
}

func TestCreateProfileSeedsSubprocessHomeDir(t *testing.T) {
	xdg := t.TempDir()

	result, err := CreateProfile(ProfileCreateOptions{
		Name:          "worker",
		XDGConfigHome: xdg,
	})
	if err != nil {
		t.Fatalf("CreateProfile: %v", err)
	}

	homeDir := filepath.Join(result.Root, "home")
	info, err := os.Stat(homeDir)
	if err != nil {
		t.Fatalf("profile subprocess home missing at %s: %v", homeDir, err)
	}
	if !info.IsDir() {
		t.Fatalf("profile subprocess home = non-directory %s", homeDir)
	}
}

func TestCreateProfileCloneAllStripsRuntimeFiles(t *testing.T) {
	xdg := t.TempDir()
	defaultRoot := filepath.Join(xdg, "gormes")
	mustWriteProfileFile(t, filepath.Join(defaultRoot, "config.toml"), "model = \"gpt-4\"\n")
	for _, path := range []string{
		"gateway.pid",
		"gateway_state.json",
		"processes.json",
		"memory.db",
		"memory.db-wal",
		"memory.db-shm",
		"sessions.db",
		"sessions.db-wal",
		"sessions.db-shm",
		"kanban.db",
	} {
		mustWriteProfileFile(t, filepath.Join(defaultRoot, path), "runtime")
	}

	if _, err := CreateProfile(ProfileCreateOptions{Name: "work", XDGConfigHome: xdg, CloneAll: true}); err != nil {
		t.Fatalf("CreateProfile clone_all: %v", err)
	}
	target := filepath.Join(xdg, "gormes", "profiles", "work")
	if _, err := os.Stat(filepath.Join(target, "config.toml")); err != nil {
		t.Fatalf("expected copied config: %v", err)
	}
	for _, path := range []string{
		"gateway.pid",
		"gateway_state.json",
		"processes.json",
		"memory.db",
		"memory.db-wal",
		"memory.db-shm",
		"sessions.db",
		"sessions.db-wal",
		"sessions.db-shm",
		"kanban.db",
	} {
		if _, err := os.Stat(filepath.Join(target, path)); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("runtime file %s should be stripped, stat err=%v", path, err)
		}
	}
}

func TestCreateProfileRejectsDefaultAndExistingTargets(t *testing.T) {
	xdg := t.TempDir()
	defaultRoot := filepath.Join(xdg, "gormes")
	mustWriteProfileFile(t, filepath.Join(defaultRoot, "config.toml"), "model = \"gpt-4\"\n")
	existing := filepath.Join(defaultRoot, "profiles", "work")
	mustWriteProfileFile(t, filepath.Join(existing, "marker.txt"), "keep")

	_, err := CreateProfile(ProfileCreateOptions{Name: "default", XDGConfigHome: xdg, CloneAll: true})
	if !errors.Is(err, ErrProfileCreateDefaultReserved) {
		t.Fatalf("CreateProfile(default) err = %v, want ErrProfileCreateDefaultReserved", err)
	}
	if _, err := os.Stat(filepath.Join(defaultRoot, "profiles", "default")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("default target should not be created, stat err=%v", err)
	}

	_, err = CreateProfile(ProfileCreateOptions{Name: "work", XDGConfigHome: xdg, CloneAll: true})
	if !errors.Is(err, ErrProfileCreateTargetExists) {
		t.Fatalf("CreateProfile(existing) err = %v, want ErrProfileCreateTargetExists", err)
	}
	got, readErr := os.ReadFile(filepath.Join(existing, "marker.txt"))
	if readErr != nil {
		t.Fatalf("existing marker missing after rejected create: %v", readErr)
	}
	if strings.TrimSpace(string(got)) != "keep" {
		t.Fatalf("existing marker mutated: %q", string(got))
	}

	_, err = CreateProfile(ProfileCreateOptions{Name: "Bad Name", XDGConfigHome: xdg, CloneAll: true})
	if !errors.Is(err, ErrProfileNameInvalidChars) {
		t.Fatalf("CreateProfile(invalid) err = %v, want ErrProfileNameInvalidChars", err)
	}
}

func mustWriteProfileFile(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
