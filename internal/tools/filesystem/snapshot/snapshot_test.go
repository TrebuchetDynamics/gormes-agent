package snapshot

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestSnapshotRestoresModifiedDeletedAndCreatedFiles(t *testing.T) {
	root := t.TempDir()
	mustWriteSnapshotTestFile(t, root, "dir/existing.txt", "before\n")
	mustWriteSnapshotTestFile(t, root, "dir/deleted.txt", "keep\n")

	snapshot, err := TakeWorkspaceSnapshot(root)
	if err != nil {
		t.Fatalf("TakeWorkspaceSnapshot: %v", err)
	}

	mustWriteSnapshotTestFile(t, root, "dir/existing.txt", "after\n")
	if err := os.Remove(filepath.Join(root, "dir/deleted.txt")); err != nil {
		t.Fatalf("remove deleted fixture: %v", err)
	}
	mustWriteSnapshotTestFile(t, root, "dir/new.txt", "created\n")

	if err := snapshot.Restore(); err != nil {
		t.Fatalf("Restore: %v", err)
	}
	assertSnapshotTestFile(t, root, "dir/existing.txt", "before\n")
	assertSnapshotTestFile(t, root, "dir/deleted.txt", "keep\n")
	if _, err := os.Stat(filepath.Join(root, "dir/new.txt")); !os.IsNotExist(err) {
		t.Fatalf("new file still exists after restore; stat err=%v", err)
	}
}

func TestSnapshotRestoresSymlinkWithoutFollowingOutsideRoot(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink privileges are not guaranteed on Windows builders")
	}
	root := t.TempDir()
	outside := t.TempDir()
	outsideFile := filepath.Join(outside, "outside.txt")
	if err := os.WriteFile(outsideFile, []byte("outside-before\n"), 0o644); err != nil {
		t.Fatalf("write outside file: %v", err)
	}
	link := filepath.Join(root, "link.txt")
	if err := os.Symlink(outsideFile, link); err != nil {
		t.Fatalf("create symlink: %v", err)
	}

	snapshot, err := TakeWorkspaceSnapshot(root)
	if err != nil {
		t.Fatalf("TakeWorkspaceSnapshot: %v", err)
	}
	if err := os.WriteFile(link, []byte("outside-after\n"), 0o644); err != nil {
		t.Fatalf("write through symlink: %v", err)
	}
	if err := os.Remove(link); err != nil {
		t.Fatalf("remove link: %v", err)
	}

	if err := snapshot.Restore(); err != nil {
		t.Fatalf("Restore: %v", err)
	}
	target, err := os.Readlink(link)
	if err != nil {
		t.Fatalf("restored link is not a symlink: %v", err)
	}
	if target != outsideFile {
		t.Fatalf("restored link target = %q, want %q", target, outsideFile)
	}
	raw, err := os.ReadFile(outsideFile)
	if err != nil {
		t.Fatalf("read outside file: %v", err)
	}
	if got := string(raw); got != "outside-after\n" {
		t.Fatalf("outside file = %q, want unchanged by snapshot restore", got)
	}
}

func mustWriteSnapshotTestFile(t *testing.T, root, rel, content string) {
	t.Helper()
	path := filepath.Join(root, rel)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", rel, err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", rel, err)
	}
}

func assertSnapshotTestFile(t *testing.T, root, rel, want string) {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(root, rel))
	if err != nil {
		t.Fatalf("read %s: %v", rel, err)
	}
	if got := string(raw); got != want {
		t.Fatalf("%s = %q, want %q", rel, got, want)
	}
}
