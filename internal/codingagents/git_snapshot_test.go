package codingagents

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func ensureGit(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skipf("git not available: %v", err)
	}
}

func runIn(t *testing.T, dir string, name string, args ...string) {
	t.Helper()
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(),
		"GIT_AUTHOR_NAME=Gormes Test",
		"GIT_AUTHOR_EMAIL=test@gormes.local",
		"GIT_COMMITTER_NAME=Gormes Test",
		"GIT_COMMITTER_EMAIL=test@gormes.local",
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("%s %v failed: %v\n%s", name, args, err, out)
	}
}

func initRepo(t *testing.T) string {
	t.Helper()
	ensureGit(t)
	dir := t.TempDir()
	runIn(t, dir, "git", "init", "-b", "main", ".")
	runIn(t, dir, "git", "config", "user.email", "test@gormes.local")
	runIn(t, dir, "git", "config", "user.name", "Gormes Test")
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("seed\n"), 0o644); err != nil {
		t.Fatalf("write seed: %v", err)
	}
	runIn(t, dir, "git", "add", "README.md")
	runIn(t, dir, "git", "commit", "-m", "seed")
	return dir
}

func TestTakeSnapshot_HappyPath(t *testing.T) {
	t.Parallel()
	dir := initRepo(t)
	snap, err := TakeSnapshot(context.Background(), dir)
	if err != nil {
		t.Fatalf("TakeSnapshot: %v", err)
	}
	if snap.Head == "" {
		t.Fatalf("expected non-empty Head")
	}
	if snap.Branch != "main" {
		t.Fatalf("Branch = %q, want main", snap.Branch)
	}
	if snap.Dirty {
		t.Fatalf("clean repo should not be Dirty: %+v", snap)
	}
	if len(snap.Files) != 0 {
		t.Fatalf("clean repo Files = %v, want empty", snap.Files)
	}
}

func TestTakeSnapshot_DirtyDetected(t *testing.T) {
	t.Parallel()
	dir := initRepo(t)
	if err := os.WriteFile(filepath.Join(dir, "new.txt"), []byte("hello"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	snap, err := TakeSnapshot(context.Background(), dir)
	if err != nil {
		t.Fatalf("TakeSnapshot: %v", err)
	}
	if !snap.Dirty {
		t.Fatalf("expected Dirty=true after untracked write, got %+v", snap)
	}
	found := false
	for _, line := range snap.Files {
		if strings.Contains(line, "new.txt") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected new.txt in Files, got %v", snap.Files)
	}
}

func TestTakeSnapshot_NotAGitRepo(t *testing.T) {
	t.Parallel()
	ensureGit(t)
	dir := t.TempDir()
	_, err := TakeSnapshot(context.Background(), dir)
	if !errors.Is(err, ErrNotAGitRepo) {
		t.Fatalf("error = %v, want ErrNotAGitRepo", err)
	}
}

func TestDiffBetween_ProducesDiffAndFileList(t *testing.T) {
	t.Parallel()
	dir := initRepo(t)
	before, err := TakeSnapshot(context.Background(), dir)
	if err != nil {
		t.Fatalf("before snapshot: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "feature.go"), []byte("package x\n"), 0o644); err != nil {
		t.Fatalf("write feature: %v", err)
	}
	runIn(t, dir, "git", "add", "feature.go")
	runIn(t, dir, "git", "commit", "-m", "add feature")
	after, err := TakeSnapshot(context.Background(), dir)
	if err != nil {
		t.Fatalf("after snapshot: %v", err)
	}
	diff, files, err := DiffBetween(context.Background(), dir, before, after)
	if err != nil {
		t.Fatalf("DiffBetween: %v", err)
	}
	if !strings.Contains(diff, "feature.go") {
		t.Fatalf("diff missing feature.go: %s", diff)
	}
	found := false
	for _, f := range files {
		if f == "feature.go" {
			found = true
		}
	}
	if !found {
		t.Fatalf("files = %v, want feature.go", files)
	}
}

func TestDiffBetween_IncludesUntrackedWorkingTreeFiles(t *testing.T) {
	t.Parallel()
	dir := initRepo(t)
	before, err := TakeSnapshot(context.Background(), dir)
	if err != nil {
		t.Fatalf("before snapshot: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "scratch.txt"), []byte("draft"), 0o644); err != nil {
		t.Fatalf("write scratch: %v", err)
	}
	after, err := TakeSnapshot(context.Background(), dir)
	if err != nil {
		t.Fatalf("after snapshot: %v", err)
	}
	_, files, err := DiffBetween(context.Background(), dir, before, after)
	if err != nil {
		t.Fatalf("DiffBetween: %v", err)
	}
	found := false
	for _, f := range files {
		if f == "scratch.txt" {
			found = true
		}
	}
	if !found {
		t.Fatalf("files = %v, want untracked scratch.txt", files)
	}
}
