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
	mainRoot := filepath.Join(xdg, "gormes", "profiles", "main")
	mustWriteProfileFile(t, filepath.Join(mainRoot, "config.toml"), "model = \"gpt-4\"\n")
	mustWriteProfileFile(t, filepath.Join(mainRoot, ".env"), "TOKEN=placeholder\n")
	mustWriteProfileFile(t, filepath.Join(mainRoot, "state.db"), "state")
	mustWriteProfileFile(t, filepath.Join(mainRoot, "home", ".gitconfig"), "[user]\n\tname = Worker\n")
	mustWriteProfileFile(t, filepath.Join(mainRoot, "sessions", "turn.json"), "{}")
	mustWriteProfileFile(t, filepath.Join(mainRoot, "runtime", "gateway.log"), "log")
	mustWriteProfileFile(t, filepath.Join(mainRoot, "skills", "my-skill", "SKILL.md"), "skill")

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
		mustWriteProfileFile(t, filepath.Join(mainRoot, path), "skip")
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
		"runtime/gateway.log",
	} {
		if _, err := os.Stat(filepath.Join(target, path)); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("expected excluded %s, stat err=%v", path, err)
		}
	}
}

func TestCreateProfileCloneAllUsesMaterializedMainProfileSource(t *testing.T) {
	xdg := t.TempDir()
	baseRoot := filepath.Join(xdg, "gormes")
	materializedMainRoot := filepath.Join(baseRoot, "profiles", "main")
	mustWriteProfileFile(t, filepath.Join(baseRoot, "legacy-only.txt"), "legacy")
	mustWriteProfileFile(t, filepath.Join(materializedMainRoot, "config.toml"), "model = \"materialized\"\n")

	result, err := CreateProfile(ProfileCreateOptions{Name: "work", XDGConfigHome: xdg, CloneAll: true})
	if err != nil {
		t.Fatalf("CreateProfile clone_all from materialized main: %v", err)
	}
	if got, want := result.Root, filepath.Join(baseRoot, "profiles", "work"); got != want {
		t.Fatalf("created root = %q, want %q", got, want)
	}
	if got, err := os.ReadFile(filepath.Join(result.Root, "config.toml")); err != nil || !strings.Contains(string(got), "materialized") {
		t.Fatalf("cloned config = %q, %v; want materialized main config", string(got), err)
	}
	if _, err := os.Stat(filepath.Join(result.Root, "legacy-only.txt")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("clone_all copied base root despite materialized main, stat err=%v", err)
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

func TestCreateProfileSeedsEditableContextFiles(t *testing.T) {
	xdg := t.TempDir()

	result, err := CreateProfile(ProfileCreateOptions{Name: "worker", DisplayName: "Worker Bee", XDGConfigHome: xdg})
	if err != nil {
		t.Fatalf("CreateProfile: %v", err)
	}

	for _, rel := range []string{"SOUL.md", "AGENTS.md", "IDENTITY.md", "TOOLS.md", filepath.Join("memory", "USER.md"), filepath.Join("memory", "MEMORY.md")} {
		body, err := os.ReadFile(filepath.Join(result.Root, rel))
		if err != nil {
			t.Fatalf("created profile missing %s: %v", rel, err)
		}
		if len(body) == 0 {
			t.Fatalf("created profile file %s is empty", rel)
		}
	}
	identity, err := os.ReadFile(filepath.Join(result.Root, "IDENTITY.md"))
	if err != nil {
		t.Fatalf("read IDENTITY.md: %v", err)
	}
	if !strings.Contains(string(identity), "- Name: Worker Bee\n- Profile ID: `worker`") {
		t.Fatalf("IDENTITY.md = %q, want profile display name and ID", string(identity))
	}
	if _, err := os.Stat(filepath.Join(result.Root, "memory.db")); err != nil {
		t.Fatalf("created profile missing bootstrapped memory.db: %v", err)
	}
}

func TestCreateProfileCloneAllStripsRuntimeFiles(t *testing.T) {
	xdg := t.TempDir()
	mainRoot := filepath.Join(xdg, "gormes", "profiles", "main")
	mustWriteProfileFile(t, filepath.Join(mainRoot, "config.toml"), "model = \"gpt-4\"\n")
	for _, path := range []string{
		"runtime/gateway.pid",
		"runtime/gateway_state.json",
		"runtime/gateway-locks",
		"runtime/gateway.log",
		"processes.json",
		"memory.db",
		"memory.db-wal",
		"memory.db-shm",
		"sessions.db",
		"sessions.db-wal",
		"sessions.db-shm",
		"kanban.db",
	} {
		mustWriteProfileFile(t, filepath.Join(mainRoot, path), "runtime")
	}

	if _, err := CreateProfile(ProfileCreateOptions{Name: "work", XDGConfigHome: xdg, CloneAll: true}); err != nil {
		t.Fatalf("CreateProfile clone_all: %v", err)
	}
	target := filepath.Join(xdg, "gormes", "profiles", "work")
	if _, err := os.Stat(filepath.Join(target, "config.toml")); err != nil {
		t.Fatalf("expected copied config: %v", err)
	}
	for _, path := range []string{
		"runtime/gateway.pid",
		"runtime/gateway_state.json",
		"runtime/gateway-locks",
		"runtime/gateway.log",
		"processes.json",
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
	memoryDB := filepath.Join(target, "memory.db")
	if body, err := os.ReadFile(memoryDB); err != nil {
		t.Fatalf("bootstrapped memory.db missing after clone-all strip: %v", err)
	} else if string(body) == "runtime" {
		t.Fatalf("memory.db copied source runtime bytes instead of bootstrapping empty schema")
	}
	assertMemoryDBBootstrapped(t, memoryDB)
}

func TestCreateProfileRejectsDefaultAndExistingTargets(t *testing.T) {
	xdg := t.TempDir()
	mainRoot := filepath.Join(xdg, "gormes")
	mustWriteProfileFile(t, filepath.Join(mainRoot, "config.toml"), "model = \"gpt-4\"\n")
	existing := filepath.Join(mainRoot, "profiles", "work")
	mustWriteProfileFile(t, filepath.Join(existing, "marker.txt"), "keep")

	_, err := CreateProfile(ProfileCreateOptions{Name: "default", XDGConfigHome: xdg, CloneAll: true})
	if !errors.Is(err, ErrProfileCreateDefaultReserved) {
		t.Fatalf("CreateProfile(default) err = %v, want ErrProfileCreateDefaultReserved", err)
	}
	if _, err := os.Stat(filepath.Join(mainRoot, "profiles", "default")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("default target should not be created, stat err=%v", err)
	}

	_, err = CreateProfile(ProfileCreateOptions{Name: "main", XDGConfigHome: xdg, CloneAll: true})
	if !errors.Is(err, ErrProfileCreateDefaultReserved) {
		t.Fatalf("CreateProfile(main) err = %v, want ErrProfileCreateDefaultReserved", err)
	}
	if _, err := os.Stat(filepath.Join(mainRoot, "profiles", "main")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("main target should not be created, stat err=%v", err)
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
