package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/skills"
)

func TestCuratorCommand_Status(t *testing.T) {
	root := setupCuratorCommandHome(t)
	writeCuratorCommandSkill(t, root, "top-dog")
	writeCuratorCommandSkill(t, root, "middling")
	writeCuratorCommandSkill(t, root, "never-used")
	for _, name := range []string{"top-dog", "middling", "never-used"} {
		if err := skills.MarkAgentCreated(root, name); err != nil {
			t.Fatalf("MarkAgentCreated(%s): %v", name, err)
		}
	}
	for i := 0; i < 10; i++ {
		if err := skills.BumpUse(root, "top-dog"); err != nil {
			t.Fatalf("BumpUse top-dog: %v", err)
		}
	}
	for i := 0; i < 2; i++ {
		if err := skills.BumpUse(root, "middling"); err != nil {
			t.Fatalf("BumpUse middling: %v", err)
		}
	}
	lastRun := time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)
	curator := skills.NewCurator(skills.CuratorConfig{Root: root})
	if err := curator.SaveState(skills.CuratorState{
		LastRunAt:      &lastRun,
		LastRunSummary: "curator run completed",
		LastReportPath: filepath.Join(root, "logs", "curator", "REPORT.md"),
		RunCount:       3,
	}); err != nil {
		t.Fatalf("SaveState: %v", err)
	}

	stdout, stderr, err := executeRootCommandForTest(newRootCommandWithRuntime(rootRuntime{}), "curator", "status")
	if err != nil {
		t.Fatalf("curator status: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	for _, want := range []string{
		"curator: ENABLED",
		"runs:           3",
		"last summary:   curator run completed",
		"interval:       every 7d",
		"stale after:    30d unused",
		"archive after:  90d unused",
		"agent-created skills: 3 total",
		"least recently active (top 5):",
		"most active (top 5):",
		"least active (top 5):",
		"top-dog",
		"activity= 10",
		"never-used",
		"activity=  0",
	} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("curator status stdout missing %q:\n%s", want, stdout)
		}
	}
}

func TestCuratorCommand_RunDryRunPauseResumePinUnpin(t *testing.T) {
	root := setupCuratorCommandHome(t)
	writeCuratorCommandSkill(t, root, "agent-skill")
	writeCuratorCommandSkill(t, root, "bundled-skill")
	if err := skills.MarkAgentCreated(root, "agent-skill"); err != nil {
		t.Fatalf("MarkAgentCreated agent-skill: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, ".bundled_manifest"), []byte("bundled-skill\n"), 0o600); err != nil {
		t.Fatalf("write bundled manifest: %v", err)
	}

	stdout, stderr, err := executeRootCommandForTest(newRootCommandWithRuntime(rootRuntime{}), "curator", "run", "--dry-run", "--sync")
	if err != nil {
		t.Fatalf("curator run --dry-run --sync: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	for _, want := range []string{"curator: running DRY-RUN", "dry-run: no changes applied", "report:"} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("dry-run stdout missing %q:\n%s", want, stdout)
		}
	}
	if _, err := os.Stat(filepath.Join(root, ".curator_backups")); !os.IsNotExist(err) {
		t.Fatalf("dry-run created backups dir or stat failed: %v", err)
	}

	stdout, stderr, err = executeRootCommandForTest(newRootCommandWithRuntime(rootRuntime{}), "curator", "pause")
	if err != nil || !strings.Contains(stdout, "curator: paused") {
		t.Fatalf("curator pause = %v stdout=%q stderr=%q", err, stdout, stderr)
	}
	state, err := skills.NewCurator(skills.CuratorConfig{Root: root}).LoadState()
	if err != nil {
		t.Fatalf("LoadState after pause: %v", err)
	}
	if !state.Paused {
		t.Fatalf("state.Paused = false after pause")
	}

	stdout, stderr, err = executeRootCommandForTest(newRootCommandWithRuntime(rootRuntime{}), "curator", "resume")
	if err != nil || !strings.Contains(stdout, "curator: resumed") {
		t.Fatalf("curator resume = %v stdout=%q stderr=%q", err, stdout, stderr)
	}
	state, err = skills.NewCurator(skills.CuratorConfig{Root: root}).LoadState()
	if err != nil {
		t.Fatalf("LoadState after resume: %v", err)
	}
	if state.Paused {
		t.Fatalf("state.Paused = true after resume")
	}

	stdout, stderr, err = executeRootCommandForTest(newRootCommandWithRuntime(rootRuntime{}), "curator", "pin", "agent-skill")
	if err != nil || !strings.Contains(stdout, "curator: pinned 'agent-skill'") {
		t.Fatalf("curator pin = %v stdout=%q stderr=%q", err, stdout, stderr)
	}
	pinned, err := skills.IsPinned(root, "agent-skill")
	if err != nil {
		t.Fatalf("IsPinned: %v", err)
	}
	if !pinned {
		t.Fatalf("agent-skill pinned = false")
	}

	stdout, stderr, err = executeRootCommandForTest(newRootCommandWithRuntime(rootRuntime{}), "curator", "unpin", "agent-skill")
	if err != nil || !strings.Contains(stdout, "curator: unpinned 'agent-skill'") {
		t.Fatalf("curator unpin = %v stdout=%q stderr=%q", err, stdout, stderr)
	}
	pinned, err = skills.IsPinned(root, "agent-skill")
	if err != nil {
		t.Fatalf("IsPinned after unpin: %v", err)
	}
	if pinned {
		t.Fatalf("agent-skill pinned = true after unpin")
	}

	stdout, stderr, err = executeRootCommandForTest(newRootCommandWithRuntime(rootRuntime{}), "curator", "pin", "bundled-skill")
	if err == nil {
		t.Fatalf("curator pin bundled-skill err = nil\nstdout=%s\nstderr=%s", stdout, stderr)
	}
	if !strings.Contains(stdout+stderr+err.Error(), "bundled or hub-installed") {
		t.Fatalf("curator pin bundled output missing refusal:\nstdout=%s\nstderr=%s\nerr=%v", stdout, stderr, err)
	}
}

func TestCuratorCommand_BackupRollbackRestore(t *testing.T) {
	root := setupCuratorCommandHome(t)
	writeCuratorCommandSkill(t, root, "keeper")
	if err := skills.MarkAgentCreated(root, "keeper"); err != nil {
		t.Fatalf("MarkAgentCreated keeper: %v", err)
	}

	stdout, stderr, err := executeRootCommandForTest(newRootCommandWithRuntime(rootRuntime{}), "curator", "backup", "--reason", "manual-test")
	if err != nil {
		t.Fatalf("curator backup: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	if !strings.Contains(stdout, "curator: snapshot created") {
		t.Fatalf("curator backup stdout = %q, want snapshot created", stdout)
	}
	backupID := newestCuratorBackupID(t, root)
	if backupID == "" {
		t.Fatalf("missing curator backup id")
	}

	if err := os.RemoveAll(filepath.Join(root, "active", "keeper")); err != nil {
		t.Fatalf("remove keeper: %v", err)
	}
	stdout, stderr, err = executeRootCommandForTest(newRootCommandWithRuntime(rootRuntime{}), "curator", "rollback", "--id", backupID, "--yes")
	if err != nil {
		t.Fatalf("curator rollback: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	if !strings.Contains(stdout, "curator: rollback restored") {
		t.Fatalf("curator rollback stdout = %q, want restored evidence", stdout)
	}
	if _, err := os.Stat(filepath.Join(root, "active", "keeper", "SKILL.md")); err != nil {
		t.Fatalf("keeper not restored: %v", err)
	}

	stdout, stderr, err = executeRootCommandForTest(newRootCommandWithRuntime(rootRuntime{}), "curator", "rollback", "--list")
	if err != nil {
		t.Fatalf("curator rollback --list: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	if !strings.Contains(stdout, backupID) || !strings.Contains(stdout, "manual-test") {
		t.Fatalf("rollback --list stdout missing backup evidence:\n%s", stdout)
	}

	archivedDir := filepath.Join(root, "active", ".archive", "archived-skill")
	if err := os.MkdirAll(archivedDir, 0o755); err != nil {
		t.Fatalf("mkdir archive: %v", err)
	}
	if err := os.WriteFile(filepath.Join(archivedDir, "SKILL.md"), []byte("---\nname: archived-skill\ndescription: archived\n---\n# archived\n"), 0o600); err != nil {
		t.Fatalf("write archived skill: %v", err)
	}
	if err := skills.MarkAgentCreated(root, "archived-skill"); err != nil {
		t.Fatalf("MarkAgentCreated archived-skill: %v", err)
	}
	if err := skills.SetSkillState(root, "archived-skill", skills.SkillStateArchived); err != nil {
		t.Fatalf("SetSkillState archived-skill: %v", err)
	}
	stdout, stderr, err = executeRootCommandForTest(newRootCommandWithRuntime(rootRuntime{}), "curator", "restore", "archived-skill")
	if err != nil {
		t.Fatalf("curator restore: %v\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	if !strings.Contains(stdout, "curator: restored 'archived-skill'") {
		t.Fatalf("restore stdout = %q, want restored archived-skill", stdout)
	}
	if _, err := os.Stat(filepath.Join(root, "active", "archived-skill", "SKILL.md")); err != nil {
		t.Fatalf("archived-skill not restored: %v", err)
	}
}

func TestRootCommandIncludesCuratorCommand(t *testing.T) {
	root := newRootCommandWithRuntime(rootRuntime{})
	cmd, _, err := root.Find([]string{"curator", "status"})
	if err != nil {
		t.Fatalf("find curator status: %v", err)
	}
	if cmd == nil || cmd.Use != "status" {
		t.Fatalf("root command did not expose curator status: %#v", cmd)
	}
}

func setupCuratorCommandHome(t *testing.T) string {
	t.Helper()
	home := filepath.Join(t.TempDir(), "gormes")
	t.Setenv("GORMES_HOME", home)
	t.Setenv("HERMES_HOME", filepath.Join(t.TempDir(), "hermes"))
	root := filepath.Join(home, "skills")
	if err := os.MkdirAll(filepath.Join(root, "active"), 0o755); err != nil {
		t.Fatalf("mkdir skills active: %v", err)
	}
	return root
}

func writeCuratorCommandSkill(t *testing.T, root, name string) {
	t.Helper()
	dir := filepath.Join(root, "active", name)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir skill %s: %v", name, err)
	}
	body := "---\nname: " + name + "\ndescription: test\n---\n# " + name + "\n"
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(body), 0o600); err != nil {
		t.Fatalf("write SKILL.md %s: %v", name, err)
	}
}

func newestCuratorBackupID(t *testing.T, root string) string {
	t.Helper()
	entries, err := os.ReadDir(filepath.Join(root, ".curator_backups"))
	if err != nil {
		t.Fatalf("read curator backups: %v", err)
	}
	var newest string
	for _, entry := range entries {
		if entry.IsDir() && entry.Name() > newest {
			newest = entry.Name()
		}
	}
	return newest
}
